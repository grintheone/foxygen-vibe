# Foxygen Vibe

Minimal fullstack starter:

- `client/`: React + Tailwind (Vite)
- `server/`: Go API with PostgreSQL-ready environment wiring
- `docker-compose.yml`: local PostgreSQL and MinIO services
- `docker-compose.production.yml`: production-oriented full stack with frontend, API, PostgreSQL, MinIO, and first-start import bootstrap

## Run the production stack with Docker

The production compose file builds and runs:

- the React frontend behind Nginx
- the Go API
- PostgreSQL
- an optional MinIO container profile for local/self-hosted object storage

On container startup the API bootstrap step:

- waits for PostgreSQL
- applies every SQL file in `server/db/schema/`
- waits for MinIO when storage is configured
- runs `server/cmd/import-dump` only if the database is still empty

Setup:

1. Put the legacy dump at `deploy/production/bootstrap/dump.json`.
2. Create `deploy/production/.env` from `deploy/production/.env.example`.
3. In `deploy/production/.env`, change at least:
   `POSTGRES_PASSWORD`
   `JWT_SECRET`
   `IMPORT_DEFAULT_PASSWORD`
4. Keep the provided company object-storage values unless your infra team gives you newer ones.
5. Start the stack:
   `docker compose --env-file deploy/production/.env -f docker-compose.production.yml up -d --build`

For your current company setup, the object-storage section should look like this:

`MINIO_ENDPOINT=s3.internal.int.best`
`MINIO_ACCESS_KEY=3L8BYP2lYjnGokfkaxX2`
`MINIO_SECRET_KEY=RcnbKT1i9QpPi59Xc7Ejg8gRLD2otbd5t3fpRR2O`
`MINIO_BUCKET=mobile-engineer`
`MINIO_USE_SSL=true`
`MINIO_LOCATION=ru-sp-01`

If you want to run your own MinIO container instead of an external S3/MinIO endpoint, start with:

`docker compose --profile with-minio --env-file deploy/production/.env -f docker-compose.production.yml up -d --build`

Useful notes:

