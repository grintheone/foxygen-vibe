# Foxygen Vibe

Minimal fullstack starter:

- `client/`: React + Tailwind (Vite)
- `server/`: Go API with PostgreSQL-ready environment wiring
- `docker-compose.yml`: local PostgreSQL and MinIO services
- `docker-compose.production.yml`: production-oriented full stack with frontend, API, PostgreSQL, MinIO, and first-start import bootstrap

## Run the production stack with Docker

The production compose file runs the frontend, API, PostgreSQL, and an optional MinIO profile. On startup, the API bootstrap waits for its dependencies, applies `server/db/schema/*.sql`, and imports the legacy dump only when the database is still empty.

Setup:

1. Put the legacy dump at `deploy/production/bootstrap/dump.json`.
2. Create `deploy/production/.env` from `deploy/production/.env.example`.
3. In `deploy/production/.env`, change at least:
   `POSTGRES_PASSWORD`
   `JWT_SECRET`
   `IMPORT_DEFAULT_PASSWORD`
4. Set `DOCKER_NETWORK_SUBNET` to a narrow subnet that does not overlap with any company LAN, VPN, or service network routes reachable from the host.
5. Fill in the object-storage settings with the values provided by your infrastructure team.
6. Start the stack with the first command your host supports:

```sh
docker compose --env-file deploy/production/.env -f docker-compose.production.yml up -d --build
```

```sh
set -a
. deploy/production/.env
set +a
docker compose -f docker-compose.production.yml up -d --build
```

```sh
set -a
. deploy/production/.env
set +a
docker-compose -f docker-compose.production.yml up -d --build
```

If you want a local MinIO container instead of external S3-compatible storage, add `--profile with-minio` to the same command.

Useful notes:

- The frontend is exposed on `PUBLIC_HTTP_PORT` from the env file and proxies `/api/*` to the backend container.
- Set `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, and either `MINIO_REGION` or `MINIO_LOCATION` when using external object storage. `MINIO_ENDPOINT` may be a bare hostname; bootstrap uses port `443` when `MINIO_USE_SSL=true`, otherwise `80`.
- Keep `DOCKER_NETWORK_SUBNET` narrow, such as `/24`, and outside any office, VPN, or service range reachable from the host. If you change it later, bring the stack down first and then start it again so Docker recreates the bridge network.
- PostgreSQL stays on the internal Docker network by default, which is safer for a company server.
- The first-start import is controlled by `BOOTSTRAP_IMPORT_ENABLED=true|false`.
- Imported users receive the temporary password from `IMPORT_DEFAULT_PASSWORD`.
- If the database already contains app data, the bootstrap step skips the import and starts the API normally.
- Use the local `with-minio` profile only when you are not connecting to an existing S3-compatible storage service.
- Before deployment, ask infrastructure which RFC1918 ranges are already in use on that host and pick a Docker subnet outside them.

## Run the server

1. Start PostgreSQL:
   `docker compose up -d postgres`
2. Start the API:
   `cd server && go run .`

The server reads `server/.env` for `DB_*` settings and builds a PostgreSQL connection string from that file. Explicit shell environment variables still override values from `.env`, and `DATABASE_URL` still takes precedence over the split fields.

## Receive synced tickets from an external system

The backend now exposes a webhook-style endpoint at `/api/v1/sync` for server-to-server ticket creation.

1. Set a shared secret in `server/.env`:
   `TICKET_SYNC_SECRET=<sync-shared-secret>`
2. Start the API from `server/`:
   `go run .`
3. Send `POST` requests to your server with the `X-Sync-Secret` header and JSON like:
   ```json
   {
     "author": "00000000-0000-0000-0000-000000000004",
     "author_title": "External Dispatcher",
     "client": "00000000-0000-0000-0000-000000000001",
     "device": "00000000-0000-0000-0000-000000000002",
     "department": "Service Department",
     "reason": "maintenance",
     "description": "Analyzer stopped sending results after reboot.",
     "source": "lab-dispatcher",
     "sync_key": "ticket-18452",
     "urgent": true
   }
   ```

Notes:

- The webhook creates a new ticket in `created` status so a coordinator can triage and assign it later.
- `department` accepts either the department UUID or the unique department title.
- `reason` must match an existing `ticket_reasons.id`.
- `client` and `device` must already exist in PostgreSQL and must be consistent with each other.
- `contact_person` is optional. When provided, it must already exist in PostgreSQL and belong to the same client.
- `author` should contain the external user UUID for the incoming ticket.
- If you send `sync_key`, the API treats repeated deliveries from the same `source` as the same ticket and returns the existing ticket instead of creating a duplicate.
- If `sync_key` is present and `source` is omitted, the API uses `external-sync` automatically.
- If that external user does not exist yet, include `author_title` so the API can create or update the external user record.
- For backward compatibility, the webhook still accepts `external_author_id` and `external_author_title`, but `author` and `author_title` are now the preferred fields.

## Enable MinIO object storage

The backend now supports the MinIO Go SDK for S3-compatible object storage. Storage stays disabled unless `MINIO_*` variables are configured.

1. Start MinIO locally:
   `docker compose up -d minio`
2. Uncomment or add these values in `server/.env`:
   `MINIO_ENDPOINT=localhost:9000`
   `MINIO_ACCESS_KEY=<local-storage-access-key>`
   `MINIO_SECRET_KEY=<local-storage-secret-key>`
   `MINIO_BUCKET=<local-storage-bucket>`
   `MINIO_USE_SSL=false`
   `MINIO_REGION=<local-storage-region>`
3. Start the API from `server/`:
   `go run .`

Notes:

- On startup, the API creates a MinIO client and verifies the configured bucket exists.
- If the bucket is missing, the API creates it automatically with the configured region.
- The health endpoint at `/api/health` now reports storage configuration state alongside the database status.

## Prepare sqlc

The backend now includes `sqlc` input files under `server/db/`.

1. Install the required tools:
   `cd server && go get github.com/jackc/pgx/v5 github.com/jackc/pgx/v5/stdlib && go mod tidy`
   `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
