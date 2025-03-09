package service

import(
	"time"
	"context"
	"net/http"
	"encoding/json"
	"errors"

	"github.com/go-fund-transfer/internal/core/model"
	"github.com/go-fund-transfer/internal/core/erro"
	go_core_observ "github.com/eliezerraj/go-core/observability"
	go_core_api "github.com/eliezerraj/go-core/api"
)

var tracerProvider go_core_observ.TracerProvider
var apiService go_core_api.ApiService

func errorStatusCode(statusCode int) error{
	var err error
	switch statusCode {
	case http.StatusUnauthorized:
		err = erro.ErrUnauthorized
	case http.StatusForbidden:
		err = erro.ErrHTTPForbiden
	case http.StatusNotFound:
		err = erro.ErrNotFound
	default:
		err = erro.ErrServer
	}
	return err
}

func (s WorkerService) AddTransfer(ctx context.Context, transfer *model.Transfer) (*model.Transfer, error){
	childLogger.Debug().Msg("AddTransfer")
	childLogger.Debug().Interface("transfer: ",transfer).Msg("")

	//Trace
	span := tracerProvider.Span(ctx, "service.AddTransfer")

	// Get the database connection
	tx, conn, err := s.workerRepository.DatabasePGServer.StartTx(ctx)
	if err != nil {
		return nil, err
	}
	
	// Handle the transaction
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
		s.workerRepository.DatabasePGServer.ReleaseTx(conn)
		span.End()
	}()
	
	// Get transaction UUID 
	res_uuid, err := s.workerRepository.GetTransactionUUID(ctx)
	if err != nil {
		return nil, err
	}

	// Business rule
	if (transfer.Type != "TRANSFER") {
		err = erro.ErrTransInvalid 
		return nil, err
	}

	time_chargeAt := time.Now()
	transfer.AccountFrom.Currency = transfer.Currency
	transfer.AccountFrom.TransactionID = res_uuid
	transfer.AccountFrom.Amount = (transfer.Amount * -1)
	transfer.AccountFrom.Type = "DEBIT"
	transfer.AccountFrom.ChargeAt = time_chargeAt

	transfer.AccountTo.Currency = transfer.Currency
	transfer.AccountTo.Amount = transfer.Amount
	transfer.AccountTo.TransactionID = res_uuid
	transfer.AccountTo.Type = "CREDIT"
	transfer.AccountTo.ChargeAt = time_chargeAt

	transfer.TransactionID = res_uuid
	transfer.Status = "TRANSFER-REST-DONE"
	transfer.TransferAt = time_chargeAt

	// Get the Account ID from Account-service
	res_acc_from, statusCode, err := apiService.CallApi(ctx,
														s.apiService[0].Url + "/" + transfer.AccountFrom.AccountID,
														s.apiService[0].Method,
														&s.apiService[0].Header_x_apigw_api_id,
														nil, 
														nil)
	if err != nil {
		return nil, errorStatusCode(statusCode)
	}
	jsonString, err := json.Marshal(res_acc_from)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	var account model.AccountStatement
	json.Unmarshal(jsonString, &account)

	transfer.AccountFrom.FkAccountID = account.ID
	
	// Get the Account ID from Account-service
	res_acc_to, statusCode, err := apiService.CallApi(ctx,
														s.apiService[0].Url + "/" + transfer.AccountTo.AccountID,
														s.apiService[0].Method,
														&s.apiService[0].Header_x_apigw_api_id,
														nil, 
														nil)
	if err != nil {
		return nil, errorStatusCode(statusCode)
	}
	jsonString, err = json.Marshal(res_acc_to)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	json.Unmarshal(jsonString, &account)

	transfer.AccountTo.FkAccountID = account.ID

	// Add (POST) the account statement Get the Account ID from Account-service
	_, statusCode, err = apiService.CallApi(ctx,
											s.apiService[1].Url,
											s.apiService[1].Method,
											&s.apiService[1].Header_x_apigw_api_id,
											nil, 
											transfer.AccountFrom)
	if err != nil {
		return nil, errorStatusCode(statusCode)
	}

	// Add (POST) the account statement Get the Account ID from Account-service
	_, statusCode, err = apiService.CallApi(ctx,
											s.apiService[2].Url,
											s.apiService[2].Method,
											&s.apiService[2].Header_x_apigw_api_id,
											nil, 
											transfer.AccountTo)
	if err != nil {
		return nil, errorStatusCode(statusCode)
	}

	// Add transfer
	res_transfer, err := s.workerRepository.AddTransfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	transfer.ID = res_transfer.ID 

	return transfer, nil
}

