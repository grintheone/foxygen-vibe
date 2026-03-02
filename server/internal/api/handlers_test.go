package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appdb "foxygen-vibe/server/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeAccountCreator struct {
	account          appdb.Account
	createAccountErr error
	createUserErr    error
	getAccountErr    error
	params           appdb.CreateAccountParams
	accountCalled    bool
	userCalled       bool
	getCalled        bool
	username         string
	userID           pgtype.UUID
}

func (f *fakeAccountCreator) CreateAccount(_ context.Context, params appdb.CreateAccountParams) (appdb.Account, error) {
	f.accountCalled = true
	f.params = params
	return f.account, f.createAccountErr
}

func (f *fakeAccountCreator) CreateUserProfile(_ context.Context, userID pgtype.UUID) (appdb.User, error) {
	f.userCalled = true
	f.userID = userID
	return appdb.User{UserID: userID}, f.createUserErr
}

func (f *fakeAccountCreator) GetAccountByUsername(_ context.Context, username string) (appdb.Account, error) {
	f.getCalled = true
	f.username = username
	return f.account, f.getAccountErr
}

func TestHealthEndpointReturnsStatus(t *testing.T) {
	t.Parallel()

	srv := &Server{databaseConfigured: true}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{
		`"status":"ok"`,
		`"configured":true`,
		`"connected":false`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content type application/json, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS header to be set, got %q", got)
	}
}

func TestHealthEndpointRejectsNonGet(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestOptionsRequestReturnsNoContent(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("unexpected allow methods header %q", got)
	}
}

func TestMessageEndpointIsRemoved(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/message", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestAccountsEndpointRequiresDatabase(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestAccountsEndpointRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	srv := &Server{queries: &fakeAccountCreator{}}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":"alice","password":"secret","extra":true}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAccountsEndpointValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	srv := &Server{queries: &fakeAccountCreator{}}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":" ","password":" "}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAccountsEndpointCreatesAccount(t *testing.T) {
	t.Parallel()

	store := &fakeAccountCreator{
		account: appdb.Account{
			UserID:   pgtype.UUID{Bytes: [16]byte{1, 2, 3}, Valid: true},
			Username: "alice",
			Disabled: false,
		},
	}
	srv := &Server{queries: store}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":" alice ","password":" secret "}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if !store.accountCalled {
		t.Fatal("expected CreateAccount to be called")
	}
	if !store.userCalled {
		t.Fatal("expected CreateUserProfile to be called")
	}
	if store.params.Username != "alice" {
		t.Fatalf("expected trimmed username, got %q", store.params.Username)
	}
	if store.params.PasswordHash == "" || store.params.PasswordHash == "secret" {
		t.Fatalf("expected hashed password, got %q", store.params.PasswordHash)
	}
	if store.userID != store.account.UserID {
		t.Fatal("expected CreateUserProfile to receive the created account user ID")
	}

	body := rec.Body.String()
	for _, want := range []string{
		`"username":"alice"`,
		`"disabled":false`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}

func TestAccountsEndpointHandlesDuplicateUsername(t *testing.T) {
	t.Parallel()

	srv := &Server{
		queries: &fakeAccountCreator{
			createAccountErr: &pgconn.PgError{Code: "23505"},
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestAccountsEndpointHandlesStoreFailure(t *testing.T) {
	t.Parallel()

	srv := &Server{
		queries: &fakeAccountCreator{
			createAccountErr: errors.New("boom"),
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestAccountsEndpointHandlesUserProfileFailure(t *testing.T) {
	t.Parallel()

	srv := &Server{
		queries: &fakeAccountCreator{
			account: appdb.Account{
				UserID:   pgtype.UUID{Bytes: [16]byte{1, 2, 3}, Valid: true},
				Username: "alice",
			},
			createUserErr: errors.New("boom"),
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestLoginEndpointRequiresDatabase(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestLoginEndpointRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	srv := &Server{queries: &fakeAccountCreator{}}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret","extra":true}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestLoginEndpointValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	srv := &Server{queries: &fakeAccountCreator{}}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":" ","password":" "}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestLoginEndpointAuthenticatesValidCredentials(t *testing.T) {
	t.Parallel()

	passwordHash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	store := &fakeAccountCreator{
		account: appdb.Account{
			UserID:       pgtype.UUID{Bytes: [16]byte{1, 2, 3}, Valid: true},
			Username:     "alice",
			PasswordHash: passwordHash,
		},
	}
	srv := &Server{queries: store}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":" alice ","password":" secret "}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !store.getCalled {
		t.Fatal("expected GetAccountByUsername to be called")
	}
	if store.username != "alice" {
		t.Fatalf("expected trimmed username, got %q", store.username)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"username":"alice"`) {
		t.Fatalf("expected response body to contain username, got %s", body)
	}
}

func TestLoginEndpointRejectsUnknownUsername(t *testing.T) {
	t.Parallel()

	srv := &Server{
		queries: &fakeAccountCreator{
			getAccountErr: pgx.ErrNoRows,
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestLoginEndpointRejectsInvalidPassword(t *testing.T) {
	t.Parallel()

	passwordHash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	srv := &Server{
		queries: &fakeAccountCreator{
			account: appdb.Account{
				Username:     "alice",
				PasswordHash: passwordHash,
			},
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"wrong"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestLoginEndpointRejectsDisabledAccounts(t *testing.T) {
	t.Parallel()

	passwordHash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	srv := &Server{
		queries: &fakeAccountCreator{
			account: appdb.Account{
				Username:     "alice",
				Disabled:     true,
				PasswordHash: passwordHash,
			},
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}
