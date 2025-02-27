# go-fund-transfer

POC for test purposes

CRUD a transfer_moviment

## Diagram

1.1 go-fund-transfer (get:get/AccountID}) == (REST) ==> go-account (service.Get) ==>(event:topic.CREDIT / status:CREDIT_EVENT_CREATED) == (KAFKA)

The service go-worker-credit was on charge of to consume the event and change the status to CREDIT_DONE

kafka <==(topic.CREDIT)==> go-worker-credit (GROUP-02) (post:/add) ==(REST)==> go-credit(Service.Add) and change the transfer_moviment to CREDIT_DONE
Or
sqs <==(topic.CREDIT)==> 

## database

    CREATE TABLE public.transfer_moviment (
        id serial4 NOT NULL,
        fk_account_id_from int4 NULL,
        fk_account_id_to int4 NULL,
        type_charge varchar(200) NULL,
        transfer_at timestamptz NULL,
        currency varchar(10) NULL,
        amount float8 NULL,
        status varchar(200) NULL,
        CONSTRAINT transfer_moviment_pkey PRIMARY KEY (id)
    );

    ALTER TABLE public.transfer_moviment 
    ADD CONSTRAINT transfer_moviment_fk_account_id_from_fkey 
    FOREIGN KEY (fk_account_id_from) REFERENCES public.account(id);
    ALTER TABLE public.transfer_moviment 
    ADD CONSTRAINT transfer_moviment_fk_account_id_to_fkey 
    FOREIGN KEY (fk_account_id_to) REFERENCES public.account(id);


## Endpoints

+ POST /creditTransferEvent

        {
            "account_from": {
                "account_id":"ACC-500"
            },
            "type_charge": "CREDIT",
            "currency": "BRL",
            "amount": 10.00
        }

+ POST /debitTransferEvent

        {
            "account_from": {
                "account_id":"ACC-500"
            },
            "type_charge": "DEBIT",
            "currency": "BRL",
            "amount": -10.00
        }

+ GET /get/1

+ POST /add/transfer

        {
            "account_from": {
                "account_id":"ACC-500"
            },
            "account_to": {
                "account_id":"ACC-600"
            },
            "type_charge": "TRANSFER",
            "currency": "BRL",
            "amount": 10.00
        }

+ POST /add/transferEvent

        {
            "account_from": {
                "account_id":"ACC-500"
            },
            "account_to": {
                "account_id":"ACC-600"
            },
            "type_charge": "TRANSFER",
            "currency": "BRL",
            "amount": 10.00
        }