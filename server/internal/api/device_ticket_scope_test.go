package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Exercise the actual list/count/facet queries against isolated PostgreSQL fixtures.
func TestDeviceTicketScopes(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL scope regression tests")
	}
	ctx := context.Background()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 1 // Keep all temporary fixtures on the same connection.
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(ctx, `
 CREATE TEMP TABLE users (user_id uuid, department uuid, first_name text, last_name text);
 CREATE TEMP TABLE departments (id uuid, title text);
 CREATE TEMP TABLE clients (id uuid, title text, address text);
 CREATE TEMP TABLE devices (id uuid, classificator uuid, serial_number text);
 CREATE TEMP TABLE classificators (id uuid, title text);
 CREATE TEMP TABLE ticket_reasons (id text, title text, future text, past text, present text);
 CREATE TEMP TABLE tickets (
   id uuid, number integer, device uuid, client uuid, executor uuid, department uuid,
   reason text, status text DEFAULT 'closed', description text DEFAULT '', result text,
   urgent boolean DEFAULT false, external_author uuid, assigned_end timestamp,
   workstarted_at timestamp, workfinished_at timestamp, closed_at timestamp);
 CREATE TEMP TABLE accounts (user_id uuid, disabled boolean DEFAULT false);
 INSERT INTO users (user_id, department) VALUES
 ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000010'),
 ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000010'),
 ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000020'),
 ('00000000-0000-0000-0000-000000000004', NULL);
 INSERT INTO clients (id, title) SELECT ('00000000-0000-0000-0000-' || lpad(i::text,12,'0'))::uuid, 'Client ' || i FROM generate_series(1,6) i;
 INSERT INTO ticket_reasons (id,title) SELECT i::text, 'Reason ' || i FROM generate_series(1,6) i;
 INSERT INTO tickets (id,number,device,client,executor,department,reason)
 SELECT ('00000000-0000-0000-0000-' || lpad(i::text,12,'0'))::uuid, i,
   CASE WHEN i = 6 THEN '00000000-0000-0000-0000-000000000099' ELSE '00000000-0000-0000-0000-000000000050' END::uuid,
   ('00000000-0000-0000-0000-' || lpad(i::text,12,'0'))::uuid,
   CASE WHEN i = 4 THEN NULL WHEN i = 5 THEN '00000000-0000-0000-0000-000000000005' ELSE ('00000000-0000-0000-0000-' || lpad(LEAST(i,3)::text,12,'0')) END::uuid,
   CASE WHEN i IN (1,2,4,5,6) THEN '00000000-0000-0000-0000-000000000010' ELSE '00000000-0000-0000-0000-000000000020' END::uuid,
   i::text FROM generate_series(1,6) i;
 INSERT INTO accounts VALUES ('00000000-0000-0000-0000-000000000005', true);
 `)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile("../../db/schema/027_visible_tickets.sql")
	if err != nil {
		t.Fatal(err)
	}
	// Create the production view in this connection's temporary schema.
	if _, err = db.Exec(ctx, strings.Replace(string(schema), "CREATE OR REPLACE VIEW", "CREATE TEMP VIEW", 1)); err != nil {
		t.Fatal(err)
	}
	srv := &Server{db: db, auth: testAuthConfig()}
	request := func(path, user string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+editorAccessToken(t, user))
		rec := httptest.NewRecorder()
		srv.handleDeviceByID(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", path, rec.Code, rec.Body.String())
		}
		return rec
	}
	for _, tc := range []struct {
		scope   string
		user    string
		numbers []int
	}{
		{"", "1", []int{4, 3, 2, 1}},
		{"all", "1", []int{4, 3, 2, 1}},
		{"mine", "1", []int{1}},
		{"department", "1", []int{4, 2, 1}},
		{"department", "4", []int{}},
	} {
		t.Run(tc.scope+tc.user, func(t *testing.T) {
			user := "00000000-0000-0000-0000-00000000000" + tc.user
			path := "/api/devices/00000000-0000-0000-0000-000000000050/tickets"
			var page struct {
				Items   []struct{ Number int }
				Total   int
				HasNext bool
			}
			if err := json.Unmarshal(request(path+"?scope="+tc.scope, user).Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
			numbers := []int{}
			for _, item := range page.Items {
				numbers = append(numbers, item.Number)
			}
			if !reflect.DeepEqual(numbers, tc.numbers) || page.Total != len(tc.numbers) {
				t.Fatalf("numbers %v total %d, want %v", numbers, page.Total, tc.numbers)
			}
			var facets ticketArchiveFacetsResponse
			if err := json.Unmarshal(request(path+"/facets?scope="+tc.scope, user).Body.Bytes(), &facets); err != nil {
				t.Fatal(err)
			}
			if len(facets.Clients) != len(tc.numbers) || len(facets.ReasonTitles) != len(tc.numbers) {
				t.Fatalf("facets do not match scope: %+v", facets)
			}
			if len(tc.numbers) > 1 {
				if err := json.Unmarshal(request(path+"?scope="+tc.scope+"&limit=1&offset=1", user).Body.Bytes(), &page); err != nil {
					t.Fatal(err)
				}
				if page.Total != len(tc.numbers) || len(page.Items) != 1 || page.Items[0].Number != tc.numbers[1] || !page.HasNext {
					t.Fatalf("incorrect pagination: %+v", page)
				}
			}
		})
	}
}
