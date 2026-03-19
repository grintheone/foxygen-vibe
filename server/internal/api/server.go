package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	appdb "foxygen-vibe/server/internal/db"
	"foxygen-vibe/server/internal/dbinit"
	"foxygen-vibe/server/internal/storage"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var newMinIOClient = storage.NewMinIO

type Server struct {
	databaseConfigured        bool
	storageConfigured         bool
	db                        *pgxpool.Pool
	queries                   accountStore
	auth                      authConfig
	sync                      syncConfig
	storage                   *storage.Client
	editorAccessCheck         func(http.ResponseWriter, *http.Request) (pgtype.UUID, bool)
	editorRoleLookup          func(context.Context, pgtype.UUID) (string, error)
	editorClientDetailLoader  func(context.Context, pgtype.UUID) (editorClientDetailResponse, bool, error)
	editorContactDetailLoader func(context.Context, pgtype.UUID) (editorContactDetailResponse, bool, error)
	editorDeviceDetailLoader  func(context.Context, pgtype.UUID) (editorDeviceDetailResponse, bool, error)
	editorRegionExists        func(context.Context, pgtype.UUID) (bool, error)
	editorClientExists        func(context.Context, pgtype.UUID) (bool, error)
	editorClassificatorExists func(context.Context, pgtype.UUID) (bool, error)
	editorClientUpdater       func(context.Context, pgtype.UUID, string, string, any, any) (int64, error)
	editorContactUpdater      func(context.Context, pgtype.UUID, string, string, string, string, pgtype.UUID) (int64, error)
	editorDeviceUpdater       func(context.Context, pgtype.UUID, any, string, json.RawMessage, bool, bool) (int64, error)
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

	storageConfig, err := resolveStorageConfig()
	if err != nil {
		return nil, err
	}

	sync, err := resolveSyncConfig()
	if err != nil {
		return nil, err
	}

	api := &Server{
		databaseConfigured: databaseURL != "",
		storageConfigured:  storageConfig.Enabled(),
		auth:               auth,
		sync:               sync,
	}

	if storageConfig.Enabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		api.connectStorage(ctx, storageConfig)
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
	if err := dbinit.EnsureSchema(ctx, db, "db/schema/*.sql"); err != nil {
		db.Close()
		return nil, err
	}

	return api, nil
}

func (s *Server) connectStorage(ctx context.Context, config storage.Config) {
	client, err := newMinIOClient(ctx, config)
	if err != nil {
		log.Printf("object storage is configured but unavailable; continuing without it: %v", err)
		return
	}

	s.storage = client
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
	mux.HandleFunc("/api/editor/clients", s.handleEditorClients)
	mux.HandleFunc("/api/editor/clients/", s.handleEditorClientByID)
	mux.HandleFunc("/api/editor/classificators", s.handleEditorClassificators)
	mux.HandleFunc("/api/editor/contacts", s.handleEditorContacts)
	mux.HandleFunc("/api/editor/contacts/", s.handleEditorContactByID)
	mux.HandleFunc("/api/editor/devices", s.handleEditorDevices)
	mux.HandleFunc("/api/editor/devices/", s.handleEditorDeviceByID)
	mux.HandleFunc("/api/editor/regions", s.handleEditorRegions)
	mux.HandleFunc("/api/profile", s.handleProfile)
	mux.HandleFunc("/api/profile/", s.handleProfile)
	mux.HandleFunc("/api/clients/", s.handleClientByID)
	mux.HandleFunc("/api/devices/", s.handleDeviceByID)
	mux.HandleFunc("/api/comments", s.handleComments)
	mux.HandleFunc("/api/departments", s.handleDepartments)
	mux.HandleFunc("/api/departments/members", s.handleDepartmentMembers)
	mux.HandleFunc("/api/ticket-reasons", s.handleTicketReasons)
	mux.HandleFunc("/api/v1/sync", s.handleTicketSync)
	mux.HandleFunc("/api/tickets", s.handleTickets)
	mux.HandleFunc("/api/tickets/", s.handleTicketByID)
	mux.HandleFunc("/api/tickets/department", s.handleDepartmentTickets)

	return withRequestLogging(withCORS(mux))
}
