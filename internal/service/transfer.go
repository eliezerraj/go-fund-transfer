package service

import (
	"time"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"encoding/json"

	"github.com/mitchellh/mapstructure"
	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/erro"
	"github.com/go-fund-transfer/internal/adapter/restapi"
	"github.com/go-fund-transfer/internal/repository/storage"
	"github.com/go-fund-transfer/internal/adapter/event"
	"github.com/go-fund-transfer/internal/lib"
)

var childLogger = log.With().Str("service", "service").Logger()

type WorkerService struct {
	workerRepo		 		*storage.WorkerRepository
	appServer				*core.AppServer
	restApiService			*restapi.RestApiService
	producerWorker			*event.ProducerWorker
	topic					*core.Topic
}

func NewWorkerService(	workerRepo		*storage.WorkerRepository,
						appServer		*core.AppServer,
						restApiService	*restapi.RestApiService,
						producerWorker	*event.ProducerWorker,
						topic			*core.Topic) *WorkerService{
	childLogger.Debug().Msg("NewWorkerService")

	return &WorkerService{
		workerRepo: 		workerRepo,
		appServer:			appServer,
		restApiService:		restApiService,
		producerWorker: 	producerWorker,
		topic:				topic,
	}
}

func (s WorkerService) SetSessionVariable(	ctx context.Context, 
											userCredential string) (bool, error){
	childLogger.Debug().Msg("SetSessionVariable")

	res, err := s.workerRepo.SetSessionVariable(ctx, userCredential)
	if err != nil {
		return false, err
	}

	return res, nil
}

func (s WorkerService) Transfer(ctx context.Context, transfer core.Transfer) (interface{}, error){
	childLogger.Debug().Msg("TransferFund")

	span := lib.Span(ctx, "service.Transfer")	

	tx, conn, err := s.workerRepo.StartTx(ctx)
	if err != nil {
		return nil, err
	}
	
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
		s.workerRepo.ReleaseTx(conn)
		span.End()
	}()

	childLogger.Debug().Interface("transfer:",transfer).Msg("")

	// Get data from account source credit
	path := s.appServer.RestEndpoint.ServiceUrlDomain + "/fundBalanceAccount/" + transfer.AccountIDFrom
	rest_interface_acc_from, err := s.restApiService.CallRestApi(ctx,"GET", path, &s.appServer.RestEndpoint.XApigwId, nil)
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
	path = s.appServer.RestEndpoint.ServiceUrlDomain + "/fundBalanceAccount/" + transfer.AccountIDTo
	rest_interface_acc_to, err := s.restApiService.CallRestApi(ctx,"GET", path, &s.appServer.RestEndpoint.XApigwId, nil)
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

	path = s.appServer.RestEndpoint.ServiceUrlDomain + "/transferFund"
	_, err = s.restApiService.CallRestApi(ctx,"POST",path, &s.appServer.RestEndpoint.XApigwId,transfer)
	if err != nil {
		return nil, err
	}

	return "sucesso", nil
}

func (s WorkerService) Get(ctx context.Context, transfer *core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("Get")
	childLogger.Debug().Interface("transfer:",transfer).Msg("")
	
	span := lib.Span(ctx, "service.Get")
	defer span.End()

	res, err := s.workerRepo.Get(ctx, transfer)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s WorkerService) CreditFundSchedule(ctx context.Context, transfer *core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("CrediTFundSchedule")

	span := lib.Span(ctx, "service.CreditFundSchedule")

	tx, conn, err := s.workerRepo.StartTx(ctx)
	if err != nil {
		return nil, err
	}
	
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
		s.workerRepo.ReleaseTx(conn)
		span.End()
	}()

	// Get account data
	path := s.appServer.RestEndpoint.ServiceUrlDomain + "/get/" + transfer.AccountIDTo
	rest_interface_acc_to, err := s.restApiService.CallRestApi(ctx,"GET", path, &s.appServer.RestEndpoint.XApigwId, nil)
	if err != nil {
		return nil, err
	}
	var acc_to_parsed core.Transfer
	err = mapstructure.Decode(rest_interface_acc_to, &acc_to_parsed)
    if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, errors.New(err.Error())
    }

	// Register the moviment into table transfer_moviment (work as a history)
	transfer.FkAccountIDFrom 	= acc_to_parsed.ID
	transfer.FkAccountIDTo 		= acc_to_parsed.ID
	transfer.Status				= "CREDIT_EVENT_CREATED"

	res, err := s.workerRepo.Transfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	// Send data to Kafka
	transfer.ID	= res.ID
	transfer.TransferAt = res.TransferAt
	eventData := core.EventData{transfer}
	event := core.Event{
		Key: transfer.AccountIDTo,
		EventDate: time.Now(),
		EventType: s.topic.Credit,
		EventData:	&eventData,	
	}
	
	err = s.producerWorker.Producer(ctx, event)
	if err != nil {
		return nil, err
	}

	// If send was OK, set the transfer_moviment to CREDIT_SCHEDULE
	transfer.Status	= "CREDIT_SCHEDULE"
	childLogger.Debug().Interface("===>transfer:",transfer).Msg("")

	res_update, err := s.workerRepo.Update(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}
	if res_update == 0 {
		err = erro.ErrUpdate
		return nil, err
	}

	return transfer, nil
}

