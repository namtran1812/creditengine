# Makefile for common developer flows

BINARY=creditengine

.PHONY: all test lint run build fmt clean

all: test

test:
	@echo "Running unit tests (race)..."
	go test -race ./... -v

lint:
	@echo "Running golangci-lint (requires golangci-lint installed)..."
	golangci-lint run --config .golangci.yml ./...

ci:
	@echo "Running local CI checks: go vet -> golangci-lint -> go test -race"
	go vet ./...
	golangci-lint run --config .golangci.yml ./...
	go test -race ./... -v

ci-fix:
	@echo "Attempting auto-fixes (gofmt + golangci-lint --fix), then running CI checks"
	gofmt -s -w .
	# golangci-lint --fix will attempt to auto-fix lint issues (requires golangci-lint >= 1.50)
	golangci-lint run --fix --config .golangci.yml ./... || true
	$(MAKE) ci

run:
	@echo "Running the service (development mode)..."
	go run ./cmd/creditengine

build:
	@echo "Building binary..."
	go build -o $(BINARY) ./cmd/creditengine

fmt:
	@echo "Formatting code..."
	gofmt -s -w .

clean:
	@echo "Cleaning..."
	rm -f $(BINARY)

integration:
	@echo "Starting docker-compose services..."
	docker-compose up --build -d

	@echo "Seeding DB..."
	./scripts/seed.sh

	@echo "Running integration tests (tagged)..."
	go test -tags=integration ./... -v

	@echo "Tearing down docker-compose services..."
	docker-compose down
