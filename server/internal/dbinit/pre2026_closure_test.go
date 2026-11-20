package dbinit

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestPre2026Closure(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL closure regression test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	exec := func(sql string) {
		t.Helper()
		if _, err := tx.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	schema := pgx.Identifier{fmt.Sprintf("closure_test_%d", time.Now().UnixNano())}.Sanitize()
	exec("CREATE SCHEMA " + schema)
	exec("SET LOCAL search_path TO " + schema)
	exec(`CREATE TABLE tickets (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(), created_at TIMESTAMP,
		status TEXT, closed_at TIMESTAMP, app_version BIGINT NOT NULL DEFAULT 0,
		result TEXT DEFAULT 'original report', double_signed BOOLEAN DEFAULT false
	);
	INSERT INTO tickets (created_at, status, closed_at, app_version) VALUES
		('2025-12-31 23:59:59.999999', 'assigned', NULL, 3),
		('2025-01-01', 'cancelled', NULL, 0),
		('2025-01-01', 'inWork', NULL, 0),
		('2025-01-01', 'worksDone', '2025-12-20', 2),
		('2025-01-01', NULL, NULL, 0),
		('2025-01-01', 'closed', '2025-12-01', 4),
		('2026-01-01', 'assigned', NULL, 0),
		('2026-02-01', 'cancelled', NULL, 0),
		(NULL, 'assigned', NULL, 0);
	CREATE TEMP TABLE before_closure AS SELECT * FROM tickets;`)
	migration, err := os.ReadFile("../../db/schema/028_close_pre2026_tickets.sql")
	if err != nil {
		t.Fatal(err)
	}
	exec(string(migration))
	exec(`DO $$ BEGIN
		IF (SELECT count(*) FROM ticket_pre2026_closure_backup) <> 5 THEN
			RAISE EXCEPTION 'Incorrect backup count';
		END IF;
		IF EXISTS (
			SELECT 1 FROM tickets t JOIN before_closure b USING (id)
			WHERE b.created_at < TIMESTAMP '2026-01-01' AND b.status IS DISTINCT FROM 'closed'
			AND (t.status IS DISTINCT FROM 'closed' OR t.closed_at IS NULL
				OR t.app_version <> b.app_version + 1
				OR (b.closed_at IS NOT NULL AND t.closed_at IS DISTINCT FROM b.closed_at)
				OR t.result IS DISTINCT FROM b.result OR t.double_signed IS DISTINCT FROM b.double_signed)
		) THEN RAISE EXCEPTION 'Closure lost existing data or did not advance version'; END IF;
		IF EXISTS (
			SELECT 1 FROM tickets t JOIN before_closure b USING (id)
			WHERE (b.created_at >= TIMESTAMP '2026-01-01' OR b.created_at IS NULL OR b.status = 'closed')
			AND to_jsonb(t) IS DISTINCT FROM to_jsonb(b)
		) THEN RAISE EXCEPTION 'Unrelated ticket changed'; END IF;
		IF EXISTS (
			SELECT 1 FROM ticket_pre2026_closure_backup s JOIN before_closure b ON s.ticket_id = b.id
			WHERE s.snapshot IS DISTINCT FROM to_jsonb(b)
		) THEN RAISE EXCEPTION 'Backup differs from original row'; END IF;
	END $$;
	CREATE TEMP TABLE after_closure AS SELECT * FROM tickets;
	CREATE TEMP TABLE after_backup AS SELECT * FROM ticket_pre2026_closure_backup;`)
	exec(string(migration))
	exec(`DO $$ BEGIN
		IF EXISTS (SELECT * FROM tickets EXCEPT SELECT * FROM after_closure)
			OR EXISTS (SELECT * FROM after_closure EXCEPT SELECT * FROM tickets)
			OR EXISTS (SELECT * FROM ticket_pre2026_closure_backup EXCEPT SELECT * FROM after_backup)
			OR EXISTS (SELECT * FROM after_backup EXCEPT SELECT * FROM ticket_pre2026_closure_backup)
		THEN RAISE EXCEPTION 'Replay changed tickets or backups'; END IF;
	END $$;`)
}
