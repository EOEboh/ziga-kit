# Ziga-Kit — API

A minimal client portal backend for freelancers. Built with Go, Chi, pgx, and Cloudflare R2.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Runtime |
| Docker | any | Local Postgres |
| `air` | latest | Hot reload in dev |
| `psql` | any | Running migrations |

```bash
# Install air (hot reload)
go install github.com/air-verse/air@latest
```

---

## First-time setup

```bash
# 1. Clone and enter the repo
git clone https://github.com/zigakit/api && cd api

# 2. Copy env file and fill in your values
cp .env.example .env

# 3. Start local Postgres and run migrations in one command
make setup

# 4. Start the dev server with hot reload
make dev
```

The API will be available at `http://localhost:8080`.
Health check: `curl http://localhost:8080/health`

---

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go          # Entrypoint — wires config, DB, router, server
│
├── internal/
│   ├── config/
│   │   └── config.go        # Env-based config struct
│   ├── db/
│   │   └── db.go            # pgxpool connection bootstrapper
│   ├── handlers/            # HTTP handlers, one file per domain (auth, projects, …)
│   ├── middleware/          # Chi middleware (auth, logging, rate-limit, …)
│   ├── models/              # Domain structs (User, Project, Deliverable, …)
│   └── mailer/              # Email sending via Resend
│
├── migrations/
│   └── 001_initial_schema.sql
│
├── scripts/                 # One-off utility scripts
├── .air.toml                # Hot-reload config
├── .env.example
├── docker-compose.yml       # Local Postgres
├── go.mod
└── Makefile
```

---

## Make commands

```bash
make help        # List all commands
make setup       # First-time: start DB + run migrations
make dev         # Hot-reload dev server
make build       # Compile binary to ./bin/api
make migrate     # Run all pending SQL migrations
make db-up       # Start Postgres container
make db-down     # Stop Postgres container
make db-reset    # Wipe DB volume and restart (destructive)
make test        # Run test suite
make lint        # Run golangci-lint
make tidy        # go mod tidy + verify
```