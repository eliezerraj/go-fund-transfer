package service

import (
	"time"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"encoding/json"

	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/erro"
	"github.com/go-fund-transfer/internal/adapter/restapi"
	"github.com/go-fund-transfer/internal/repository/storage"
	"github.com/go-fund-transfer/internal/adapter/event"
	"github.com/go-fund-transfer/internal/lib"
	"github.com/go-fund-transfer/internal/handler/listener"
)

var childLogger = log.With().Str("service", "service").Logger()
var restApiCallData core.RestApiCallData

type WorkerService struct {
	workerRepo		 		*storage.WorkerRepository
	appServer				*core.AppServer
	restApiService			*restapi.RestApiService
	producerWorker			event.EventNotifier
	topic					*core.Topic
	tokenRefresher			*listener.TokenRefresh			
}

func NewWorkerService(	workerRepo		*storage.WorkerRepository,
						appServer		*core.AppServer,
						restApiService	*restapi.RestApiService,
						eventNotifier	event.EventNotifier,
						topic			*core.Topic,
						tokenRefresher	*listener.TokenRefresh) *WorkerService{
	childLogger.Debug().Msg("NewWorkerService")

	return &WorkerService{
		workerRepo: 		workerRepo,
		appServer:			appServer,
		restApiService:		restApiService,
		producerWorker: 	eventNotifier,
		topic:				topic,
		tokenRefresher:		tokenRefresher,
	}
}

func (s WorkerService) Transfer(ctx context.Context, transfer core.Transfer) (interface{}, error){
	childLogger.Debug().Msg("TransferFund")
	childLogger.Debug().Interface(" -----------------------------> transfer:",transfer).Msg("")

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

	// Get data from account source credit
	restApiCallData.Method = "GET"
	restApiCallData.Url = s.appServer.RestEndpoint.ServiceUrlDomain + "/fundBalanceAccount/" + transfer.AccountIDFrom
	restApiCallData.X_Api_Id = &s.appServer.RestEndpoint.XApigwId

	rest_interface_acc_from, err := s.restApiService.CallApiRest(ctx, restApiCallData, nil)
	if err != nil {
		childLogger.Error().Err(err).Msg("error CallApiRest /fundBalanceAccount")
		return nil, err
	}
	jsonString, err  := json.Marshal(rest_interface_acc_from)
	if err != nil {
		childLogger.Error().Err(err).Msg("error Marshal")
		return nil, errors.New(err.Error())
    }
	var acc_parsed_from core.AccountBalance
	json.Unmarshal(jsonString, &acc_parsed_from)

	childLogger.Debug().Interface(" ##################### >>>>>>>> acc_parsed_from: ",acc_parsed_from).Msg("")

	// Get data from account source debit
	restApiCallData.Method = "GET"
	restApiCallData.Url = s.appServer.RestEndpoint.ServiceUrlDomain + "/fundBalanceAccount/" + transfer.AccountIDTo
	restApiCallData.X_Api_Id = &s.appServer.RestEndpoint.XApigwId

	rest_interface_acc_to, err := s.restApiService.CallApiRest(ctx, restApiCallData, nil)
	if err != nil {
		childLogger.Error().Err(err).Msg("error CallApiRest /fundBalanceAccount")
		return nil, err
	}
	jsonString, err = json.Marshal(rest_interface_acc_to)
	if err != nil {
		childLogger.Error().Err(err).Msg("error Marshal")
		return nil, errors.New(err.Error())
    }
	var acc_parsed_to core.AccountBalance
	json.Unmarshal(jsonString, &acc_parsed_to)

	childLogger.Debug().Interface(" ##################### >>>>>>>> acc_parsed_to: ",acc_parsed_to).Msg("")

	transfer.FkAccountIDFrom = acc_parsed_from.FkAccountID
	transfer.FkAccountIDTo = acc_parsed_to.FkAccountID

	if (acc_parsed_from.Amount < transfer.Amount) {
		childLogger.Error().Err(err).Msg("error insuficient fund")
		return nil, erro.ErrOverDraft
	}

	childLogger.Debug().Interface(" ++++++++++++++ >>>>>>>> transfer: ",transfer).Msg("")

	restApiCallData.Method = "POST"
	restApiCallData.Url = s.appServer.RestEndpoint.ServiceUrlDomain + "/transferFund"
	restApiCallData.X_Api_Id = &s.appServer.RestEndpoint.XApigwId

	_, err = s.restApiService.CallApiRest(ctx, restApiCallData, transfer)
	if err != nil {
		childLogger.Error().Err(err).Msg("error CallApiRest /transferFund")
		return nil, err
	}

	return transfer, nil
}

