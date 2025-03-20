package database

import (
	"context"
	"errors"
	"time"

	"github.com/go-fund-transfer/internal/core/model"
	"github.com/go-fund-transfer/internal/core/erro"

	go_core_observ "github.com/eliezerraj/go-core/observability"
	go_core_pg "github.com/eliezerraj/go-core/database/pg"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

var tracerProvider go_core_observ.TracerProvider
var childLogger = log.With().Str("adapter", "database").Logger()

type WorkerRepository struct {
	DatabasePGServer *go_core_pg.DatabasePGServer
}

func NewWorkerRepository(databasePGServer *go_core_pg.DatabasePGServer) *WorkerRepository{
	childLogger.Info().Msg("NewWorkerRepository")

	return &WorkerRepository{
		DatabasePGServer: databasePGServer,
	}
}

// About create a uuid transaction
func (w WorkerRepository) GetTransactionUUID(ctx context.Context) (*string, error){
	childLogger.Info().Interface("trace-resquest-id", ctx.Value("trace-request-id")).Msg("GetTransactionUUID")
	
	// Trace
	span := tracerProvider.Span(ctx, "database.GetTransactionUUID")
	defer span.End()

	// get DB connection
	conn, err := w.DatabasePGServer.Acquire(ctx)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer w.DatabasePGServer.Release(conn)

	// Prepare
	var uuid string

	// Query and Execute
	query := `SELECT uuid_generate_v4()`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&uuid) 
		if err != nil {
			return nil, errors.New(err.Error())
        }
		return &uuid, nil
	}
	
	return &uuid, nil
}

// About add a transfer transaction
func (w WorkerRepository) AddTransfer(ctx context.Context, tx pgx.Tx, transfer *model.Transfer) (*model.Transfer, error){
	childLogger.Info().Interface("trace-resquest-id", ctx.Value("trace-request-id")).Msg("AddTransfer")
	childLogger.Info().Interface("trace-resquest-id", ctx.Value("trace-request-id")).Interface("transfer: ",transfer).Msg("")

	// Trace
	span := tracerProvider.Span(ctx, "database.AddTransfer")
	defer span.End()

	// Prepare
	var id int
	transfer.TransferAt = time.Now()

	// Query and Execute
	query := `INSERT INTO transfer_moviment(fk_account_id_from, 
											fk_account_id_to,
											type_charge,
											status,  
											transfer_at,
											currency,
											amount,
											transaction_id) 
				VALUES($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`

	row := tx.QueryRow(ctx, query,	transfer.AccountFrom.FkAccountID, 
									transfer.AccountTo.FkAccountID,
									transfer.Type,
									transfer.Status,
									transfer.TransferAt,
									transfer.Currency,
									transfer.Amount,
									transfer.TransactionID)				

	if err := row.Scan(&id); err != nil {
		return nil, errors.New(err.Error())
	}

	// Set PK
	transfer.ID = id
	return transfer , nil
}

// About add transfer transaction
func (w WorkerRepository) GetTransfer(ctx context.Context, transfer *model.Transfer) (*model.Transfer, error){
	childLogger.Info().Interface("trace-resquest-id", ctx.Value("trace-request-id")).Msg("GetTransfer")

	// Trace
	span := tracerProvider.Span(ctx, "database.GetTransfer")
	defer span.End()

	// get DB connection
	conn, err := w.DatabasePGServer.Acquire(ctx)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer w.DatabasePGServer.Release(conn)

	// Prepare
	res_accountFrom := model.AccountStatement{}
	res_accountTo := model.AccountStatement{}
	res_transfer := model.Transfer{	AccountFrom: &res_accountFrom,
									AccountTo: &res_accountTo}

	// Query e Execute
	query :=  `SELECT 	trans.id,
						fk_account_id_from,
						fr.account_id,
						fk_account_id_to,
						t.account_id,
						type_charge,
						status,
						transfer_at,
						currency, 
						amount,
						transaction_id
				FROM transfer_moviment as trans,
					account as fr,
					account as t
				WHERE trans.id = $1
				and fk_account_id_from = fr.id
				and fk_account_id_to = t.id	`

	rows, err := conn.Query(ctx, query, transfer.ID)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan( 	&res_transfer.ID,
							&res_accountFrom.FkAccountID, 
							&res_accountFrom.AccountID, 
							&res_accountTo.FkAccountID,
							&res_accountTo.AccountID,
							&res_transfer.Type, 
							&res_transfer.Status,
							&res_transfer.TransferAt,
							&res_transfer.Currency,
							&res_transfer.Amount,
							&res_transfer.TransactionID,
						)
		if err != nil {
			return nil, errors.New(err.Error())
        }

		return &res_transfer , nil
	}

	return nil, erro.ErrNotFound
}