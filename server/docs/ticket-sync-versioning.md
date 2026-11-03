# Ticket synchronization version contract

Apply `db/schema/025_ticket_app_version.sql` before starting the updated API.
Production bootstrap applies schema files on startup. The migration is replay-safe
and protects existing closed tickets by assigning version 1 if still at version 0.
Existing non-closed tickets and legacy imports start at version 0.

App-created tickets (including recommendation follow-ups) start at version 1.
Every app lifecycle transition or editor save increments `tickets.app_version` in
the same SQL update as the ticket fields. Events expose it as `ticket.appVersion`.
Attachment uploads do not change ticket fields or increment the counter.

The external consumer must persist and echo `appVersion` from the snapshot it
actually used. It must not fetch a newer version and attach it to older contents.
For example, an attachment event at version 4 must retain version 4 even if a
later closure event arrives at version 5.

PouchDB ticket upserts compare incoming `appVersion` with the current database
version atomically in `ON CONFLICT ... DO UPDATE ... WHERE`. Only an exact match
can update an existing ticket. Older and future versions are skipped and logged;
the changes-feed checkpoint still advances. Sync does not increment this counter:
it tracks app writes, while the PouchDB feed orders external changes.

Missing `appVersion` means the legacy baseline (0). This preserves sync for
untouched legacy tickets, but unversioned updates cannot overwrite app-edited or
closed/backfilled tickets. Until the external consumer supports this contract,
those updates will be intentionally ignored. Non-integer, null, or negative
versions are rejected as malformed documents. PouchDB tombstones have no reliable
base version, so they can delete only baseline version-0 tickets.

This protects app changes from stale external snapshots. It is not general
multi-writer conflict resolution, and direct SQL writes or manual legacy reimports
bypass it. A matching-version external update may legitimately change the status,
including reopening a ticket. Previously published events cannot be retroactively
versioned.

Run the database regression tests against a disposable PostgreSQL database:

```sh
cd server
TICKET_VERSION_TEST_DATABASE_URL='postgres://user@localhost:5432/testdb?sslmode=disable' \
  go test ./internal/pouchsync ./internal/api
```

The PostgreSQL test creates a schema within a transaction and rolls it back.
Without this variable the PostgreSQL test skips; parser and event tests still run.
