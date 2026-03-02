# Foxygen Vibe

Minimal fullstack starter:

- `client/`: React + Tailwind (Vite)
- `server/`: Go API with PostgreSQL-ready environment wiring
- `docker-compose.yml`: local PostgreSQL service

## Run the server

1. Start PostgreSQL if you want database-backed responses:
   `docker compose up -d postgres`
2. Start the API:
   `cd server && DATABASE_URL=postgres://postgres:postgres@localhost:5432/foxygen_vibe?sslmode=disable go run .`

The API runs now with stdlib only and exposes health/message endpoints immediately. `DATABASE_URL` is wired in so the next backend step can add a PostgreSQL driver and persistence.

## Prepare sqlc

The backend now includes `sqlc` input files under `server/db/`.

1. Install the required tools:
   `cd server && go get github.com/jackc/pgx/v5 github.com/jackc/pgx/v5/stdlib && go mod tidy`
   `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
2. Generate Go code:
   `cd server && sqlc generate`

Generated files will be written to `server/internal/db/`.

## Run the client

1. Install dependencies:
   `cd client && npm install`
2. Start Vite:
   `npm run dev`

The Vite dev server proxies `/api/*` to `http://localhost:8080`.
