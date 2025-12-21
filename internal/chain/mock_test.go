package chain_test

import (
    "context"
    "testing"

    "github.com/namtran/creditengine/internal/chain"
)

func TestMockClient_ReturnsInfo(t *testing.T) {
    m := chain.NewMock()
    m.Block = 100
    m.TxInfo["0xabc"] = struct{Block uint64; Hash string; Reverted bool}{Block: 90, Hash: "0xhash", Reverted: false}

    txBlock, conf, hash, found, reverted, err := m.ConfirmationsFromTxHash(context.Background(), "0xabc")
    if err != nil { t.Fatalf("err: %v", err) }
    if !found { t.Fatalf("expected found") }
    if txBlock != 90 { t.Fatalf("expected txBlock 90 got %d", txBlock) }
    if conf == 0 { t.Fatalf("expected confirmations > 0") }
    if hash != "0xhash" { t.Fatalf("unexpected hash") }
    if reverted { t.Fatalf("unexpected reverted") }
}