func (s *WorkerService) GetTransfer(ctx context.Context, transfer *model.Transfer) (*model.Transfer, error){
	childLogger.Debug().Msg("GetTransfer")
	childLogger.Debug().Interface("transfer: ", transfer).Msg("")

	// Trace
	span := tracerProvider.Span(ctx, "service.GetTransfer")
	defer span.End()
	
	// Get transfer
	res, err := s.workerRepository.GetTransfer(ctx, transfer)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *WorkerService) CreditTransferEvent(ctx context.Context, transfer *model.Transfer) (*model.Transfer, error){
	childLogger.Debug().Msg("CreditTransferEvent")
	childLogger.Debug().Interface("transfer: ", transfer).Msg("")

	// Trace
	span := tracerProvider.Span(ctx, "service.CreditTransferEvent")
	defer span.End()

	// Get the database connection
	tx, conn, err := s.workerRepository.DatabasePGServer.StartTx(ctx)
	if err != nil {
		return nil, err
	}
	
	// Start Kafka transaction
	err = s.workerEvent.WorkerKafka.BeginTransaction()
	if err != nil {
		childLogger.Error().Err(err).Msg("failed to kafka BeginTransaction")
		return nil, err
	}

	// Handle the transaction
	defer func() {
		if err != nil {
			childLogger.Debug().Msg("ROLLBACK !!!!")
			err :=  s.workerEvent.WorkerKafka.AbortTransaction(ctx)
			if err != nil {
				childLogger.Error().Err(err).Msg("Failed to Kafka AbortTransaction")
			}		
			tx.Rollback(ctx)
		} else {
			err =  s.workerEvent.WorkerKafka.CommitTransaction(ctx)
			if err != nil {
				childLogger.Error().Err(err).Msg("Failed to Kafka CommitTransaction")
			}
			tx.Commit(ctx)
		}
		s.workerRepository.DatabasePGServer.ReleaseTx(conn)
		span.End()
	}()
	
	// Get transaction UUID 
	res_uuid, err := s.workerRepository.GetTransactionUUID(ctx)
	if err != nil {
		return nil, err
	}

	// Get the Account ID from Account-service
	res_acc_from, statusCode, err := apiService.CallApi(ctx,
														s.apiService[0].Url + "/" + transfer.AccountFrom.AccountID,
														s.apiService[0].Method,
														&s.apiService[0].Header_x_apigw_api_id,
														nil, 
														nil)
	if err != nil {
		return nil, errorStatusCode(statusCode)
	}
	jsonString, err := json.Marshal(res_acc_from)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	var accountStatement model.AccountStatement
	json.Unmarshal(jsonString, &accountStatement)

	// Businness rule
	if transfer.Amount < 0 {
		return nil, erro.ErrAmountInvalid
	}
	time_chargeAt := time.Now()
	transfer.AccountFrom.FkAccountID = accountStatement.ID
	transfer.AccountTo = transfer.AccountFrom // From and To are the same in case of Credit
	transfer.AccountFrom.Currency = transfer.Currency
	transfer.AccountFrom.Amount = transfer.Amount
	transfer.AccountFrom.TransactionID = res_uuid
	transfer.AccountFrom.ChargeAt = time_chargeAt
	transfer.AccountFrom.Type = "CREDIT"

	transfer.Status			= "CREDIT_EVENT_CREATED"
	transfer.TransactionID = res_uuid
	transfer.TransferAt = time_chargeAt

	// Add transfer
	res_transfer, err := s.workerRepository.AddTransfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	// Prepare to event credit
	key := string(res_transfer.ID)
	payload_bytes, err := json.Marshal(res_transfer)
	if err != nil {
		return nil, err
	}

	// publish event credit
	childSpanKafka := tracerProvider.Span(ctx, "workerKafka.Producer")
	err = s.workerEvent.WorkerKafka.Producer(ctx, s.workerEvent.Topics[0], key, payload_bytes)
	if err != nil {
		return nil, err
	}
	defer childSpanKafka.End()

	// Just for testing (breaking) the transaction and testing kafka
	if transfer.Currency == "USD" {
		err =  erro.ErrCurrencyInvalid
		return nil, err
	}

	return res_transfer, nil
}

