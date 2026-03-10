package api

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appdb "foxygen-vibe/server/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type fakeAccountStore struct {
	account             appdb.Account
	accountByUserID     appdb.Account
	profile             appdb.GetUserProfileByUserIDRow
	refreshTokens       []appdb.RefreshToken
	createAccountErr    error
	createUserErr       error
	getAccountErr       error
	getAccountByIDErr   error
	getProfileErr       error
	createRefreshErr    error
	getRefreshErrs      []error
	rotateRefreshErr    error
	createAccountParams appdb.CreateAccountParams
	createRefreshParams appdb.CreateRefreshTokenParams
	rotateRefreshParams appdb.RotateRefreshTokenParams
	accountCalled       bool
	userCalled          bool
	getCalled           bool
	getByIDCalled       bool
	getProfileCalled    bool
	createRefreshCalled bool
	rotateRefreshCalled bool
	username            string
	userID              pgtype.UUID
	rotateRows          int64
	refreshIndex        int
}

func (f *fakeAccountStore) CreateAccount(_ context.Context, params appdb.CreateAccountParams) (appdb.Account, error) {
	f.accountCalled = true
	f.createAccountParams = params
	return f.account, f.createAccountErr
}

func (f *fakeAccountStore) CreateUserProfile(_ context.Context, userID pgtype.UUID) (appdb.User, error) {
	f.userCalled = true
	f.userID = userID
	return appdb.User{UserID: userID}, f.createUserErr
}

func (f *fakeAccountStore) GetAccountByUsername(_ context.Context, username string) (appdb.Account, error) {
	f.getCalled = true
	f.username = username
	return f.account, f.getAccountErr
}

func (f *fakeAccountStore) GetAccountByUserID(_ context.Context, userID pgtype.UUID) (appdb.Account, error) {
	f.getByIDCalled = true
	f.userID = userID
	return f.accountByUserID, f.getAccountByIDErr
}

func (f *fakeAccountStore) GetUserProfileByUserID(_ context.Context, userID pgtype.UUID) (appdb.GetUserProfileByUserIDRow, error) {
	f.getProfileCalled = true
	f.userID = userID
	return f.profile, f.getProfileErr
}

func (f *fakeAccountStore) CreateRefreshToken(_ context.Context, params appdb.CreateRefreshTokenParams) (appdb.RefreshToken, error) {
	f.createRefreshCalled = true
	f.createRefreshParams = params
	if f.createRefreshErr != nil {
		return appdb.RefreshToken{}, f.createRefreshErr
	}

	return appdb.RefreshToken{
		TokenID: pgtype.UUID{Bytes: [16]byte{9, 9, 9}, Valid: true},
		UserID:  params.UserID,
	}, nil
}

func (f *fakeAccountStore) GetRefreshTokenByHash(_ context.Context, _ string) (appdb.RefreshToken, error) {
	index := f.refreshIndex
	f.refreshIndex++

	if index < len(f.getRefreshErrs) && f.getRefreshErrs[index] != nil {
		return appdb.RefreshToken{}, f.getRefreshErrs[index]
	}
	if index < len(f.refreshTokens) {
		return f.refreshTokens[index], nil
	}

	return appdb.RefreshToken{}, pgx.ErrNoRows
}

func (f *fakeAccountStore) RotateRefreshToken(_ context.Context, params appdb.RotateRefreshTokenParams) (int64, error) {
	f.rotateRefreshCalled = true
	f.rotateRefreshParams = params
	return f.rotateRows, f.rotateRefreshErr
}

func testAuthConfig() authConfig {
	return authConfig{
		jwtSecret:       []byte("test-secret"),
		accessTokenTTL:  15 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour,
	}
}

