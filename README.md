# go-fund-transfer

POC for test purposes.

CRUD a transfer_moviment

## Diagram

1.1 go-fund-transfer (get:get/AccountID}) == (REST) ==> go-account (service.Get) 

1.2 go-fund-transfer (event:topic.debit) == (KAFKA) ==> go-worker-debit/credit 

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

+ POST /debitFundSchedule

        {
            "currency": "BRL",
            "amount": -13.00,
            "account_id_to": "ACC-2"
        }

+ GET /get/1

