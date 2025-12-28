package store_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/namtran/creditengine/internal/models"
	"github.com/namtran/creditengine/internal/store"
)

func TestCreditIfNotCredited_Idempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	s := store.New(db)

	ctx := context.Background()

	// begin
	mock.ExpectBegin()
	// select status
	rows := sqlmock.NewRows([]string{"status"}).AddRow("pending")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM deposits WHERE id = $1 FOR UPDATE")).WillReturnRows(rows)
	// update accounts
	mock.ExpectExec(regexp.QuoteMeta("UPDATE accounts SET balance = balance + $1 WHERE address = $2")).WillReturnResult(sqlmock.NewResult(1, 1))
	// update deposit
	mock.ExpectExec(regexp.QuoteMeta("UPDATE deposits SET status = 'credited', credited_at = $1 WHERE id = $2")).WillReturnResult(sqlmock.NewResult(1, 1))
	// insert audit
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audits(deposit_id, action, created_at) VALUES($1, $2, $3)")).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	d := models.Deposit{ID: 1, TxHash: "0xabc", Address: "0xaddr", Amount: 1000, Confirmations: 12}

	// call through
	if err := s.CreditIfNotCredited(ctx, d); err != nil {
		t.Fatalf("credit failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestGetPendingDeposits_Nulls(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	s := store.New(db)

	// prepare rows with NULL tx_block and NULL block_hash
	ts, _ := time.Parse("2006-01-02 15:04:05", "2025-12-21 00:00:00")
	rows := sqlmock.NewRows([]string{"id", "tx_hash", "address", "amount", "confirmations", "tx_block", "block_hash", "status", "received_at"}).AddRow(1, "0xabc", "0xaddr", 1000, 0, nil, nil, "pending", ts)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, tx_hash, address, amount, confirmations, tx_block, block_hash, status, received_at FROM deposits WHERE status = 'pending'")).WillReturnRows(rows)

	deps, err := s.GetPendingDeposits(context.Background())
	if err != nil {
		t.Fatalf("GetPendingDeposits error: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 deposit, got %d", len(deps))
	}
	d := deps[0]
	if d.TxBlock.Valid {
		t.Fatalf("expected TxBlock to be NULL, got valid: %v", d.TxBlock)
	}
	if d.BlockHash.Valid {
		t.Fatalf("expected BlockHash to be NULL, got valid: %v", d.BlockHash)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
