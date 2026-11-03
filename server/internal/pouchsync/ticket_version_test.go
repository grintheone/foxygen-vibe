package pouchsync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestParseTicketAppVersion(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		want    int64
		invalid bool
	}{
		{"", 0, false}, {"0", 0, false}, {"7", 7, false},
		{"null", 0, true}, {" null ", 0, true}, {"-1", 0, true},
		{`"7"`, 0, true}, {"1.5", 0, true}, {"9223372036854775808", 0, true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := parseTicketAppVersion(json.RawMessage(tc.raw))
			if (err != nil) != tc.invalid || got != tc.want {
				t.Fatalf("got %d, %v", got, err)
			}
		})
	}
}

// Uses a disposable schema and rolls back all test data and DDL.
func TestTicketVersionPostgres(t *testing.T) {
	dsn := os.Getenv("TICKET_VERSION_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TICKET_VERSION_TEST_DATABASE_URL to run PostgreSQL regression tests")
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
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec("CREATE SCHEMA ticket_version_regression")
	exec("SET LOCAL search_path TO ticket_version_regression, public")
	files, err := filepath.Glob("../../db/schema/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) >= "020" {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		exec(string(body))
	}
	exec("INSERT INTO ticket_statuses (type, title) VALUES ('worksDone', 'done'), ('closed', 'closed')")
	exec("INSERT INTO tickets (id, status) VALUES ('11111111-1111-1111-1111-111111111111', 'closed')")
	migration, err := os.ReadFile("../../db/schema/025_ticket_app_version.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(migration))
	exec(string(migration)) // Bootstrap replays migrations; this must be safe.
	var version int64
	if err := tx.QueryRow(ctx, "SELECT app_version FROM tickets WHERE id = '11111111-1111-1111-1111-111111111111'").Scan(&version); err != nil || version != 1 {
		t.Fatalf("closed-ticket backfill: %d %v", version, err)
	}
	for i, numbered := range []bool{false, true} {
		t.Run(fmt.Sprintf("numbered=%v", numbered), func(t *testing.T) {
			id := fmt.Sprintf("22222222-2222-2222-2222-%012d", i+1)
			body := `{"status":"worksDone","result":"legacy"}`
			if numbered {
				body = `{"status":"worksDone","result":"legacy","number":100}`
			}
			change := changeEvent{ID: "ticket_" + id, Doc: []byte(body)}
			apply := func() {
				t.Helper()
				if err := applyTicket(ctx, tx, change); err != nil {
					t.Fatal(err)
				}
			}
			apply() // Baseline legacy insert and update remain supported.
			apply()
			exec("UPDATE tickets SET status = 'closed', result = 'final report', closed_at = '2026-11-03', double_signed = true, app_version = app_version + 1 WHERE id = $1", id)
			for _, incoming := range []string{"", `,"appVersion":0`, `,"appVersion":2`} {
				change.Doc = []byte(strings.TrimSuffix(body, "}") + incoming + "}")
				apply()
				var status, result string
				var closed, signed bool
				if err := tx.QueryRow(ctx, "SELECT status, result, closed_at IS NOT NULL, double_signed FROM tickets WHERE id = $1", id).Scan(&status, &result, &closed, &signed); err != nil {
					t.Fatal(err)
				}
				if status != "closed" || result != "final report" || !closed || !signed {
					t.Fatalf("stale update changed closed ticket: %s %s %v %v", status, result, closed, signed)
				}
			}
			change.Deleted = true
			apply() // Unversioned tombstones cannot remove app edits.
			change.Deleted = false
			change.Doc = []byte(strings.TrimSuffix(body, "}") + `,"appVersion":1}`)
			apply() // An explicitly matching version can make a legitimate external edit.
			var status string
			if err := tx.QueryRow(ctx, "SELECT status, app_version FROM tickets WHERE id = $1", id).Scan(&status, &version); err != nil || status != "worksDone" || version != 1 {
				t.Fatalf("matching version: %s %d %v", status, version, err)
			}
		})
	}
}
