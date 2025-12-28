package engine

import "time"

type Deposit struct {
	ID            int64
	TxHash        string
	Address       string
	Amount        int64
	Confirmations uint64
	TxBlock       uint64
	BlockHash     string
	Status        string
	ReceivedAt    time.Time
}

type Account struct {
	ID      int64
	Address string
	Balance int64
}
