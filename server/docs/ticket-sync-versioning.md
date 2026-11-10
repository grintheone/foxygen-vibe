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

## Timestamp contract

PostgreSQL ticket timestamp columns contain UTC wall-clock values. API events
broadcast through Sidecar and `foxygen.latestTicket` emit these instants with a
`Z` suffix. Consumers must preserve the instant: retain `Z`/an explicit offset,
or use Unix milliseconds. Do not strip `Z` and then echo the UTC clock value as
an offset-free local time.

Live PouchDB sync and the legacy dump importer share a parser:

- Unix milliseconds and strings with `Z` or an explicit offset preserve their instant.
- Offset-free strings and date-only strings represent Moscow Standard Time
  (UTC+03:00), and are converted to UTC before storage.
- This applies to creation, assignment, planned/assigned intervals, work start/end,
  and closure derived from the actual interval.

For example, legacy `2026-11-09T12:00:00` is stored as `09:00:00` UTC and emitted
as `2026-11-09T09:00:00Z`; the app displays `12:00` Moscow time. The previous
parser stored it as `12:00:00` UTC, causing the app to display `15:00`.

Deploying the parser corrects subsequent imports and sync writes. It does not
rewrite existing rows or rewind the sync checkpoint. Historical repair must
compare each affected field with its original source value and respect app
versions/edits. Do not subtract three hours from all rows: Unix/offset-bearing
source values and app-created timestamps were already absolute UTC instants.
A full legacy reimport can overwrite later app edits and must not be used as a
blanket timestamp repair.
