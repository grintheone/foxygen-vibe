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
	auth               authConfig
}

type accountStore interface {
	CreateAccount(context.Context, appdb.CreateAccountParams) (appdb.Account, error)
	CreateUserProfile(context.Context, pgtype.UUID) (appdb.User, error)
	GetAccountByUsername(context.Context, string) (appdb.Account, error)
	GetAccountByUserID(context.Context, pgtype.UUID) (appdb.Account, error)
	GetUserProfileByUserID(context.Context, pgtype.UUID) (appdb.GetUserProfileByUserIDRow, error)
	CreateRefreshToken(context.Context, appdb.CreateRefreshTokenParams) (appdb.RefreshToken, error)
	GetRefreshTokenByHash(context.Context, string) (appdb.RefreshToken, error)
	RotateRefreshToken(context.Context, appdb.RotateRefreshTokenParams) (int64, error)
}

func New() (*Server, error) {
	databaseURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	auth, err := resolveAuthConfig()
	if err != nil {
		return nil, err
	}

	api := &Server{
		databaseConfigured: databaseURL != "",
		auth:               auth,
	}
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
	mux.HandleFunc("/api/auth/refresh", s.handleRefresh)
	mux.HandleFunc("/api/auth/session", s.handleSession)
	mux.HandleFunc("/api/profile", s.handleProfile)
	mux.HandleFunc("/api/tickets", s.handleTickets)
	mux.HandleFunc("/api/tickets/department", s.handleDepartmentTickets)

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
