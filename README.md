# Foxygen Vibe

Minimal fullstack starter:

- `client/`: React + Tailwind (Vite)
- `server/`: Go API with PostgreSQL-ready environment wiring
- `docker-compose.yml`: local PostgreSQL service

## Run the server

1. Start PostgreSQL:
   `docker compose up -d postgres`
2. Start the API:
   `cd server && go run .`

The server reads `server/.env` for `DB_*` settings and builds a PostgreSQL connection string from that file. Explicit shell environment variables still override values from `.env`, and `DATABASE_URL` still takes precedence over the split fields.

## Prepare sqlc

The backend now includes `sqlc` input files under `server/db/`.

1. Install the required tools:
   `cd server && go get github.com/jackc/pgx/v5 github.com/jackc/pgx/v5/stdlib && go mod tidy`
   `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
2. Generate Go code:
   `cd server && sqlc generate`

Generated files will be written to `server/internal/db/`.

## Import legacy users

If you have a legacy CouchDB `_all_docs` export, you can import the `user_*` records into the current PostgreSQL schema:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Run the importer from the `server/` directory:
   `go run ./cmd/import-users -source "/absolute/path/to/_all_docs.json" -default-password "ChangeMe123!"`

Notes:

- The legacy dump does not contain passwords, so the importer assigns the temporary password you provide to every imported account.
- Imported usernames are assigned deterministically as `user_1`, `user_2`, `user_3`, and so on.
- The command is safe to rerun. If a username already exists, that user is skipped.
- Use `-dry-run` first if you want to inspect the planned usernames before writing to the database.

## Run the client

1. Install dependencies:
   `cd client && npm install`
2. Start Vite:
   `npm run dev`

The Vite dev server proxies `/api/*` to `http://localhost:8080`.