func (s WorkerService) Get(ctx context.Context, transfer *core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("Get")
	
	span := lib.Span(ctx, "service.Get")
	defer span.End()

	s.tokenRefresher.GetToken()

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
	restApiCallData.Method = "GET"
	restApiCallData.Url = s.appServer.RestEndpoint.ServiceUrlDomain + "/get/" + transfer.AccountIDTo
	restApiCallData.X_Api_Id = &s.appServer.RestEndpoint.XApigwId

	res_interface, err := s.restApiService.CallApiRest(ctx, restApiCallData, nil)
	if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, err
	}
	jsonString, err  := json.Marshal(res_interface)
	if err != nil {
		childLogger.Error().Err(err).Msg("error Marshal")
		return nil, errors.New(err.Error())
    }
	var acc_to_parsed core.Transfer
	json.Unmarshal(jsonString, &acc_to_parsed)

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
	eventData := core.EventData{Transfer: transfer}
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
	restApiCallData.Method = "GET"
	restApiCallData.Url = s.appServer.RestEndpoint.ServiceUrlDomain + "/get/" + transfer.AccountIDTo
	restApiCallData.X_Api_Id = &s.appServer.RestEndpoint.XApigwId

	res_interface, err := s.restApiService.CallApiRest(ctx, restApiCallData, nil)
	if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, err
	}
	jsonString, err  := json.Marshal(res_interface)
	if err != nil {
		childLogger.Error().Err(err).Msg("error Marshal")
		return nil, errors.New(err.Error())
    }
	var acc_to_parsed core.Transfer
	json.Unmarshal(jsonString, &acc_to_parsed)

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
	eventData := core.EventData{Transfer: transfer}
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
	restApiCallData.Method = "GET"
	restApiCallData.Url = s.appServer.RestEndpoint.ServiceUrlDomain + "/fundBalanceAccount/" + transfer.AccountIDFrom
	restApiCallData.X_Api_Id = &s.appServer.RestEndpoint.XApigwId

	rest_interface_acc_from, err := s.restApiService.CallApiRest(ctx, restApiCallData, nil)
	if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, err
	}
	jsonString, err  := json.Marshal(rest_interface_acc_from)
	if err != nil {
		childLogger.Error().Err(err).Msg("error Marshal")
		return nil, errors.New(err.Error())
    }
	var acc_parsed_from core.AccountBalance
	json.Unmarshal(jsonString, &acc_parsed_from)

	// Get data from account source debit
	restApiCallData.Method = "GET"
	restApiCallData.Url = s.appServer.RestEndpoint.ServiceUrlDomain + "/fundBalanceAccount/" + transfer.AccountIDTo
	restApiCallData.X_Api_Id = &s.appServer.RestEndpoint.XApigwId

	rest_interface_acc_to, err := s.restApiService.CallApiRest(ctx, restApiCallData, nil)
	if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
		return nil, err
	}
	jsonString, err = json.Marshal(rest_interface_acc_to)
	if err != nil {
		childLogger.Error().Err(err).Msg("error Marshal")
		return nil, errors.New(err.Error())
    }
	var acc_parsed_to core.AccountBalance
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
	eventData := core.EventData{Transfer: transfer}
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