func validRefreshTokenRecord(tokenID byte, userID pgtype.UUID) appdb.RefreshToken {
	return appdb.RefreshToken{
		TokenID:   pgtype.UUID{Bytes: [16]byte{tokenID}, Valid: true},
		UserID:    userID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
	}
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
		`"storage":{"configured":false,"connected":false}`,
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
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type" {
		t.Fatalf("unexpected allow headers %q", got)
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

	srv := &Server{queries: &fakeAccountStore{}}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":"alice","password":"secret","extra":true}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAccountsEndpointValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	srv := &Server{queries: &fakeAccountStore{}}
	req := httptest.NewRequest(http.MethodPost, "/api/accounts", strings.NewReader(`{"username":" ","password":" "}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAccountsEndpointCreatesAccount(t *testing.T) {
	t.Parallel()

	store := &fakeAccountStore{
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
	if store.createAccountParams.Username != "alice" {
		t.Fatalf("expected trimmed username, got %q", store.createAccountParams.Username)
	}
	if store.createAccountParams.PasswordHash == "" || store.createAccountParams.PasswordHash == "secret" {
		t.Fatalf("expected hashed password, got %q", store.createAccountParams.PasswordHash)
	}
	if !strings.HasPrefix(store.createAccountParams.PasswordHash, "$2") {
		t.Fatalf("expected bcrypt password hash, got %q", store.createAccountParams.PasswordHash)
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
		queries: &fakeAccountStore{
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
		queries: &fakeAccountStore{
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
		queries: &fakeAccountStore{
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

	srv := &Server{auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestLoginEndpointRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	srv := &Server{queries: &fakeAccountStore{}, auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret","extra":true}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestLoginEndpointValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	srv := &Server{queries: &fakeAccountStore{}, auth: testAuthConfig()}
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

	store := &fakeAccountStore{
		account: appdb.Account{
			UserID:       pgtype.UUID{Bytes: [16]byte{1, 2, 3}, Valid: true},
			Username:     "alice",
			PasswordHash: passwordHash,
		},
	}
	srv := &Server{queries: store, auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":" alice ","password":" secret "}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !store.getCalled {
		t.Fatal("expected GetAccountByUsername to be called")
	}
	if !store.createRefreshCalled {
		t.Fatal("expected CreateRefreshToken to be called")
	}
	if store.username != "alice" {
		t.Fatalf("expected trimmed username, got %q", store.username)
	}

	body := rec.Body.String()
	for _, want := range []string{
		`"username":"alice"`,
		`"access_token":"`,
		`"refresh_token":"`,
		`"token_type":"Bearer"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}

func TestLoginEndpointRejectsUnknownUsername(t *testing.T) {
	t.Parallel()

	srv := &Server{
		queries: &fakeAccountStore{
			getAccountErr: pgx.ErrNoRows,
		},
		auth: testAuthConfig(),
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
		queries: &fakeAccountStore{
			account: appdb.Account{
				Username:     "alice",
				PasswordHash: passwordHash,
			},
		},
		auth: testAuthConfig(),
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
		queries: &fakeAccountStore{
			account: appdb.Account{
				Username:     "alice",
				Disabled:     true,
				PasswordHash: passwordHash,
			},
		},
		auth: testAuthConfig(),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestVerifyPasswordSupportsLegacySHA256Hashes(t *testing.T) {
	t.Parallel()

	salt := []byte("0123456789abcdef")
	stored := "sha256$" + hex.EncodeToString(salt) + "$" + hashPasswordWithSalt("secret", salt)

	if !verifyPassword("secret", stored) {
		t.Fatal("expected legacy SHA-256 password hash to verify")
	}
	if verifyPassword("wrong", stored) {
		t.Fatal("expected legacy SHA-256 password hash to reject the wrong password")
	}
}

func TestRefreshEndpointRequiresDatabase(t *testing.T) {
	t.Parallel()

	srv := &Server{auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refresh_token":"abc"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestRefreshEndpointRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	srv := &Server{queries: &fakeAccountStore{}, auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refresh_token":"abc","extra":true}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRefreshEndpointRotatesRefreshToken(t *testing.T) {
	t.Parallel()

	userID := pgtype.UUID{Bytes: [16]byte{7, 7, 7}, Valid: true}
	store := &fakeAccountStore{
		accountByUserID: appdb.Account{
			UserID:       userID,
			Username:     "alice",
			PasswordHash: "ignored",
		},
		refreshTokens: []appdb.RefreshToken{
			validRefreshTokenRecord(1, userID),
			validRefreshTokenRecord(2, userID),
		},
		rotateRows: 1,
	}
	srv := &Server{queries: store, auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refresh_token":"abc"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !store.getByIDCalled {
		t.Fatal("expected GetAccountByUserID to be called")
	}
	if !store.createRefreshCalled {
		t.Fatal("expected CreateRefreshToken to be called")
	}
	if !store.rotateRefreshCalled {
		t.Fatal("expected RotateRefreshToken to be called")
	}

	body := rec.Body.String()
	for _, want := range []string{
		`"access_token":"`,
		`"refresh_token":"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}

func TestRefreshEndpointRejectsUnknownToken(t *testing.T) {
	t.Parallel()

	srv := &Server{
		queries: &fakeAccountStore{
			getRefreshErrs: []error{pgx.ErrNoRows},
		},
		auth: testAuthConfig(),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refresh_token":"abc"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestRefreshEndpointRejectsRotatedToken(t *testing.T) {
	t.Parallel()

	userID := pgtype.UUID{Bytes: [16]byte{8, 8, 8}, Valid: true}
	record := validRefreshTokenRecord(1, userID)
	record.RotatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}

	srv := &Server{
		queries: &fakeAccountStore{
			refreshTokens: []appdb.RefreshToken{record},
		},
		auth: testAuthConfig(),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refresh_token":"abc"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestSessionEndpointRejectsMissingToken(t *testing.T) {
	t.Parallel()

	srv := &Server{auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestSessionEndpointReturnsClaims(t *testing.T) {
	t.Parallel()

	secret := testAuthConfig().jwtSecret
	token, err := signJWT(secret, accessTokenClaims{
		Subject:   "user-123",
		Username:  "alice",
		TokenType: accessTokenType,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	srv := &Server{auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"username":"alice"`) {
		t.Fatalf("expected response body to contain username, got %s", body)
	}
}

func TestProfileEndpointRequiresDatabase(t *testing.T) {
	t.Parallel()

	secret := testAuthConfig().jwtSecret
	token, err := signJWT(secret, accessTokenClaims{
		Subject:   "11111111-1111-1111-1111-111111111111",
		Username:  "mobile.lead",
		TokenType: accessTokenType,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	srv := &Server{auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestProfileEndpointRejectsMissingToken(t *testing.T) {
	t.Parallel()

	srv := &Server{
		queries: &fakeAccountStore{},
		auth:    testAuthConfig(),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestProfileEndpointReturnsProfile(t *testing.T) {
	t.Parallel()

	secret := testAuthConfig().jwtSecret
	userID := pgtype.UUID{Bytes: [16]byte{4, 5, 6}, Valid: true}
	token, err := signJWT(secret, accessTokenClaims{
		Subject:   userID.String(),
		Username:  "mobile.lead",
		TokenType: accessTokenType,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	store := &fakeAccountStore{
		profile: appdb.GetUserProfileByUserIDRow{
			UserID:     userID,
			Username:   "mobile.lead",
			Name:       "Maya Hernandez",
			Email:      "maya.hernandez@foxygen.dev",
			Department: "Mobile Engineering",
		},
	}
	srv := &Server{queries: store, auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !store.getProfileCalled {
		t.Fatal("expected GetUserProfileByUserID to be called")
	}
	if store.userID != userID {
		t.Fatal("expected profile lookup to use token subject")
	}

	body := rec.Body.String()
	for _, want := range []string{
		`"username":"mobile.lead"`,
		`"name":"Maya Hernandez"`,
		`"email":"maya.hernandez@foxygen.dev"`,
		`"department":"Mobile Engineering"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}
