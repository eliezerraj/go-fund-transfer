# go-fund-transfer

## database

    CREATE TABLE transfer_moviment (
        id                  SERIAL PRIMARY KEY,
        fk_account_id_from  integer REFERENCES account(id),
        fk_account_id_to    integer REFERENCES account(id),
        type_charge         varchar(200) NULL,
        transfer_at         timestamptz NULL,
        currency            varchar(10) NULL,   
        amount              float8 NULL,
        status              varchar(10) NULL
    );