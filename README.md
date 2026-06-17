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

## Set up daily PostgreSQL backups

The production database runs inside the `postgres` compose service, so the simplest reliable backup flow is:

1. Keep the stack running with `docker compose -f docker-compose.production.yml up -d`.
2. Run [`deploy/production/backup-db.sh`](/Users/grintheone/Dev/Projects/foxygen-v3.1/deploy/production/backup-db.sh) from the host on a daily cron schedule.
3. Store the generated files on persistent host storage and copy them off the machine if you need disaster recovery.

The script:

- Loads `deploy/production/.env`
- Runs `pg_dump` inside the `postgres` container
- Writes a compressed SQL backup to `deploy/production/backups/`
- Deletes backup files older than `BACKUP_KEEP_DAYS`

Optional settings in `deploy/production/.env`:

- `BACKUP_DIR=./deploy/production/backups`
- `BACKUP_PREFIX=foxygen-db`
- `BACKUP_KEEP_DAYS=14`

Run a backup manually:

```sh
./deploy/production/backup-db.sh
```

Example cron entry for a daily 02:15 backup on the Docker host:

```cron
15 2 * * * cd /path/to/foxygen-v3.1 && ./deploy/production/backup-db.sh >> /var/log/foxygen-db-backup.log 2>&1
```

Quick restore example from a compressed backup:

```sh
gzip -dc deploy/production/backups/foxygen-db_2026-05-15_02-15-00.sql.gz \
  | docker compose --env-file deploy/production/.env -f docker-compose.production.yml exec -T postgres \
      sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

For stronger disaster recovery, sync the backup directory to external storage after each run. Host-only backups protect against accidental deletes and bad migrations, but they do not protect against full-host loss.

## Run the server

1. Start PostgreSQL:
   `docker compose up -d postgres`
2. Start the API:
   `cd server && go run .`

The server reads `server/.env` for `DB_*` settings and builds a PostgreSQL connection string from that file. Explicit shell environment variables still override values from `.env`, and `DATABASE_URL` still takes precedence over the split fields.

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
