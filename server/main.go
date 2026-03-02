package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type message struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

type server struct {
	databaseConfigured bool
	db                 *pgxpool.Pool
}

func main() {
	api, err := newServer()
	if err != nil {
		log.Fatal(err)
	}
	if api.db != nil {
		defer api.db.Close()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", api.handleHealth)
	mux.HandleFunc("/api/message", api.handleMessage)

	addr := ":8080"
	log.Printf("server listening on %s", addr)

	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func newServer() (*server, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	api := &server{databaseConfigured: databaseURL != ""}
	if databaseURL == "" {
		return api, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}

	api.db = db
	if err := api.ensureSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return api, nil
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Status   string `json:"status"`
		Database struct {
			Configured bool `json:"configured"`
			Connected  bool `json:"connected"`
		} `json:"database"`
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload := response{Status: "ok"}
	payload.Database.Configured = s.databaseConfigured
	payload.Database.Connected = s.db != nil

	writeJSON(w, http.StatusOK, payload)
}

func (s *server) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	payload := message{
		ID:      0,
		Content: "Hello from the Go API.",
	}

	if s.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		const query = `
			SELECT id, content
			FROM messages
			ORDER BY id DESC
			LIMIT 1
		`

		if err := s.db.QueryRow(ctx, query).Scan(&payload.ID, &payload.Content); err != nil {
			log.Printf("message query failed: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, payload)
}

func (s *server) ensureSchema(ctx context.Context) error {
	const schema = `
		CREATE TABLE IF NOT EXISTS messages (
			id BIGSERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`

	if _, err := s.db.Exec(ctx, schema); err != nil {
		return err
	}

	const seed = `
		INSERT INTO messages (content)
		SELECT 'Hello from PostgreSQL.'
		WHERE NOT EXISTS (SELECT 1 FROM messages)
	`

	_, err := s.db.Exec(ctx, seed)
	return err
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