func (s *WorkerService) DebitTransferEvent(ctx context.Context, transfer *model.Transfer) (*model.Transfer, error){
	childLogger.Debug().Msg("DebitTransferEvent")
	childLogger.Debug().Interface("transfer: ", transfer).Msg("")

	// Trace
	span := tracerProvider.Span(ctx, "service.DebitTransferEvent")
	defer span.End()

	// Get the database connection
	tx, conn, err := s.workerRepository.DatabasePGServer.StartTx(ctx)
	if err != nil {
		return nil, err
	}

	// Start Kafka transaction
	err = s.workerEvent.WorkerKafka.BeginTransaction()
	if err != nil {
		childLogger.Error().Err(err).Msg("failed to kafka BeginTransaction")
		return nil, err
	}

	// Handle the transaction
	defer func() {
		if err != nil {
			childLogger.Debug().Msg("ROLLBACK !!!!")
			err :=  s.workerEvent.WorkerKafka.AbortTransaction(ctx)
			if err != nil {
				childLogger.Error().Err(err).Msg("Failed to Kafka AbortTransaction")
			}		
			tx.Rollback(ctx)
		} else {
			err =  s.workerEvent.WorkerKafka.CommitTransaction(ctx)
			if err != nil {
				childLogger.Error().Err(err).Msg("Failed to Kafka CommitTransaction")
			}
			tx.Commit(ctx)
		}
		s.workerRepository.DatabasePGServer.ReleaseTx(conn)
		span.End()
	}()
	
	// Get transaction UUID 
	res_uuid, err := s.workerRepository.GetTransactionUUID(ctx)
	if err != nil {
		return nil, err
	}

	// Get the Account ID from Account-service
	res_acc_from, statusCode, err := apiService.CallApi(ctx,
														s.apiService[0].Url + "/" + transfer.AccountFrom.AccountID,
														s.apiService[0].Method,
														&s.apiService[0].Header_x_apigw_api_id,
														nil, 
														nil)
	if err != nil {
		return nil, errorStatusCode(statusCode)
	}
	jsonString, err := json.Marshal(res_acc_from)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	var accountStatement model.AccountStatement
	json.Unmarshal(jsonString, &accountStatement)

	// Businness rule
	if transfer.Amount > 0 {
		return nil, erro.ErrAmountInvalid
	}

	time_chargeAt := time.Now()
	transfer.AccountFrom.FkAccountID = accountStatement.ID
	transfer.AccountTo = transfer.AccountFrom // From and To are the same in case of Credit
	transfer.AccountFrom.Currency = transfer.Currency
	transfer.AccountFrom.Amount = transfer.Amount
	transfer.AccountFrom.TransactionID = res_uuid
	transfer.AccountFrom.ChargeAt = time_chargeAt
	transfer.AccountFrom.Type = "DEBIT"

	transfer.Status			= "DEBIT_EVENT_CREATED"
	transfer.TransactionID = res_uuid
	transfer.TransferAt = time_chargeAt

	// Add transfer
	res_transfer, err := s.workerRepository.AddTransfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	// Prepare to event debit
	key := string(res_transfer.ID)
	payload_bytes, err := json.Marshal(res_transfer)
	if err != nil {
		return nil, err
	}

	// publish event debit
	childSpanKafka := tracerProvider.Span(ctx, "workerKafka.Producer")
	err = s.workerEvent.WorkerKafka.Producer(ctx, s.workerEvent.Topics[1], key, payload_bytes)
	if err != nil {
		return nil, err
	}
	defer childSpanKafka.End()

	return res_transfer, nil
}

