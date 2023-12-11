package core

import (
	"time"

)

type Transfer struct {
	ID				int			`json:"id,omitempty"`
	AccountIDFrom	string		`json:"account_id_from,omitempty"`
	FkAccountIDFrom	int			`json:"fk_account_id_from,omitempty"`
	TransferAt		time.Time 	`json:"transfer_at,omitempty"`
	Type			string  	`json:"type_charge,omitempty"`
	Status			string  	`json:"status,omitempty"`
	Currency		string  	`json:"currency,omitempty"`
	Amount			float64 	`json:"amount,omitempty"`
	AccountIDTo		string		`json:"account_id_to,omitempty"`
	FkAccountIDTo	int			`json:"fk_account_id_to,omitempty"`
}