- The frontend is exposed on `PUBLIC_HTTP_PORT` from the env file and proxies `/api/*` to the backend container.
- PostgreSQL stays on the internal Docker network by default, which is safer for a company server.
- For object storage, set `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, and either `MINIO_REGION` or `MINIO_LOCATION`.
- `MINIO_ENDPOINT` can be a bare hostname. Bootstrap will default to port `443` when `MINIO_USE_SSL=true`, otherwise `80`.
- The first-start import is controlled by `BOOTSTRAP_IMPORT_ENABLED=true|false`.
- Imported users receive the temporary password from `IMPORT_DEFAULT_PASSWORD`.
- If the database already contains app data, the bootstrap step skips the import and starts the API normally.
- Your current production path uses the company object storage endpoint and does not need the local `with-minio` profile.

## Run the server

1. Start PostgreSQL:
   `docker compose up -d postgres`
2. Start the API:
   `cd server && go run .`

The server reads `server/.env` for `DB_*` settings and builds a PostgreSQL connection string from that file. Explicit shell environment variables still override values from `.env`, and `DATABASE_URL` still takes precedence over the split fields.

## Receive synced tickets from an external system

The backend now exposes a webhook-style endpoint at `/api/v1/sync` for server-to-server ticket creation.

1. Set a shared secret in `server/.env`:
   `TICKET_SYNC_SECRET=replace-with-a-long-random-secret`
2. Start the API from `server/`:
   `go run .`
3. Send `POST` requests to your server with the `X-Sync-Secret` header and JSON like:
   ```json
   {
     "author": "00000000-0000-0000-0000-000000000004",
     "author_title": "External Dispatcher",
     "client": "00000000-0000-0000-0000-000000000001",
     "device": "00000000-0000-0000-0000-000000000002",
     "contact_person": "00000000-0000-0000-0000-000000000003",
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
- `client`, `device`, and `contact_person` must already exist in PostgreSQL and must be consistent with each other.
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
   `MINIO_ACCESS_KEY=minioadmin`
   `MINIO_SECRET_KEY=minioadmin`
   `MINIO_BUCKET=foxygen-vibe`
   `MINIO_USE_SSL=false`
   `MINIO_REGION=us-east-1`
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

## Import the full legacy dump

If you want to run the whole legacy import flow in one command, place the dump at `server/dump.json` and run the orchestrator from the `server/` directory:

`go run ./cmd/import-dump -default-password "ChangeMe123!"`

Notes:

- The command defaults to `dump.json`, so it will pick up `server/dump.json` automatically when run from `server/`.
- It runs the existing importers in dependency order: regions, clients, contacts, research types, manufacturers, classificators, devices, ticket metadata, users, tickets, attachments, then agreements.
- Use `-only users,tickets` to run a subset, `-dry-run` to validate without writes, and `-keep-going` if you want later steps to continue after a failure.
- `server/dump.json` is now ignored by git, but if it was already tracked in your local clone you still need to untrack it once with `git rm --cached server/dump.json`.

## Import legacy users

If you have a legacy CouchDB `_all_docs` export, you can import the `user_*` records into the current PostgreSQL schema:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only users -default-password "ChangeMe123!"`

Notes:

- The legacy dump does not contain passwords, so the importer assigns the temporary password you provide to every imported account.
- Imported usernames are assigned deterministically as `user_1`, `user_2`, `user_3`, and so on.
- The command is safe to rerun. If a username already exists, that user is skipped.
- Use `-dry-run` first if you want to inspect the planned usernames before writing to the database.

## Import legacy regions

If you have the same legacy CouchDB `_all_docs` export, you can import the `region_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only regions`

Notes:

- The importer creates the `regions` table if it does not exist yet.
- Legacy region UUIDs are preserved, which keeps later imports compatible with the old references.
- The command is safe to rerun. Existing region IDs are updated in place.
- Use `-dry-run` first if you want to inspect the planned region records before writing to the database.

## Import legacy clients

If you have the same legacy CouchDB `_all_docs` export, you can import the `client_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Import regions first so client region references can be resolved.
3. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only clients`

Notes:

- The importer creates the `clients` table if it does not exist yet.
- Legacy client UUIDs are preserved.
- The importer carries over the legacy `region` UUID only when that region already exists in PostgreSQL; otherwise it stores `NULL` and logs the mismatch.
- The command is safe to rerun. Existing client IDs are updated in place.
- Use `-dry-run` first if you want to inspect the planned client records before writing to the database.

## Import legacy contacts

If you have the same legacy CouchDB `_all_docs` export, you can import the `contact_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Import clients first so contact client references can be resolved.
3. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only contacts`

Notes:

- The importer creates the `contacts` table if it does not exist yet.
- Legacy contact UUIDs are preserved.
- The importer maps each contact `ref` to the imported `clients.id`; if the client is missing it stores `NULL` and logs the mismatch.
- The command is safe to rerun. Existing contact IDs are updated in place.
- Use `-dry-run` first if you want to inspect the planned contact records before writing to the database.

## Import legacy research types

If you have the same legacy CouchDB `_all_docs` export, you can import the `researchType_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only research-types`

Notes:

- The importer creates the `research_type` table if it does not exist yet.
- Legacy research type UUIDs are preserved.
- The command is safe to rerun. Existing rows are updated in place by `id`.

## Import legacy manufacturers

If you have the same legacy CouchDB `_all_docs` export, you can import the `manufacturer_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only manufacturers`

Notes:

- The importer creates the `manufacturers` table if it does not exist yet.
- Legacy manufacturer UUIDs are preserved.
- Manufacturer titles are not forced unique, matching the legacy schema.
- The command is safe to rerun. Existing rows are updated in place by `id`.

## Import legacy classificators

If you have the same legacy CouchDB `_all_docs` export, you can import the `classificator_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Import `research_type` and `manufacturers` first so classificator references can be resolved.
3. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only classificators`

Notes:

- The importer creates the `classificators` table if it does not exist yet.
- Legacy classificator UUIDs are preserved.
- The importer maps `manufacturer` and `researchType` to the imported PostgreSQL rows when present; missing legacy references are stored as `NULL` and logged.
- `registration_certificate` and `maintenance_regulations` are preserved as JSONB, and `attachments` and `images` are imported as text arrays.
- The command is safe to rerun. Existing rows are updated in place by `id`.

## Import legacy devices

If you have the same legacy CouchDB `_all_docs` export, you can import the `device_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Import `classificators` first so device classificator references can be resolved.
3. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only devices`

Notes:

- The importer creates the `devices` table if it does not exist yet.
- Legacy device UUIDs are preserved.
- The importer maps `classificator` to the imported PostgreSQL row when present; missing legacy references are stored as `NULL` and logged.
- `properties` is preserved as JSONB; `connected_to_lis` and `is_used` are imported as booleans.
- The command is safe to rerun. Existing rows are updated in place by `id`.

## Import legacy ticket statuses

If you have the same legacy CouchDB `_all_docs` export, you can import ticket statuses into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only ticket-statuses`

Notes:

- This importer enforces the canonical status set: `created`, `assigned`, `inWork`, `worksDone`, `closed`, `cancelled`.
- Titles are seeded as: `создан`, `назначен`, `в работе`, `работы завершены`, `закрыт`, `отменен`.
- Any other existing status rows are removed.

## Import legacy ticket types

If you have the same legacy CouchDB `_all_docs` export, you can import ticket types into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only ticket-types`

Notes:

- This importer enforces the canonical type set: `internal`, `external`.
- Titles are seeded as: `внутренний`, `внешний`.
- Any other existing type rows are removed.

## Import legacy ticket reasons

If you have the same legacy CouchDB `_all_docs` export, you can import the `ticketReason_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only ticket-reasons`

Notes:

- The importer creates the `ticket_reasons` table if it does not exist yet.
- Legacy reason ids are preserved.
- The command is safe to rerun. Existing rows are updated in place by `id`.

## Import legacy tickets

If you have the same legacy CouchDB `_all_docs` export, you can import the `ticket_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Import clients, devices, users/accounts, departments, contacts, ticket statuses, ticket types, and ticket reasons first.
3. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only tickets`

Notes:

- The importer creates the `tickets` table if it does not exist yet.
- Legacy ticket UUIDs are preserved.
- Legacy statuses and reasons are normalized to the current schema: `finished -> worksDone`, `repairs -> repair`, `maintenance/mainenance -> maintanence`, and `fast -> internal`.
- Missing foreign keys are stored as `NULL` and counted in the import summary.
- Legacy ticket numbers are preserved when present; blank numbers are generated by PostgreSQL and the identity sequence is advanced to the current max.

## Import legacy attachments

If you have the same legacy CouchDB `_all_docs` export, you can import the attachment objects embedded in `ticket_*` records into PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Import tickets first so attachment `ref_id` values can resolve.
3. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only attachments`

Notes:

- The importer creates the `attachments` table if it does not exist yet.
- Attachment `id`, `name`, `media_type`, and `ext` are copied directly from the ticket payload.
- `ref_id` is set to the imported legacy ticket UUID.
- If a referenced ticket is missing, that attachment is skipped and counted.

## Import legacy agreements

If you have the same legacy CouchDB `_all_docs` export, you can translate device `bindings` into `agreements` rows in PostgreSQL:

1. Make sure PostgreSQL is running and `server/.env` contains your database settings.
2. Import clients and devices first so binding references can resolve.
3. Run the importer from the `server/` directory:
   `go run ./cmd/import-dump -source "/absolute/path/to/_all_docs.json" -only agreements`

Notes:

- Each device binding produces one agreement with a deterministic synthetic UUID based on `device_id + client_id`, so reruns remain idempotent.
- `actual_client` is set from the binding client, `device` is set from the device, `distributor` is left `NULL`.
- Because bindings do not include dates or warranty flags, the importer defaults `assigned_at`/`finished_at` to `NULL`, `is_active` to `TRUE`, and `on_warranty` to `TRUE`.

## Run the client

1. Install dependencies:
   `cd client && npm install`
2. Start Vite:
   `npm run dev`

The Vite dev server proxies `/api/*` to `http://localhost:8080`.
