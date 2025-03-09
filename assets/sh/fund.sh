#!/bin/bash

var_acc=0
genAcc(){
    var_acc=$(($RANDOM%($max-$min+1)+$min))
}

var_accfrom=0
genAccFrom(){
    var_accfrom=$(($RANDOM%($maxf-$minf+1)+$minf))
}

var_amount=0
genAmount(){
    var_amount=$(($RANDOM%($max_amount-$min_amount+1)+$min_amount))
}

# --------------------creditFundSchedule-------------------------
domain=localhost:5005/creditFundSchedule

min=500
max=510

max_amount=20
min_amount=10

for (( x=0; x<=10; x++ ))
do
    genAcc
    genAmount
    echo curl -X POST $domain -H 'Content-Type: application/json' -d '{"currency": "BRL","amount": '$var_amount',"account_id_to": "ACC-'$var_acc'"}'
    #curl -X POST $domain -H 'Content-Type: application/json' -d '{"currency": "BRL","amount": '$var_amount',"account_id_to": "ACC-'$var_acc'"}'
done

# -------------------transfer-------------------------
domain=localhost:5005/transfer

minf=511
maxf=520

min=500
max=510

max_amount=50
min_amount=10

for (( x=0; x<=10; x++ ))
do
    genAcc
    genAccFrom
    genAmount
    echo curl -X POST $domain -H 'Content-Type: application/json' -d '{"account_id_from":"ACC-'$var_accfrom'","account_id_to":"ACC-'$var_acc'","type_charge":"TRANSFER","currency":"BRL","amount": '$var_amount'}'
    curl -X POST $domain -H 'Content-Type: application/json' -d '{"account_id_from":"ACC-'$var_accfrom'","account_id_to":"ACC-'$var_acc'","type_charge":"TRANSFER","currency":"BRL","amount": '$var_amount'}'
done

# --------------------Load n per 1-------------------------
domain=localhost:5005/debitFundSchedule

min=500
max=510

max_amount=-10
min_amount=-30

for (( x=0; x<=10; x++ ))
do
    genAcc
    genAmount
    echo curl -X POST $domain -H 'Content-Type: application/json' -d '{"currency": "BRL","amount":'$var_amount',"account_id_to":"ACC-'$var_acc'"}'
    curl -X POST $domain -H 'Content-Type: application/json' -d '{"currency": "BRL","amount":'$var_amount',"account_id_to":"ACC-'$var_acc'"}'
done


