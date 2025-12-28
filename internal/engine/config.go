package engine

import "time"

type Config struct {
	RPCUrl        string
	Confirmations uint64
	PollInterval  time.Duration
	PostgresDSN   string
}

func DefaultConfig() *Config {
	return &Config{
		RPCUrl:        "http://localhost:8545",
		Confirmations: 12,
		PollInterval:  2 * time.Second,
		PostgresDSN:   "postgres://postgres:password@localhost:5432/creditengine?sslmode=disable",
	}
}
