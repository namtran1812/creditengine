//go:build integration
// +build integration

package store_test

import (
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestDBPing is a safe smoke integration test that verifies a Postgres
// instance reachable via DATABASE_URL is responsive. It does not modify
// any data. Mark this test with the `integration` build tag so it only
// runs when explicitly requested.
func TestDBPing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// set a short timeout context for the ping
	c := make(chan error, 1)
	go func() {
		c <- db.Ping()
	}()

	select {
	case err := <-c:
		if err != nil {
			t.Fatalf("db ping failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("db ping timed out")
	}
}
