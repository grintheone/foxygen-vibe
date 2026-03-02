package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	appdb "foxygen-vibe/server/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	databaseConfigured bool
	db                 *pgxpool.Pool
	queries            accountStore
}

type accountStore interface {
	CreateAccount(context.Context, appdb.CreateAccountParams) (appdb.Account, error)
	CreateUserProfile(context.Context, pgtype.UUID) (appdb.User, error)
	GetAccountByUsername(context.Context, string) (appdb.Account, error)
}

func New() (*Server, error) {
	databaseURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	api := &Server{databaseConfigured: databaseURL != ""}
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
	api.queries = appdb.New(db)
	if err := api.ensureSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return api, nil
}

func (s *Server) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/accounts", s.handleAccounts)
	mux.HandleFunc("/api/auth/login", s.handleLogin)

	return withRequestLogging(withCORS(mux))
}

func (s *Server) ensureSchema(ctx context.Context) error {
	schemaFiles, err := filepath.Glob("db/schema/*.sql")
	if err != nil {
		return fmt.Errorf("list schema files: %w", err)
	}

	sort.Strings(schemaFiles)

	for _, schemaFile := range schemaFiles {
		schema, readErr := os.ReadFile(schemaFile)
		if readErr != nil {
			return fmt.Errorf("read schema %s: %w", schemaFile, readErr)
		}

		if _, execErr := s.db.Exec(ctx, string(schema)); execErr != nil {
			return fmt.Errorf("apply schema %s: %w", schemaFile, execErr)
		}
	}

	return nil
}
