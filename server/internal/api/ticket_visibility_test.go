package api

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestTicketVisibility(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL visibility regression tests")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	// Temporary tables isolate these fixtures from any application data.
	_, err = tx.Exec(ctx, `
		CREATE TEMP TABLE accounts (user_id integer PRIMARY KEY, disabled boolean NOT NULL);
		CREATE TEMP TABLE tickets (id integer PRIMARY KEY, executor integer, author integer, status text, client text);
		INSERT INTO accounts VALUES (1, false), (2, true);
		INSERT INTO tickets VALUES
		  (1, 1, 2, 'assigned', 'active'),
		  (2, 2, 1, 'assigned', 'disabled-only'),
		  (3, 2, 1, 'closed', 'disabled-only'),
		  (4, NULL, 2, 'created', 'unassigned');
	`)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile("../../db/schema/027_visible_tickets.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, string(schema)); err != nil {
		t.Fatal(err)
	}
	assertIDs := func(want []int) {
		t.Helper()
		rows, err := tx.Query(ctx, `SELECT id FROM visible_tickets ORDER BY id`)
		if err != nil {
			t.Fatal(err)
		}
		got, err := pgx.CollectRows(rows, pgx.RowTo[int])
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("visible tickets = %v, want %v", got, want)
		}
	}
	assertIDs([]int{1, 4})
	var total, stored int
	var hiddenDetail, hiddenFacet, hiddenLatest bool
	err = tx.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM visible_tickets),
		       (SELECT count(*) FROM tickets),
		       EXISTS (SELECT 1 FROM visible_tickets WHERE id = 2),
		       EXISTS (SELECT DISTINCT client FROM visible_tickets WHERE client = 'disabled-only'),
		       EXISTS (SELECT 1 FROM (VALUES (2)) u(latest_ticket)
		               JOIN visible_tickets lt ON lt.id = u.latest_ticket)
	`).Scan(&total, &stored, &hiddenDetail, &hiddenFacet, &hiddenLatest)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || stored != 4 || hiddenDetail || hiddenFacet || hiddenLatest {
		t.Fatalf("unexpected visibility: total=%d stored=%d detail=%v facet=%v latest=%v", total, stored, hiddenDetail, hiddenFacet, hiddenLatest)
	}
	if _, err := tx.Exec(ctx, `UPDATE accounts SET disabled = false WHERE user_id = 2`); err != nil {
		t.Fatal(err)
	}
	assertIDs([]int{1, 2, 3, 4})
	if _, err := tx.Exec(ctx, `UPDATE accounts SET disabled = true WHERE user_id = 1`); err != nil {
		t.Fatal(err)
	}
	assertIDs([]int{2, 3, 4})
	if _, err := tx.Exec(ctx, `UPDATE tickets SET executor = 1 WHERE id = 2`); err != nil {
		t.Fatal(err)
	}
	assertIDs([]int{3, 4})
}
