# On-Chain Deposit Finality & Credit Engine (PoC)

This repository contains a small proof-of-concept service that monitors blockchain deposits and credits user accounts only after a configurable number of confirmations.

Features
- Idempotent crediting using DB transactions
- Simple polling loop (placeholder for real RPC logic)
- Postgres migrations included for local testing

Run locally (requires Docker)

1. Start services:

   docker-compose up --build

2. The service currently runs for a short time and exits; in real deployments it should run as a long-lived process.

Next steps
- Implement RPC queries to fetch confirmations and detect re-orgs
- Add tests and monitoring/alerts
