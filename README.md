# On-Chain Deposit Finality & Credit Engine (PoC)

This repository is a small proof-of-concept service that demonstrates safe, idempotent crediting of on-chain deposits into user accounts.

Features
- Idempotent credits: credits are performed inside DB transactions and are safe to retry. The store exposes `CreditIfNotCredited` which checks-and-credits atomically.
- Re-org handling: when a receipt is missing or a tx is marked reverted, deposits are set to `reorged` instead of being credited.
- Testability: components are decoupled for unit testing (sqlmock for DB, a chain mock for RPC behavior).

Run locally (requires Docker)

1. Start services:

   docker-compose up --build

2. The service currently runs for a short time and exits; in real deployments it should run as a long-lived process.

Next steps
- Add integration tests that run against a real Postgres container in CI
- Wire a real Ethereum RPC client and optional subscription-based feeds
- Make the engine stateful with checkpointing and graceful shutdown
- Harden re-org detection and implement configurable rollback strategies

## Core ideas

## What is in this repo

- `cmd/creditengine` — small runner for the service
- `internal/engine` — orchestration/service that polls deposits, queries the chain client, and updates the store
- `internal/store` — Postgres-backed store: queries deposits, updates confirmations, performs idempotent credits, records audits
- `internal/chain` — chain client abstraction and a deterministic mock used in tests
- `internal/models` — shared data types and SQL scan-friendly nullable fields
- `migrations/` — SQL migrations for local development
- `.github/workflows/ci.yml` — CI workflow (vet, golangci-lint, race-tested unit tests)

## Quickstart (local development)

Prerequisites

- Go 1.20+ (module-aware)
- Docker & docker-compose (for running Postgres locally)
- Homebrew (optional) to install `golangci-lint` locally: `brew install golangci-lint`

Run a local Postgres instance (uses docker-compose):

```sh
docker-compose up --build -d
```

Apply migrations (you can use `psql` or any migration runner against the DSN used in the service):

```sh
# example, adjust DSN as needed
psql "postgres://postgres:postgres@localhost:5432/creditengine?sslmode=disable" -f migrations/001_init.sql
```

Run the service locally (development mode):

```sh
# Use env vars or modify DefaultConfig in code
go run ./cmd/creditengine
```

The service is currently a PoC and intentionally minimal — it's safe to run locally to exercise tests and logic.

## Configuration

Configuration is read via the engine `Config` (see `internal/engine/config.go`). Important values:

- RPC URL (Ethereum node)
- Confirmation threshold (number of confirmations before crediting)
- Postgres DSN

For tests and CI the code uses sqlmock and a deterministic chain mock so no external node is required.

## Testing

Run unit tests (with race detector):

```sh
go test -race ./... -v
```

The repository includes unit tests for the `store` idempotency logic and the `engine` processing flow (using mocks).

Integration tests

To avoid running integration tests in the unit CI job, mark long-running DB-backed tests with the `integration` build tag. Example at the top of a test file:

```go
//go:build integration
// +build integration

package store_test

import (
   "testing"
)

func TestSomeIntegration(t *testing.T) {
   if testing.Short() {
      t.Skip("skip integration test in short mode")
   }
   // ... test logic that uses DATABASE_URL or a running Postgres
}
```

Run integration tests locally (after starting Postgres):

```sh
# runs only integration-tagged tests
go test -tags=integration ./... -v
```

Makefile helper targets

There are convenient Makefile targets to mirror CI and help with development:

- `make test` — run unit tests with the race detector
- `make lint` — run `golangci-lint` (requires it to be installed locally)
- `make ci` — run `go vet`, `golangci-lint`, and `go test -race` in sequence (mirrors CI checks)
- `make ci-fix` — attempt auto-fixes (`gofmt -s -w .`, `golangci-lint --fix`) and then run `make ci`
- `make run` / `make build` — run or build the CLI

Example: run CI checks locally:

```sh
make ci
```

Run integration tests with docker-compose (example):

```sh
# start Postgres with docker-compose
docker-compose up --build -d

# export DATABASE_URL (adjust if your docker-compose uses different credentials)
export DATABASE_URL=postgres://postgres:password@127.0.0.1:5432/creditengine?sslmode=disable

# run only integration tests
go test -tags=integration ./... -v

# tear down
docker-compose down

make integration helper

You can also run the full integration flow using the Makefile helper which wraps `docker-compose`, the seed script, and the integration test run:

```sh
# optional: set DATABASE_URL to match your docker-compose credentials
export DATABASE_URL=postgres://postgres:password@127.0.0.1:5432/creditengine?sslmode=disable

# start services, seed DB, run integration tests, teardown
make integration
```

Notes:
- `make integration` assumes `docker-compose.yml` defines the Postgres service and `scripts/seed.sh` seeds the DB. Adjust `DATABASE_URL` if your compose file uses different credentials or hostnames.
- If you prefer `docker compose` (v2) replace `docker-compose` calls in the Makefile accordingly.
```

## Linting and static analysis

We use `golangci-lint` in CI. To run it locally (Homebrew install recommended):

```sh
brew install golangci-lint
golangci-lint run --config .golangci.yml ./...
```

The repository ships a conservative `.golangci.yml` tuned for the PoC to reduce noisy rules.

## CI

GitHub Actions runs the following checks on pushes/PRs:

- `go vet ./...`
- `golangci-lint` (via `golangci/golangci-lint-action@v4`)
- `go test -race ./...`
- `go test ./...`

If you push the branch `ci/add-vet-lint-race` (already present in this repo), open a PR to trigger CI and see the checks.

Integration job notes

- The `integration` job in `.github/workflows/ci.yml` starts Postgres:15 as a service container, runs `migrations/001_init.sql` to create the schema, and then runs integration-tagged tests using `go test -tags=integration ./... -v`.
- Integration tests should be fast smoke checks where possible. For longer scenarios, mark them `integration` and consider running them in a matrix or separate workflow.

## Contributing

This is a PoC — contributions are welcome but please:

- Keep changes focused and small
- Add unit tests for logic changes (sqlmock + chain mock are available)
- Run `golangci-lint` and `go test -race ./...` before opening a PR

## License

MIT — see `LICENSE` (if not present, assume a permissive license for the PoC and add one before production use).

---

If you want, I can also add a small `Makefile` with common targets (`make test`, `make lint`, `make run`) and/or open the PR that contains CI changes. What would you like next?
