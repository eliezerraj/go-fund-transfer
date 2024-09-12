package storage

import (
	"context"
	"time"
	"errors"

	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/lib"
	"github.com/go-fund-transfer/internal/erro"
	"github.com/go-fund-transfer/internal/repository/pg"

	"github.com/rs/zerolog/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var childLogger = log.With().Str("repository.pg", "storage").Logger()

//-----------------------------------------------
type WorkerRepository struct {
	databasePG pg.DatabasePG
}

func NewWorkerRepository(databasePG pg.DatabasePG) WorkerRepository {
	childLogger.Debug().Msg("NewWorkerRepository")
	return WorkerRepository{
		databasePG: databasePG,
	}
}

func (w WorkerRepository) SetSessionVariable(ctx context.Context, userCredential string) (bool, error) {
	childLogger.Debug().Msg("++++++++++++++++++++++++++++++++")
	childLogger.Debug().Msg("SetSessionVariable")

	conn, err := w.databasePG.Acquire(ctx)
	if err != nil {
		childLogger.Error().Err(err).Msg("Erro Acquire")
		return false, errors.New(err.Error())
	}
	defer w.databasePG.Release(conn)
	
	_, err = conn.Query(ctx, "SET sess.user_credential to '" + userCredential+ "'")
	if err != nil {
		childLogger.Error().Err(err).Msg("SET SESSION statement ERROR")
		return false, errors.New(err.Error())
	}

	return true, nil
}

func (w WorkerRepository) GetSessionVariable(ctx context.Context) (*string, error) {
	childLogger.Debug().Msg("++++++++++++++++++++++++++++++++")
	childLogger.Debug().Msg("GetSessionVariable")

	conn, err := w.databasePG.Acquire(ctx)
	if err != nil {
		childLogger.Error().Err(err).Msg("Erro Acquire")
		return nil, errors.New(err.Error())
	}
	defer w.databasePG.Release(conn)

	var res_balance string
	rows, err := conn.Query(ctx, "SELECT current_setting('sess.user_credential')" )
	if err != nil {
		childLogger.Error().Err(err).Msg("Prepare statement")
		return nil, errors.New(err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan( &res_balance )
		if err != nil {
			childLogger.Error().Err(err).Msg("Scan statement")
			return nil, errors.New(err.Error())
        }
		return &res_balance, nil
	}

	return nil, erro.ErrNotFound
}

func (w WorkerRepository) StartTx(ctx context.Context) (pgx.Tx, *pgxpool.Conn,error) {
	childLogger.Debug().Msg("StartTx")

	span := lib.Span(ctx, "repo.StartTx")
	defer span.End()

	span = lib.Span(ctx, "repo.Acquire")
	conn, err := w.databasePG.Acquire(ctx)
	if err != nil {
		childLogger.Error().Err(err).Msg("Erro Acquire")
		return nil, nil, errors.New(err.Error())
	}
	span.End()

	tx, err := conn.Begin(ctx)
    if err != nil {
        return nil, nil ,errors.New(err.Error())
    }

	return tx, conn, nil
}

func (w WorkerRepository) ReleaseTx(connection *pgxpool.Conn) {
	childLogger.Debug().Msg("ReleaseTx")

	defer connection.Release()
}
//------------
func (w WorkerRepository) Transfer(ctx context.Context, tx pgx.Tx ,transfer *core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("Transfer")
	//childLogger.Debug().Interface("transfer:",transfer).Msg("")

	span := lib.Span(ctx, "repo.Transfer")	
    defer span.End()

	transfer.TransferAt = time.Now()

	query := `INSERT INTO transfer_moviment(fk_account_id_from, 
											fk_account_id_to,
											type_charge,
											status,  
											transfer_at,
											currency,
											amount) 
				VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	row := tx.QueryRow(ctx, query, transfer.FkAccountIDFrom, transfer.FkAccountIDTo, transfer.Type,transfer.Status,transfer.TransferAt,transfer.Currency,transfer.Amount) 					

	var id int
	if err := row.Scan(&id); err != nil {
		childLogger.Error().Err(err).Msg("INSERT statement")
		return nil, errors.New(err.Error())
	}

	transfer.ID = id

	return transfer , nil
}

func (w WorkerRepository) Get(ctx context.Context, transfer *core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("Get")

	span := lib.Span(ctx, "repo.Get")	
    defer span.End()

	span = lib.Span(ctx, "repo.Acquire")
	conn, err := w.databasePG.Acquire(ctx)
	if err != nil {
		childLogger.Error().Err(err).Msg("Erro Acquire")
		return nil, errors.New(err.Error())
	}
	span.End()
	defer w.databasePG.Release(conn)

	result_query := core.Transfer{}

	query :=  `SELECT 	id, 
						fk_account_id_to, 
						fk_account_id_from, 
						type_charge,
						status,
						transfer_at,
						currency, 
						amount
				FROM transfer_moviment 
				WHERE id =$1 `

	rows, err := conn.Query(ctx, query, transfer.ID)
	if err != nil {
		childLogger.Error().Err(err).Msg("SELECT statement")
		return nil, errors.New(err.Error())
	}

	for rows.Next() {
		err := rows.Scan( 	&result_query.ID, 
							&result_query.FkAccountIDTo, 
							&result_query.FkAccountIDFrom, 
							&result_query.Type, 
							&result_query.Status,
							&result_query.TransferAt,
							&result_query.Currency,
							&result_query.Amount,
						)
		if err != nil {
			childLogger.Error().Err(err).Msg("Scan statement")
			return nil, errors.New(err.Error())
        }
	}

	defer rows.Close()
	return &result_query , nil
}

func (w WorkerRepository) Update(ctx context.Context, tx pgx.Tx, transfer *core.Transfer) (int64, error){
	childLogger.Debug().Msg("Update")
	childLogger.Debug().Interface("transfer : ", transfer).Msg("")

	span := lib.Span(ctx, "repo.Update")	
    defer span.End()

	query := `Update transfer_moviment
					set status = $2
				where id = $1`

	row, err := tx.Exec(ctx, query, transfer.ID, transfer.Status)
	if err != nil {
		childLogger.Error().Err(err).Msg("Exec statement")
		return 0, errors.New(err.Error())
	}

	childLogger.Debug().Interface("rowsAffected : ", row.RowsAffected()).Msg("")

	return int64(row.RowsAffected()) , nil
}