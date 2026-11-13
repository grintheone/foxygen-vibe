package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestArchiveSearchValidation(t *testing.T) {
	auth := testAuthConfig()
	token, err := signJWT(auth.jwtSecret, accessTokenClaims{
		Subject: "11111111-1111-1111-1111-111111111111", TokenType: accessTokenType,
		ExpiresAt: time.Now().Add(time.Minute).Unix(), IssuedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth}
	for _, tc := range []struct {
		name, method, query, token string
		status                     int
	}{
		{"anonymous", "GET", "test", "", http.StatusUnauthorized},
		{"method", "POST", "test", token, http.StatusMethodNotAllowed},
		{"empty", "GET", "  ", token, http.StatusBadRequest},
		{"short Cyrillic", "GET", "Я", token, http.StatusBadRequest},
		{"long", "GET", strings.Repeat("я", 101), token, http.StatusBadRequest},
		{"valid Cyrillic reaches database", "GET", "ЯЯ", token, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/archive-search?q="+url.QueryEscape(tc.query), nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.status, rec.Body.String())
			}
		})
	}
}

func TestArchiveSearchSQL(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL search tests")
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
	_, err = tx.Exec(ctx, `
	 CREATE TEMP TABLE clients (id int, title text, address text, region int);
	 CREATE TEMP TABLE regions (id int, title text);
	 CREATE TEMP TABLE devices (id int, classificator int, serial_number text, properties jsonb);
	 CREATE TEMP TABLE classificators (id int, title text, manufacturer int);
	 CREATE TEMP TABLE manufacturers (id int, title text);
	 INSERT INTO regions VALUES (1, 'Москва');
	 INSERT INTO clients VALUES (1, 'Клиника Альфа', 'Ленина 15', 1), (2, 'Clinic 100%_test', NULL, NULL);
	 INSERT INTO manufacturers VALUES (1, 'Acme');
	 INSERT INTO classificators VALUES (1, 'Analyzer', 1);
	 INSERT INTO devices VALUES (1, 1, 'SN-123', '{"inventory":"INV-42"}'), (2, NULL, 'ORPHAN', '{}');
	`)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		query, kind, id string
	}{
		{"КЛИНИКА Москва", "client", "1"},
		{"Ленина 15", "client", "1"},
		{"100%_", "client", "2"},
		{"Analyzer SN-123", "device", "1"},
		{"acme inv-42", "device", "1"},
		{"ORPHAN", "device", "2"},
		{"Клиника missing", "", ""},
		{"%' OR true --", "", ""},
	} {
		t.Run(tc.query, func(t *testing.T) {
			rows, err := tx.Query(ctx, archiveSearchSQL, strings.Fields(strings.ToLower(tc.query)))
			if err != nil {
				t.Fatal(err)
			}
			items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (archiveSearchItem, error) {
				var item archiveSearchItem
				err := row.Scan(&item.Kind, &item.ID, &item.Title, &item.Subtitle)
				return item, err
			})
			if err != nil {
				t.Fatal(err)
			}
			if tc.kind == "" {
				if len(items) != 0 {
					t.Fatalf("unexpected matches: %+v", items)
				}
			} else if len(items) != 1 || items[0].Kind != tc.kind || items[0].ID != tc.id {
				t.Fatalf("unexpected matches: %+v", items)
			}
		})
	}
	_, err = tx.Exec(ctx, `
	 INSERT INTO clients SELECT n, 'Common', '', NULL FROM generate_series(100, 130) n;
	 INSERT INTO devices SELECT n, 1, 'Common', '{}' FROM generate_series(100, 130) n;
	`)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := tx.Query(ctx, archiveSearchSQL, []string{"common"})
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var item archiveSearchItem
		if err := rows.Scan(&item.Kind, &item.ID, &item.Title, &item.Subtitle); err != nil {
			t.Fatal(err)
		}
		counts[item.Kind]++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if counts["client"] != 21 || counts["device"] != 21 {
		t.Fatalf("independent result limits: %v", counts)
	}
}
