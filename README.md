# go-fund-transfer

POC for test purposes.

CRUD a account_statement data.

## database

    CREATE TABLE transfer_moviment (
        id                  SERIAL PRIMARY KEY,
        fk_account_id_from  integer REFERENCES account(id),
        fk_account_id_to    integer REFERENCES account(id),
        type_charge         varchar(200) NULL,
        transfer_at         timestamptz NULL,
        currency            varchar(10) NULL,   
        amount              float8 NULL,
        status              varchar(200) NULL
    );

## Endpoints

+ POST /creditFundSchedule

        {
            "currency": "BRL",
            "amount": 13.00,
            "account_id_to": "ACC-2"
        }

This endpoint producer a mesage in Kafka (topic.credit)