package chain

import (
    "context"
    "math/big"

    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
)

type Client struct{
    cli *ethclient.Client
}

// ChainClient defines the subset of chain behaviours we need. This allows tests to
// inject a mock implementation.
type ChainClient interface {
    BlockNumber(ctx context.Context) (uint64, error)
    Confirmations(ctx context.Context, txBlockNumber uint64) (uint64, error)
    ConfirmationsFromTxHash(ctx context.Context, txHash string) (txBlock uint64, confirmations uint64, blockHash string, found bool, reverted bool, err error)
}

func New(url string) (*Client, error) {
    c, err := ethclient.Dial(url)
    if err != nil {
        return nil, err
    }
    return &Client{cli: c}, nil
}

// BlockNumber returns the latest block number.
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
    h, err := c.cli.BlockNumber(ctx)
    if err != nil {
        return 0, err
    }
    return h, nil
}

// Confirmations computes confirmations for a given tx by comparing provided block number.
// In a full implementation we'd fetch the tx receipt and derive its block number.
func (c *Client) Confirmations(ctx context.Context, txBlockNumber uint64) (uint64, error) {
    h, err := c.BlockNumber(ctx)
    if err != nil {
        return 0, err
    }
    if h < txBlockNumber {
        return 0, nil
    }
    return h - txBlockNumber + 1, nil
}

// helper to convert big.Int
func bigToU64(b *big.Int) uint64 {
    if b == nil { return 0 }
    return b.Uint64()
}

// ConfirmationsFromTxHash fetches the tx receipt and returns the block number and confirmations.
func (c *Client) ConfirmationsFromTxHash(ctx context.Context, txHash string) (txBlock uint64, confirmations uint64, blockHash string, found bool, reverted bool, err error) {
    // use underlying rpc client to get receipt
    // ethclient has TransactionReceipt which wraps rpc call
    h := common.HexToHash(txHash)
    rec, err := c.cli.TransactionReceipt(ctx, h)
    if err != nil {
        // if receipt not found, return found=false
        // note: client.TransactionReceipt returns an error when not found
        return 0, 0, "", false, false, nil
    }
    if rec == nil {
        return 0, 0, "", false, false, nil
    }
    txBlock = bigToU64(rec.BlockNumber)
    bh := rec.BlockHash.Hex()
    bn, err := c.BlockNumber(ctx)
    if err != nil {
        return txBlock, 0, bh, true, false, err
    }
    if bn < txBlock {
        return txBlock, 0, bh, true, false, nil
    }
    confirmations = bn - txBlock + 1
    reverted = rec.Status == types.ReceiptStatusFailed
    return txBlock, confirmations, bh, true, reverted, nil
}
