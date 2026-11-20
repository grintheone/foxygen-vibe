package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTicketCancellation(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL cancellation tests")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 1
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(ctx, `
		CREATE TEMP TABLE users (user_id uuid, department uuid);
		CREATE TEMP TABLE roles (id integer, name text);
		CREATE TEMP TABLE account_roles (user_id uuid, role_id integer);
		CREATE TEMP TABLE tickets (id uuid, department uuid, status text, app_version bigint, closed_at timestamp);
		INSERT INTO roles VALUES (1, 'coordinator');
		INSERT INTO users VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010');
		INSERT INTO account_roles VALUES ('00000000-0000-0000-0000-000000000001', 1);
	`)
	if err != nil {
		t.Fatal(err)
	}
	userID := mustUUID(t, "00000000-0000-0000-0000-000000000001")
	ticketID := mustUUID(t, "00000000-0000-0000-0000-000000000002")
	srv := &Server{db: db}
	for _, tc := range []struct {
		name, role, status, department string
		want                           int
	}{
		{"coordinator cancels new ticket", "coordinator", "created", "00000000-0000-0000-0000-000000000010", http.StatusOK},
		{"other department", "coordinator", "created", "00000000-0000-0000-0000-000000000020", http.StatusConflict},
		{"no department", "coordinator", "created", "", http.StatusForbidden},
		{"engineer", "user", "created", "00000000-0000-0000-0000-000000000010", http.StatusForbidden},
		{"admin", "admin", "created", "00000000-0000-0000-0000-000000000010", http.StatusForbidden},
		{"already assigned", "coordinator", "assigned", "00000000-0000-0000-0000-000000000010", http.StatusConflict},
		{"work started", "coordinator", "inWork", "00000000-0000-0000-0000-000000000010", http.StatusConflict},
		{"work done", "coordinator", "worksDone", "00000000-0000-0000-0000-000000000010", http.StatusConflict},
		{"closed", "coordinator", "closed", "00000000-0000-0000-0000-000000000010", http.StatusConflict},
		{"already cancelled", "coordinator", "cancelled", "00000000-0000-0000-0000-000000000010", http.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := db.Exec(ctx, `UPDATE roles SET name = $1`, tc.role); err != nil {
				t.Fatal(err)
			}
			var department pgtype.UUID
			if tc.department != "" {
				department = mustUUID(t, tc.department)
			}
			if _, err := db.Exec(ctx, `UPDATE users SET department = $1`, department); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(ctx, `DELETE FROM tickets`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(ctx, `INSERT INTO tickets VALUES ($1, '00000000-0000-0000-0000-000000000010', $2, 7, NULL)`, ticketID, tc.status); err != nil {
				t.Fatal(err)
			}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPatch, "/api/tickets/"+ticketID.String(), strings.NewReader(`{"status":"cancelled"}`))
			srv.handleTicketByIDPatch(rec, req, ticketID, userID)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.want, rec.Body.String())
			}
			var status string
			var version int64
			var closedAt pgtype.Timestamp
			if err := db.QueryRow(ctx, `SELECT status, app_version, closed_at FROM tickets`).Scan(&status, &version, &closedAt); err != nil {
				t.Fatal(err)
			}
			if tc.want == http.StatusOK {
				if status != "cancelled" || version != 8 || !closedAt.Valid {
					t.Fatalf("unexpected cancellation: %s, %d, %v", status, version, closedAt)
				}
				var response struct {
					Status   string `json:"status"`
					ClosedAt string `json:"closed_at"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if response.Status != "cancelled" || response.ClosedAt == "" {
					t.Fatalf("unexpected response: %s", rec.Body.String())
				}
			} else if status != tc.status || version != 7 || closedAt.Valid {
				t.Fatal("rejected cancellation modified ticket")
			}
		})
	}
}