func (s *WorkerService) AddTransferEvent(ctx context.Context, transfer *model.Transfer) (*model.Transfer, error){
	childLogger.Debug().Msg("AddTransferEvent")
	childLogger.Debug().Interface("transfer: ", transfer).Msg("")

	// Trace
	span := tracerProvider.Span(ctx, "service.AddTransferEvent")
	defer span.End()

	// Get the database connection
	tx, conn, err := s.workerRepository.DatabasePGServer.StartTx(ctx)
	if err != nil {
		return nil, err
	}

	// Start Kafka transaction
	err = s.workerEvent.WorkerKafka.BeginTransaction()
	if err != nil {
		childLogger.Error().Err(err).Msg("failed to kafka BeginTransaction")
		return nil, err
	}

	// Handle the transaction
	defer func() {
		if err != nil {
			childLogger.Debug().Msg("ROLLBACK !!!!")
			err :=  s.workerEvent.WorkerKafka.AbortTransaction(ctx)
			if err != nil {
				childLogger.Error().Err(err).Msg("Failed to Kafka AbortTransaction")
			}		
			tx.Rollback(ctx)
		} else {
			err =  s.workerEvent.WorkerKafka.CommitTransaction(ctx)
			if err != nil {
				childLogger.Error().Err(err).Msg("Failed to Kafka CommitTransaction")
			}
			tx.Commit(ctx)
		}
		s.workerRepository.DatabasePGServer.ReleaseTx(conn)
		span.End()
	}()
	
	// Get transaction UUID 
	res_uuid, err := s.workerRepository.GetTransactionUUID(ctx)
	if err != nil {
		return nil, err
	}

	// Business rule
	if (transfer.Type != "TRANSFER") {
		err = erro.ErrTransInvalid 
		return nil, err
	}

	time_chargeAt := time.Now()
	transfer.AccountFrom.Currency = transfer.Currency
	transfer.AccountFrom.TransactionID = res_uuid
	transfer.AccountFrom.Amount = (transfer.Amount * -1)
	transfer.AccountFrom.Type = "DEBIT"
	transfer.AccountFrom.ChargeAt = time_chargeAt

	transfer.AccountTo.Currency = transfer.Currency
	transfer.AccountTo.Amount = transfer.Amount
	transfer.AccountTo.TransactionID = res_uuid
	transfer.AccountTo.Type = "CREDIT"
	transfer.AccountTo.ChargeAt = time_chargeAt

	transfer.TransactionID = res_uuid
	transfer.Status = "TRANSFER-EVENT-CREATED"
	transfer.TransferAt = time_chargeAt

	// Get the Account ID from Account-service
	res_acc_from, statusCode, err := apiService.CallApi(ctx,
														s.apiService[0].Url + "/" + transfer.AccountFrom.AccountID,
														s.apiService[0].Method,
														&s.apiService[0].Header_x_apigw_api_id,
														nil, 
														nil)
	if err != nil {
		return nil, errorStatusCode(statusCode)
	}
	jsonString, err := json.Marshal(res_acc_from)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	var accountStatement model.AccountStatement
	json.Unmarshal(jsonString, &accountStatement)

	transfer.AccountFrom.FkAccountID = accountStatement.ID

	// Get the Account ID from Account-service
	res_acc_to, statusCode, err := apiService.CallApi(ctx,
														s.apiService[0].Url + "/" + transfer.AccountTo.AccountID,
														s.apiService[0].Method,
														&s.apiService[0].Header_x_apigw_api_id,
														nil, 
														nil)
	if err != nil {
		return nil, errorStatusCode(statusCode)
	}
	jsonString, err = json.Marshal(res_acc_to)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	json.Unmarshal(jsonString, &accountStatement)

	transfer.AccountTo.FkAccountID = accountStatement.ID

	// Add transfer
	res_transfer, err := s.workerRepository.AddTransfer(ctx, tx, transfer)
	if err != nil {
		return nil, err
	}

	// Prepare to event transfer
	key := string(res_transfer.ID)
	payload_bytes, err := json.Marshal(transfer)
	if err != nil {
		return nil, err
	}

	// publish event transfer
	childSpanKafka := tracerProvider.Span(ctx, "workerKafka.Producer")
	err = s.workerEvent.WorkerKafka.Producer(ctx, s.workerEvent.Topics[2], key, payload_bytes)
	if err != nil {
		return nil, err
	}
	defer childSpanKafka.End()

	return res_transfer, nil
}