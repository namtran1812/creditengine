package models

import (
	"database/sql"
	"time"
)

type Deposit struct {
	ID            int64
	TxHash        string
	Address       string
	Amount        int64
	Confirmations uint64
	TxBlock       sql.NullInt64
	BlockHash     sql.NullString
	Status        string
	ReceivedAt    time.Time
}

type Account struct {
	ID      int64
	Address string
	Balance int64
}
