# go-fund-transfer

POC for test purposes

CRUD a transfer_moviment

## Diagram

1.1 go-fund-transfer (get:get/AccountID}) == (REST) ==> go-account (service.Get) ==>(event:topic.CREDIT / status:CREDIT_EVENT_CREATED) == (KAFKA)

After

kafka <==(topic.CREDIT)==> go-worker-credit (GROUP-02) (post:/add) ==(REST)==> go-credit(Service.Add) and change the transfer_moviment to CREDIT_DONE

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

+ POST /transfer

        {
            "account_id_from": "ACC-1",
            "account_id_to": "ACC-2",
            "type_charge": "TRANSFER",
            "currency": "BRL",
            "amount": 1.00
        }