func (s WorkerService) DebitFundSchedule(ctx context.Context, transfer *core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("DebitFundSchedule")

	span := lib.Span(ctx, "service.DebitFundSchedule")

	tx, conn, err := s.workerRepo.StartTx(ctx)
	if err != nil {
		return nil, err
	}
	
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
		s.workerRepo.ReleaseTx(conn)
		span.End()
	}()

	// Get account data
	path := s.appServer.RestEndpoint.ServiceUrlDomain + "/get/" + transfer.AccountIDTo
	rest_interface_acc_to, err := s.restApiService.CallRestApi(ctx,"GET", path, &s.appServer.RestEndpoint.XApigwId, nil)
	if err != nil {
		return nil, err
	}
	var acc_to_parsed core.Transfer
	err = mapstructure.Decode(rest_interface_acc_to, &acc_to_parsed)
    if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, errors.New(err.Error())
    }

	// Register the moviment into table transfer_moviment (work as a history)
	transfer.FkAccountIDFrom 	= acc_to_parsed.ID
	transfer.FkAccountIDTo 		= acc_to_parsed.ID
	transfer.Status				= "DEBIT_EVENT_CREATED"

	res, err := s.workerRepo.Transfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	// Send data to Kafka
	transfer.ID	= res.ID
	transfer.TransferAt = res.TransferAt
	eventData := core.EventData{transfer}
	event := core.Event{
		Key: transfer.AccountIDTo,
		EventDate: time.Now(),
		EventType: s.topic.Dedit,
		EventData:	&eventData,	
	}
	
	err = s.producerWorker.Producer(ctx, event)
	if err != nil {
		return nil, err
	}

	// If send was OK, set the transfer_moviment to DEBIT_SCHEDULE
	transfer.Status	= "DEBIT_SCHEDULE"
	childLogger.Debug().Interface("===>transfer:",transfer).Msg("")

	res_update, err := s.workerRepo.Update(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}
	if res_update == 0 {
		err = erro.ErrUpdate
		return nil, err
	}

	return transfer, nil
}

func (s WorkerService) TransferViaEvent(ctx context.Context, transfer *core.Transfer) (interface{}, error){
	childLogger.Debug().Msg("TransferViaEvent")
	childLogger.Debug().Interface("transfer:",transfer).Msg("")

	span := lib.Span(ctx, "service.TransferViaEvent")

	tx, conn, err := s.workerRepo.StartTx(ctx)
	if err != nil {
		return nil, err
	}
	
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
		s.workerRepo.ReleaseTx(conn)
		span.End()
	}()

	// Get data from account source credit
	path := s.appServer.RestEndpoint.ServiceUrlDomain + "/fundBalanceAccount/" + transfer.AccountIDFrom
	rest_interface_acc_from, err := s.restApiService.CallRestApi(ctx,"GET", path, &s.appServer.RestEndpoint.XApigwId, nil)
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
	path = s.appServer.RestEndpoint.ServiceUrlDomain + "/fundBalanceAccount/" + transfer.AccountIDTo
	rest_interface_acc_to, err := s.restApiService.CallRestApi(ctx,"GET", path, &s.appServer.RestEndpoint.XApigwId, nil)
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

	// Register the moviment into table transfer_moviment (work as a history)
	transfer.FkAccountIDFrom 	= acc_parsed_from.FkAccountID
	transfer.FkAccountIDTo 		= acc_parsed_to.FkAccountID
	transfer.Status				= "TRANSFER_EVENT_CREATED"

	res, err := s.workerRepo.Transfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	// Send data to Kafka
	transfer.ID	= res.ID
	eventData := core.EventData{transfer}
	event := core.Event{
		Key: transfer.AccountIDFrom + ":" + transfer.AccountIDTo,
		EventDate: time.Now(),
		EventType: s.topic.Transfer,
		EventData:	&eventData,	
	}
	
	err = s.producerWorker.Producer(ctx, event)
	if err != nil {
		return nil, err
	}

	// If send was OK, set the transfer_moviment to TRANSFER_SCHEDULE
	transfer.Status	= "TRANSFER_SCHEDULE"
	childLogger.Debug().Interface("===>transfer:",transfer).Msg("")

	res_update, err := s.workerRepo.Update(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}
	if res_update == 0 {
		err = erro.ErrUpdate
		return nil, err
	}

	return transfer, nil
}