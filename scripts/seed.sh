#!/usr/bin/env bash
# scripts/seed.sh - apply migrations and seed example data into DATABASE_URL

set -euo pipefail

# Default DATABASE_URL if not provided
: ${DATABASE_URL:="postgres://postgres:password@127.0.0.1:5432/creditengine?sslmode=disable"}

echo "Using DATABASE_URL=${DATABASE_URL}"

# Apply migration
if ! command -v psql >/dev/null 2>&1; then
  echo "psql not found. On macOS, install via: brew install libpq && brew link --force libpq"
  exit 1
fi

echo "Applying migrations..."
psql "$DATABASE_URL" -f migrations/001_init.sql

echo "Seeding example data..."
psql "$DATABASE_URL" <<'SQL'
BEGIN;

-- sample account
INSERT INTO accounts (address, balance) VALUES ('0xaddr', 0) ON CONFLICT (address) DO NOTHING;

-- sample pending deposits
INSERT INTO deposits (tx_hash, address, amount, confirmations, status, received_at)
VALUES
  ('0xabc', '0xaddr', 1000, 0, 'pending', now()),
  ('0xdef', '0xaddr', 2000, 0, 'pending', now())
ON CONFLICT (tx_hash) DO NOTHING;

COMMIT;
SQL

echo "Seed completed. Example data inserted into accounts and deposits."

echo "You can verify with: psql \"$DATABASE_URL\" -c \"SELECT * FROM accounts; SELECT * FROM deposits;\""