2. Generate Go code:
   `cd server && sqlc generate`

Generated files will be written to `server/internal/db/`.

## Import legacy data

The README used to document every legacy import step separately, but most of that content repeated the same setup and command shape. The short version is:

1. Make sure PostgreSQL is running and `server/.env` contains the database settings.
2. Use a legacy CouchDB `_all_docs` export, or place it at `server/dump.json`.
3. Run the orchestrator from `server/`.

Import the whole dump:

```sh
cd server
go run ./cmd/import-dump -default-password "<temporary-password>"
```

Import only selected steps:

```sh
cd server
go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only users,tickets -default-password "<temporary-password>"
```

List the available step names:

```sh
cd server
go run ./cmd/import-dump -list
```

Available steps:

- `regions`
- `clients`
- `contacts`
- `research-types`
- `manufacturers`
- `classificators`
- `devices`
- `ticket-statuses`
- `ticket-types`
- `ticket-reasons`
- `users`
- `external-users`
- `tickets`
- `attachments`
- `agreements`

Useful flags:

- `-only step1,step2` runs a subset in the order defined by the importer.
- `-dry-run` validates and prints the plan without writing to PostgreSQL.
- `-keep-going` continues after a failed step.
- `-default-password` is required when the selection includes `users`.
- `-timeout` overrides the per-step timeout.
- `-per-user-timeout` only affects the `users` step.

Useful notes:

- If no `-source` is provided, the importer looks for `dump.json` and then `server/dump.json`.
- The full import keeps dependency order for you, so running multiple related steps together is safer than invoking them one by one.
- The importers are designed to be rerunnable; most steps update existing rows in place.
- `ticket-statuses` and `ticket-types` seed the canonical app values rather than preserving arbitrary legacy rows.
- `server/dump.json` is ignored by git, but if it was already tracked locally you still need to untrack it once with `git rm --cached server/dump.json`.

## Run the client

1. Install dependencies:
   `cd client && npm install`
2. Start Vite:
   `npm run dev`

The Vite dev server proxies `/api/*` to `http://localhost:8080`.
