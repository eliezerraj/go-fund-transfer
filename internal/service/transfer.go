package service

import (
	"time"
	"context"
	"errors"
	"github.com/rs/zerolog/log"

	"github.com/mitchellh/mapstructure"
	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/erro"
	"github.com/go-fund-transfer/internal/adapter/restapi"
	"github.com/go-fund-transfer/internal/repository/postgre"
	"github.com/go-fund-transfer/internal/adapter/event"
	"github.com/aws/aws-xray-sdk-go/xray"

)

var childLogger = log.With().Str("service", "service").Logger()

type WorkerService struct {
	workerRepository 		*postgre.WorkerRepository
	restapi					*restapi.RestApiSConfig
	producerWorker			*event.ProducerWorker
}

func NewWorkerService(	workerRepository 	*postgre.WorkerRepository,
						restapi				*restapi.RestApiSConfig,
						producerWorker		*event.ProducerWorker) *WorkerService{
	childLogger.Debug().Msg("NewWorkerService")

	return &WorkerService{
		workerRepository:	workerRepository,
		restapi:			restapi,
		producerWorker: 	producerWorker,
	}
}

func (s WorkerService) SetSessionVariable(ctx context.Context, userCredential string) (bool, error){
	childLogger.Debug().Msg("SetSessionVariable")

	res, err := s.workerRepository.SetSessionVariable(ctx, userCredential)
	if err != nil {
		return false, err
	}

	return res, nil
}

func (s WorkerService) Transfer(ctx context.Context, transfer core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("TransferFund")

	_, root := xray.BeginSubsegment(ctx, "Service.TransferFund")

	tx, err := s.workerRepository.StartTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
		root.Close(nil)
	}()

	childLogger.Debug().Interface("transfer:",transfer).Msg("")

	rest_interface_acc_from, err := s.restapi.GetData(ctx, transfer.AccountIDFrom, "/get")
	if err != nil {
		return nil, err
	}
	var acc_from_parsed core.Transfer
	err = mapstructure.Decode(rest_interface_acc_from, &acc_from_parsed)
    if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, errors.New(err.Error())
    }

	rest_interface_acc_to, err := s.restapi.GetData(ctx, transfer.AccountIDTo, "/get")
	if err != nil {
		return nil, err
	}
	var acc_to_parsed core.Transfer
	err = mapstructure.Decode(rest_interface_acc_to, &acc_to_parsed)
    if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, errors.New(err.Error())
    }

	transfer.FkAccountIDFrom 	= acc_from_parsed.ID
	transfer.FkAccountIDTo 		= acc_to_parsed.ID
	transfer.Status				= "CREATED"

	res, err := s.workerRepository.Transfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	eventData := core.EventData{&transfer}
	event := core.Event{
		EventDate: time.Now(),
		EventType: "topic.transfer",
		EventData:	&eventData,	
	}
	
	err = s.producerWorker.Producer(ctx, event)
	if err != nil {
		return nil, err
	}

	transfer.Status	= "SEND"
	transfer.ID	= res.ID
	res_update, err := s.workerRepository.Update(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}
	if res_update == 0 {
		return nil, erro.ErrUpdate
	}

	return res, nil
}

func (s WorkerService) Get(ctx context.Context, transfer core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("Get")
	childLogger.Debug().Interface("transfer:",transfer).Msg("")
	
	_, root := xray.BeginSubsegment(ctx, "Service.transfer")
	defer root.Close(nil)

	res, err := s.workerRepository.Get(ctx, transfer)
	if err != nil {
		return nil, err
	}

	return res, nil
}