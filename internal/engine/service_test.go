package engine

import (
    "context"
    "testing"
    "time"
    "regexp"

    "github.com/DATA-DOG/go-sqlmock"
    "github.com/namtran/creditengine/internal/chain"
    st "github.com/namtran/creditengine/internal/store"
)

func TestProcessOnce_CreditsWhenConfirmed(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock: %v", err) }
    defer db.Close()

    // pending deposit row
    rows := sqlmock.NewRows([]string{"id","tx_hash","address","amount","confirmations","tx_block","block_hash","status","received_at"}).AddRow(1,"0xabc","0xaddr",1000,11,90,"0xhash","pending",time.Now())
    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, tx_hash, address, amount, confirmations, tx_block, block_hash, status, received_at FROM deposits WHERE status = 'pending'")).WillReturnRows(rows)

    // Update tx info
    mock.ExpectExec(regexp.QuoteMeta("UPDATE deposits SET tx_block = $1, block_hash = $2 WHERE id = $3")).WillReturnResult(sqlmock.NewResult(1,1))
    // Update confirmations
    mock.ExpectExec(regexp.QuoteMeta("UPDATE deposits SET confirmations = $1 WHERE id = $2")).WillReturnResult(sqlmock.NewResult(1,1))
    // Begin credit transaction
    mock.ExpectBegin()
    mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM deposits WHERE id = $1 FOR UPDATE")).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("pending"))
    mock.ExpectExec(regexp.QuoteMeta("UPDATE accounts SET balance = balance + $1 WHERE address = $2")).WillReturnResult(sqlmock.NewResult(1,1))
    mock.ExpectExec(regexp.QuoteMeta("UPDATE deposits SET status = 'credited', credited_at = $1 WHERE id = $2")).WillReturnResult(sqlmock.NewResult(1,1))
    mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audits(deposit_id, action, created_at) VALUES($1, $2, $3)")).WillReturnResult(sqlmock.NewResult(1,1))
    mock.ExpectCommit()

    // Setup mock chain with confirmations >= 12
    mc := chain.NewMock()
    mc.Block = 102
    mc.TxInfo["0xabc"] = struct{Block uint64; Hash string; Reverted bool}{Block: 90, Hash: "0xhash", Reverted: false}

    cfg := DefaultConfig()
    svc := NewServiceWithStore(cfg, st.New(db), mc)

    if err := svc.ProcessOnce(context.Background()); err != nil {
        t.Fatalf("ProcessOnce error: %v", err)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

func TestProcessOnce_MarksReorgWhenReceiptMissing(t *testing.T) {
    db, mock, err := sqlmock.New()
    if err != nil { t.Fatalf("sqlmock: %v", err) }
    defer db.Close()

    // pending deposit row
    rows := sqlmock.NewRows([]string{"id","tx_hash","address","amount","confirmations","tx_block","block_hash","status","received_at"}).AddRow(2,"0xdef","0xaddr",2000,0,nil,nil,"pending",time.Now())
    mock.ExpectQuery(regexp.QuoteMeta("SELECT id, tx_hash, address, amount, confirmations, tx_block, block_hash, status, received_at FROM deposits WHERE status = 'pending'")).WillReturnRows(rows)

    // When receipt not found, mark reorged
    mock.ExpectExec(regexp.QuoteMeta("UPDATE deposits SET status = 'reorged' WHERE id = $1")).WithArgs(2).WillReturnResult(sqlmock.NewResult(1,1))
    mock.ExpectExec(regexp.QuoteMeta("INSERT INTO audits(deposit_id, action, created_at) VALUES($1, $2, $3)")).WillReturnResult(sqlmock.NewResult(1,1))

    mc := chain.NewMock()
    mc.Block = 100
    // No entry for 0xdef => receipt not found

    cfg := DefaultConfig()
    svc := NewServiceWithStore(cfg, st.New(db), mc)

    if err := svc.ProcessOnce(context.Background()); err != nil {
        t.Fatalf("ProcessOnce error: %v", err)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Fatalf("unmet expectations: %v", err)
    }
}

