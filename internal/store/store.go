package store

import (
    "context"
    "database/sql"
    "errors"
    "log"
    "time"

    "github.com/namtran/creditengine/internal/models"
)

type Store struct {
    db *sql.DB
}

func New(db *sql.DB) *Store { return &Store{db: db} }

// GetPendingDeposits returns deposits that are not yet credited
func (s *Store) GetPendingDeposits(ctx context.Context) ([]models.Deposit, error) {
    rows, err := s.db.QueryContext(ctx, `SELECT id, tx_hash, address, amount, confirmations, tx_block, block_hash, status, received_at FROM deposits WHERE status = 'pending'`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var res []models.Deposit
    for rows.Next() {
        var d models.Deposit
        if err := rows.Scan(&d.ID, &d.TxHash, &d.Address, &d.Amount, &d.Confirmations, &d.TxBlock, &d.BlockHash, &d.Status, &d.ReceivedAt); err != nil {
            return nil, err
        }
        res = append(res, d)
    }
    return res, nil
}

// UpdateDepositConfirmations updates the confirmations column for a deposit
func (s *Store) UpdateDepositConfirmations(ctx context.Context, id int64, confirmations uint64) error {
    _, err := s.db.ExecContext(ctx, `UPDATE deposits SET confirmations = $1 WHERE id = $2`, confirmations, id)
    return err
}

// UpdateDepositTxInfo stores tx block and block hash for a deposit
func (s *Store) UpdateDepositTxInfo(ctx context.Context, id int64, txBlock uint64, blockHash string) error {
    _, err := s.db.ExecContext(ctx, `UPDATE deposits SET tx_block = $1, block_hash = $2 WHERE id = $3`, txBlock, blockHash, id)
    return err
}

// MarkDepositReorged marks a deposit as reorged when its receipt disappears or block hash mismatches
func (s *Store) MarkDepositReorged(ctx context.Context, id int64) error {
    _, err := s.db.ExecContext(ctx, `UPDATE deposits SET status = 'reorged' WHERE id = $1`, id)
    if err != nil {
        return err
    }
    _, err = s.db.ExecContext(ctx, `INSERT INTO audits(deposit_id, action, created_at) VALUES($1, $2, $3)`, id, "reorged", time.Now())
    if err != nil {
        log.Printf("failed to write audit: %v", err)
    }
    return nil
}

// ListDeposits returns deposits (optionally all statuses)
func (s *Store) ListDeposits(ctx context.Context) ([]models.Deposit, error) {
    rows, err := s.db.QueryContext(ctx, `SELECT id, tx_hash, address, amount, confirmations, tx_block, block_hash, status, received_at FROM deposits ORDER BY received_at DESC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var res []models.Deposit
    for rows.Next() {
        var d models.Deposit
        if err := rows.Scan(&d.ID, &d.TxHash, &d.Address, &d.Amount, &d.Confirmations, &d.TxBlock, &d.BlockHash, &d.Status, &d.ReceivedAt); err != nil {
            return nil, err
        }
        res = append(res, d)
    }
    return res, nil
}

// ReverseCredit attempts to reverse a previously credited deposit (for demo/test only)
func (s *Store) ReverseCredit(ctx context.Context, depositID int64) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback()
        }
    }()

    // find deposit and status
    var addr string
    var amount int64
    var status string
    err = tx.QueryRowContext(ctx, `SELECT address, amount, status FROM deposits WHERE id = $1 FOR UPDATE`, depositID).Scan(&addr, &amount, &status)
    if err != nil {
        return err
    }
    if status != "credited" {
        return errors.New("deposit not credited")
    }

    // decrement account balance (simple demo)
    _, err = tx.ExecContext(ctx, `UPDATE accounts SET balance = balance - $1 WHERE address = $2`, amount, addr)
    if err != nil {
        return err
    }

    _, err = tx.ExecContext(ctx, `UPDATE deposits SET status = 'reversed' WHERE id = $1`, depositID)
    if err != nil {
        return err
    }

    _, err = tx.ExecContext(ctx, `INSERT INTO audits(deposit_id, action, created_at) VALUES($1, $2, $3)`, depositID, "reversed", time.Now())
    if err != nil {
        log.Printf("failed to write audit: %v", err)
    }

    if err := tx.Commit(); err != nil {
        return err
    }
    return nil
}


var ErrAlreadyCredited = errors.New("already credited")

// CreditIfNotCredited performs idempotent credit: only credits if deposit not previously credited.
func (s *Store) CreditIfNotCredited(ctx context.Context, d models.Deposit) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback()
        }
    }()

    // check deposit status
    var status string
    err = tx.QueryRowContext(ctx, `SELECT status FROM deposits WHERE id = $1 FOR UPDATE`, d.ID).Scan(&status)
    if err == sql.ErrNoRows {
        return errors.New("deposit not found")
    }
    if err != nil {
        return err
    }
    if status == "credited" {
        return ErrAlreadyCredited
    }

    // update account balance
    _, err = tx.ExecContext(ctx, `UPDATE accounts SET balance = balance + $1 WHERE address = $2`, d.Amount, d.Address)
    if err != nil {
        return err
    }

    // mark deposit credited and write audit
    _, err = tx.ExecContext(ctx, `UPDATE deposits SET status = 'credited', credited_at = $1 WHERE id = $2`, time.Now(), d.ID)
    if err != nil {
        return err
    }

    _, err = tx.ExecContext(ctx, `INSERT INTO audits(deposit_id, action, created_at) VALUES($1, $2, $3)`, d.ID, "credited", time.Now())
    if err != nil {
        log.Printf("failed to write audit: %v", err)
    }

    if err := tx.Commit(); err != nil {
        return err
    }
    return nil
}
