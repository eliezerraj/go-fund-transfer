package postgre

import (
	"context"
	"time"
	"errors"

	_ "github.com/lib/pq"
	"database/sql"

	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/lib"
)

func (w WorkerRepository) Transfer(ctx context.Context, tx *sql.Tx ,transfer core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("Transfer")
	childLogger.Debug().Interface("transfer:",transfer).Msg("")

	span := lib.Span(ctx, "repo.Transfer")	
    defer span.End()

	stmt, err := tx.Prepare(`INSERT INTO transfer_moviment ( 	fk_account_id_from, 
																fk_account_id_to,
																type_charge,
																status,  
																transfer_at,
																currency,
																amount) 
									VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id`)
	if err != nil {
		childLogger.Error().Err(err).Msg("INSERT statement")
		return nil, errors.New(err.Error())
	}

	var id int
	err = stmt.QueryRowContext(	ctx, 
								transfer.FkAccountIDFrom, 
								transfer.FkAccountIDTo, 
								transfer.Type,
								transfer.Status,
								time.Now(),
								transfer.Currency,
								transfer.Amount).Scan(&id)
	if err != nil {
		childLogger.Error().Err(err).Msg("Exec statement")
		return nil, errors.New(err.Error())
	}

	res_transfer := core.Transfer{}
	res_transfer.ID = id
	res_transfer.TransferAt = time.Now()

	defer stmt.Close()
	return &res_transfer , nil
}

func (w WorkerRepository) Get(ctx context.Context, transfer core.Transfer) (*core.Transfer, error){
	childLogger.Debug().Msg("Get")

	span := lib.Span(ctx, "repo.Get")	
    defer span.End()

	client:= w.databaseHelper.GetConnection()
	
	result_query := core.Transfer{}
	rows, err := client.QueryContext(ctx, `SELECT 	id, 
													fk_account_id_to, 
													fk_account_id_from, 
													type_charge,
													status,
													transfer_at,
													currency, 
													amount
											FROM transfer_moviment 
											WHERE id =$1 `, transfer.ID)
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

func (w WorkerRepository) Update(ctx context.Context, tx *sql.Tx, transfer core.Transfer) (int64, error){
	childLogger.Debug().Msg("Update")
	childLogger.Debug().Interface("transfer : ", transfer).Msg("")

	span := lib.Span(ctx, "repo.Update")	
    defer span.End()

	stmt, err := tx.Prepare(`Update transfer_moviment
									set status = $2
								where id = $1 `)
	if err != nil {
		childLogger.Error().Err(err).Msg("UPDATE statement")
		return 0, errors.New(err.Error())
	}

	result, err := stmt.ExecContext(ctx,	
									transfer.ID,
									transfer.Status,
								)
	if err != nil {
		childLogger.Error().Err(err).Msg("Exec statement")
		return 0, errors.New(err.Error())
	}

	rowsAffected, _ := result.RowsAffected()
	childLogger.Debug().Int("rowsAffected : ",int(rowsAffected)).Msg("")

	defer stmt.Close()
	return rowsAffected , nil
}
