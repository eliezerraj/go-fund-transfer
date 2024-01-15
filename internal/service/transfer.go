package service

import (
	"os"
	"time"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"encoding/json"

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
	topic					*core.Topic
}

func NewWorkerService(	workerRepository 	*postgre.WorkerRepository,
						restapi				*restapi.RestApiSConfig,
						producerWorker		*event.ProducerWorker,
						topic				*core.Topic) *WorkerService{
	childLogger.Debug().Msg("NewWorkerService")

	return &WorkerService{
		workerRepository:	workerRepository,
		restapi:			restapi,
		producerWorker: 	producerWorker,
		topic:				topic,
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

func (s WorkerService) Transfer(ctx context.Context, transfer core.Transfer) (interface{}, error){
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

	// Get data from account source credit
	rest_interface_acc_from, err := s.restapi.GetData(ctx, s.restapi.ServerUrlDomain , s.restapi.XApigwId  ,"/fundBalanceAccount", transfer.AccountIDFrom )
	if err != nil {
		return nil, err
	}
	var acc_parsed_from core.AccountBalance

	jsonString, err  := json.Marshal(rest_interface_acc_from)
	if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, errors.New(err.Error())
    }
	json.Unmarshal(jsonString, &acc_parsed_from)

	// Get data from account source debit
	rest_interface_acc_to, err := s.restapi.GetData(ctx, s.restapi.ServerUrlDomain , s.restapi.XApigwId  ,"/fundBalanceAccount", transfer.AccountIDTo )
	if err != nil {
		return nil, err
	}
	var acc_parsed_to core.AccountBalance
	
	jsonString, err = json.Marshal(rest_interface_acc_to)
	if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, errors.New(err.Error())
    }
	json.Unmarshal(jsonString, &acc_parsed_to)

	transfer.FkAccountIDFrom = acc_parsed_from.FkAccountID
	transfer.FkAccountIDTo = acc_parsed_to.FkAccountID

	if (acc_parsed_from.Amount < transfer.Amount) {
		childLogger.Error().Err(err).Msg("error insuficient fund")
		return nil, erro.ErrOverDraft
	}

	childLogger.Debug().Interface("transfer:",transfer).Msg("")
	_, err = s.restapi.PostData(ctx, s.restapi.ServerUrlDomain, s.restapi.XApigwId, "/transferFund", transfer)
	if err != nil {
		return nil, err
	}

	return "sucesso", nil
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

func (s WorkerService) CreditFundSchedule(ctx context.Context, transfer core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("CrediTFundSchedule")

	_, root := xray.BeginSubsegment(ctx, "Service.CreditFundSchedule")

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

	rest_interface_acc_to, err := s.restapi.GetData(ctx, s.restapi.ServerUrlDomain , s.restapi.XApigwId , "/get" ,transfer.AccountIDTo )
	if err != nil {
		return nil, err
	}
	var acc_to_parsed core.Transfer
	err = mapstructure.Decode(rest_interface_acc_to, &acc_to_parsed)
    if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, errors.New(err.Error())
    }

	transfer.FkAccountIDFrom 	= acc_to_parsed.ID
	transfer.FkAccountIDTo 		= acc_to_parsed.ID
	transfer.Type				= "CREDIT"
	transfer.Status				= "CREDIT_EVENT_CREATED"

	res, err := s.workerRepository.Transfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	transfer.ID	= res.ID

	childLogger.Debug().Interface("_X_AMZN_TRACE_ID:",  os.Getenv("X_AMZN_TRACE_ID")).Msg("")

	eventData := core.EventData{&transfer}
	event := core.Event{
		EventDate: time.Now(),
		EventType: s.topic.Credit,
		EventData:	&eventData,	
	}
	
	err = s.producerWorker.Producer(ctx, event)
	if err != nil {
		return nil, err
	}

	transfer.Status	= "CREDIT_SCHEDULE"

	childLogger.Debug().Interface("===>transfer:",transfer).Msg("")

	res_update, err := s.workerRepository.Update(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}
	if res_update == 0 {
		return nil, erro.ErrUpdate
	}

	return res, nil
}

func (s WorkerService) DebitFundSchedule(ctx context.Context, transfer core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("DebitFundSchedule")

	_, root := xray.BeginSubsegment(ctx, "Service.DebitFundSchedule")

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

	rest_interface_acc_to, err := s.restapi.GetData(ctx, s.restapi.ServerUrlDomain , s.restapi.XApigwId , "/get" ,transfer.AccountIDTo )
	if err != nil {
		return nil, err
	}
	var acc_to_parsed core.Transfer
	err = mapstructure.Decode(rest_interface_acc_to, &acc_to_parsed)
    if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, errors.New(err.Error())
    }

	transfer.FkAccountIDFrom 	= acc_to_parsed.ID
	transfer.FkAccountIDTo 		= acc_to_parsed.ID
	transfer.Type				= "DEBIT"
	transfer.Status				= "DEBIT_EVENT_CREATED"

	res, err := s.workerRepository.Transfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	transfer.ID	= res.ID
	
	eventData := core.EventData{&transfer}
	event := core.Event{
		EventDate: time.Now(),
		EventType: s.topic.Dedit,
		EventData:	&eventData,	
	}
	
	err = s.producerWorker.Producer(ctx, event)
	if err != nil {
		return nil, err
	}

	transfer.Status	= "DEBIT_SCHEDULE"

	childLogger.Debug().Interface("===>transfer:",transfer).Msg("")

	res_update, err := s.workerRepository.Update(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}
	if res_update == 0 {
		return nil, erro.ErrUpdate
	}

	return res, nil
}