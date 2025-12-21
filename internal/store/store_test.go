package store_test

import (
    "context"
    "testing"
    "regexp"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/namtran/creditengine/internal/store"
    "github.com/namtran/creditengine/internal/models"
)

func TestCreditIfNotCredited_Idempotent(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("failed to create sqlmock: %v", err)
    }
    defer db.Close()

    s := store.New(db)

    ctx := context.Background()

    // begin
    mock.ExpectBegin()
    // select status
    rows := sqlmock.NewRows([]string{"status"}).AddRow("pending")
    mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM deposits WHERE id = $1 FOR UPDATE")).WillReturnRows(rows)
    // update accounts
    mock.ExpectExec(regexp.QuoteMeta("UPDATE accounts SET balance = balance + $1 WHERE address = $2")).WillReturnResult(sqlmock.NewResult(1,1))
    // update deposit
    mock.ExpectExec(regexp.QuoteMeta("UPDATE deposits SET status = 'credited', credited_at = $1 WHERE id = $2")).WillReturnResult(sqlmock.NewResult(1,1))
    // insert audit
    mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audits(deposit_id, action, created_at) VALUES($1, $2, $3)")).WillReturnResult(sqlmock.NewResult(1,1))
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
