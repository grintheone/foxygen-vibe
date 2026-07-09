package pouchsync

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestUpsertTicketWithNumberDoesNotUpdateIdentityNumber(t *testing.T) {
	if !strings.Contains(upsertTicketWithNumberSQL, "OVERRIDING SYSTEM VALUE") {
		t.Fatal("expected numbered ticket insert to override the identity value")
	}
	if strings.Contains(upsertTicketWithNumberSQL, "SET number =") {
		t.Fatal("ticket identity number must not be updated on conflict")
	}
	if !strings.Contains(upsertTicketWithNumberSQL, "ON CONFLICT (id) DO UPDATE") {
		t.Fatal("expected numbered ticket insert to keep updating existing ticket fields")
	}
}

func TestUUIDLookupsSkipMalformedValuesBeforeQuery(t *testing.T) {
	ctx := context.Background()
	tx := panicQueryTx{}

	if got := existsByID(ctx, tx, "clients", "not-a-uuid"); got {
		t.Fatal("malformed UUID client id should not exist")
	}
	if got := resolveOptionalID(ctx, tx, "accounts", "legacy-user", "user_id"); got != nil {
		t.Fatalf("malformed UUID account id should resolve to nil, got %v", got)
	}
	authorID, externalAuthorID := resolveTicketAuthor(ctx, tx, "legacy-user")
	if authorID != nil || externalAuthorID != nil {
		t.Fatalf("malformed UUID author should resolve to nils, got author=%v external=%v", authorID, externalAuthorID)
	}
}

func TestTextLookupsDoNotRequireUUID(t *testing.T) {
	if columnRequiresUUID("ticket_statuses", "type") {
		t.Fatal("ticket status ids are textual lookup values")
	}
	if columnRequiresUUID("ticket_reasons", "id") {
		t.Fatal("ticket reason ids are textual lookup values")
	}
}

func TestParseTicketRecordMapsAssignedWithoutExecutorToCancelled(t *testing.T) {
	item, err := parseTicketRecord(changeEvent{
		ID: "ticket_1_aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Doc: []byte(`{
			"createdAt": "2024-01-02T03:04:05",
			"ticketType": "external",
			"executor": null,
			"status": "assigned"
		}`),
	})
	if err != nil {
		t.Fatalf("parseTicketRecord: %v", err)
	}

	if item.Status != "cancelled" || item.ExecutorID != "" {
		t.Fatalf("expected assigned ticket without executor to map to cancelled, got status=%q executor=%q", item.Status, item.ExecutorID)
	}
}

type panicQueryTx struct{}

func (panicQueryTx) Begin(context.Context) (pgx.Tx, error) {
	panic("unexpected Begin")
}

func (panicQueryTx) Commit(context.Context) error {
	panic("unexpected Commit")
}

func (panicQueryTx) Rollback(context.Context) error {
	panic("unexpected Rollback")
}

func (panicQueryTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	panic("unexpected CopyFrom")
}

func (panicQueryTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	panic("unexpected SendBatch")
}

func (panicQueryTx) LargeObjects() pgx.LargeObjects {
	panic("unexpected LargeObjects")
}

func (panicQueryTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	panic("unexpected Prepare")
}

func (panicQueryTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	panic("unexpected Exec")
}

func (panicQueryTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("unexpected Query")
}

func (panicQueryTx) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("unexpected QueryRow")
}

func (panicQueryTx) Conn() *pgx.Conn {
	panic("unexpected Conn")
}
