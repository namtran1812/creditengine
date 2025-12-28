package chain

import (
	"context"
)

// MockClient is a simple test double for ChainClient.
type MockClient struct {
	Block  uint64
	TxInfo map[string]struct {
		Block    uint64
		Hash     string
		Reverted bool
	}
}

func NewMock() *MockClient {
	return &MockClient{TxInfo: make(map[string]struct {
		Block    uint64
		Hash     string
		Reverted bool
	})}
}

func (m *MockClient) BlockNumber(ctx context.Context) (uint64, error) { return m.Block, nil }

func (m *MockClient) Confirmations(ctx context.Context, txBlockNumber uint64) (uint64, error) {
	if m.Block < txBlockNumber {
		return 0, nil
	}
	return m.Block - txBlockNumber + 1, nil
}

func (m *MockClient) ConfirmationsFromTxHash(ctx context.Context, txHash string) (uint64, uint64, string, bool, bool, error) {
	info, ok := m.TxInfo[txHash]
	if !ok {
		return 0, 0, "", false, false, nil
	}
	conf, _ := m.Confirmations(ctx, info.Block)
	return info.Block, conf, info.Hash, true, info.Reverted, nil
}
