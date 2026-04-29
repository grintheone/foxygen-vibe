package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
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
	account              appdb.Account
	accountByUserID      appdb.Account
	profile              appdb.GetUserProfileByUserIDRow
	refreshTokens        []appdb.RefreshToken
	createAccountErr     error
	createUserErr        error
	getAccountErr        error
	getAccountByIDErr    error
	getProfileErr        error
	createRefreshErr     error
	getRefreshErrs       []error
	rotateRefreshErr     error
	updatePasswordErr    error
	createAccountParams  appdb.CreateAccountParams
	createRefreshParams  appdb.CreateRefreshTokenParams
	rotateRefreshParams  appdb.RotateRefreshTokenParams
	updatePasswordArgs   appdb.UpdateAccountPasswordHashParams
	accountCalled        bool
	userCalled           bool
	getCalled            bool
	getByIDCalled        bool
	getProfileCalled     bool
	createRefreshCalled  bool
	rotateRefreshCalled  bool
	updatePasswordCalled bool
	username             string
	userID               pgtype.UUID
	rotateRows           int64
	updatePasswordRows   int64
	refreshIndex         int
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

func (f *fakeAccountStore) UpdateAccountPasswordHash(_ context.Context, params appdb.UpdateAccountPasswordHashParams) (int64, error) {
	f.updatePasswordCalled = true
	f.updatePasswordArgs = params
	return f.updatePasswordRows, f.updatePasswordErr
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
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, PATCH, OPTIONS" {
		t.Fatalf("unexpected allow methods header %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type, X-Sync-Secret" {
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

func TestSyncEndpointRejectsNonPost(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestSyncEndpointRequiresConfiguration(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestSyncEndpointRequiresSecret(t *testing.T) {
	t.Parallel()

	srv := &Server{sync: syncConfig{sharedSecret: "shared-ticket-secret"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestSyncSecretMatches(t *testing.T) {
	t.Parallel()

	if !syncSecretMatches("shared-ticket-secret", "shared-ticket-secret") {
		t.Fatal("expected matching secrets to compare true")
	}
	if syncSecretMatches("shared-ticket-secret", "wrong-secret") {
		t.Fatal("expected different secrets to compare false")
	}
	if syncSecretMatches("", "shared-ticket-secret") {
		t.Fatal("expected empty configured secret to compare false")
	}
}

func TestNormalizeTicketSyncMetadataDefaultsSourceWhenKeyIsPresent(t *testing.T) {
	t.Parallel()

	source, key := normalizeTicketSyncMetadata(" ", " abc-123 ")
	if source != defaultTicketSyncSource {
		t.Fatalf("expected default source %q, got %q", defaultTicketSyncSource, source)
	}
	if key != "abc-123" {
		t.Fatalf("expected trimmed key, got %q", key)
	}
}

func TestNormalizeTicketSyncMetadataPreservesExplicitSource(t *testing.T) {
	t.Parallel()

	source, key := normalizeTicketSyncMetadata("lab-dispatcher", "abc-123")
	if source != "lab-dispatcher" {
		t.Fatalf("expected explicit source to be preserved, got %q", source)
	}
	if key != "abc-123" {
		t.Fatalf("expected key to be preserved, got %q", key)
	}
}

func TestNormalizeTicketSyncAuthorPrefersAuthorFields(t *testing.T) {
	t.Parallel()

	author, title := normalizeTicketSyncAuthor(" author-id ", " Dispatcher ", "legacy-id", "Legacy Dispatcher")
	if author != "author-id" {
		t.Fatalf("expected author field to win, got %q", author)
	}
	if title != "Dispatcher" {
		t.Fatalf("expected author_title to be trimmed, got %q", title)
	}
}

func TestNormalizeTicketSyncAuthorFallsBackToLegacyFields(t *testing.T) {
	t.Parallel()

	author, title := normalizeTicketSyncAuthor("", "", " legacy-id ", " Legacy Dispatcher ")
	if author != "legacy-id" {
		t.Fatalf("expected legacy author id to be used, got %q", author)
	}
	if title != "Legacy Dispatcher" {
		t.Fatalf("expected legacy author title to be used, got %q", title)
	}
}

func TestNormalizeSyncedTicketReasonPreservesKnownReason(t *testing.T) {
	t.Parallel()

	got := normalizeSyncedTicketReason("maintenance", true)
	if got != "maintenance" {
		t.Fatalf("expected known reason to be preserved, got %q", got)
	}
}

func TestNormalizeSyncedTicketReasonDropsUnknownReason(t *testing.T) {
	t.Parallel()

	got := normalizeSyncedTicketReason("maintenance", false)
	if got != "" {
		t.Fatalf("expected unknown reason to be dropped, got %q", got)
	}
}

func TestParseOptionalUUIDAllowsEmptyValue(t *testing.T) {
	t.Parallel()

	value, err := parseOptionalUUID("")
	if err != nil {
		t.Fatalf("parse empty optional uuid: %v", err)
	}
	if value.Valid {
		t.Fatal("expected empty optional uuid to stay invalid")
	}
}

func TestParseOptionalUUIDRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	if _, err := parseOptionalUUID("not-a-uuid"); err == nil {
		t.Fatal("expected invalid uuid error")
	}
}

func TestDecodeTicketSyncRequestsSupportsTicketEnvelope(t *testing.T) {
	t.Parallel()

	requests, wrapped, err := decodeTicketSyncRequests([]byte(`{
		"type":"tickets",
		"data":[
			{
				"id":"ticket-123",
				"ticketType":"internal",
				"author":{"id":"user-1","title":"Dispatcher","login":"ignored"},
				"client":{"id":"client-1","title":"ignored"},
				"contactPerson":{"id":"contact-1","firstName":"ignored"},
				"department":{"id":"department-1","title":"Service Department"},
				"description":"  Needs diagnostics  ",
				"device":{"id":"device-1","serialNumber":"SN-42"},
				"reason":"diagnostic",
				"urgent":true
			}
		]
	}`))
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if !wrapped {
		t.Fatal("expected wrapped tickets payload")
	}
	if len(requests) != 1 {
		t.Fatalf("expected one request, got %d", len(requests))
	}

	got := requests[0]
	if got.SyncKey != "ticket-123" {
		t.Fatalf("expected sync key from ticket id, got %q", got.SyncKey)
	}
	if got.Source != "tickets" {
		t.Fatalf("expected source tickets, got %q", got.Source)
	}
	if got.Client != "client-1" || got.Device != "device-1" || got.ContactPerson != "contact-1" {
		t.Fatalf("expected ids copied from nested refs, got %+v", got)
	}
	if got.Department != "department-1" {
		t.Fatalf("expected department id preferred, got %q", got.Department)
	}
	if got.Author != "user-1" || got.AuthorTitle != "Dispatcher" {
		t.Fatalf("expected author mapped, got %+v", got)
	}
	if got.Description != "Needs diagnostics" {
		t.Fatalf("expected trimmed description, got %q", got.Description)
	}
	if got.Reason != "diagnostic" || got.TicketType != "internal" {
		t.Fatalf("expected ticket metadata mapped, got %+v", got)
	}
	if !got.Urgent {
		t.Fatal("expected urgent flag preserved")
	}
}

func TestDecodeTicketSyncRequestsRejectsUnsupportedEnvelopeType(t *testing.T) {
	t.Parallel()

	_, wrapped, err := decodeTicketSyncRequests([]byte(`{"type":"device","data":{"id":"device-1"}}`))
	if err == nil {
		t.Fatal("expected unsupported type error")
	}
	if !wrapped {
		t.Fatal("expected typed envelope to be detected as wrapped")
	}
	if !strings.Contains(err.Error(), `unsupported sync type "device"`) {
		t.Fatalf("expected unsupported type error, got %q", err.Error())
	}
}

func TestDecodeDeviceSyncEnvelopeData(t *testing.T) {
	t.Parallel()

	input, err := decodeDeviceSyncEnvelopeData(json.RawMessage(`{
		"id":"device-123",
		"ref":"legacy-ref-1",
		"serialNumber":"SN-42",
		"bindings":[{"client":"client-1"}],
		"properties":{"rack":2},
		"connectedToLis":true
	}`))
	if err != nil {
		t.Fatalf("decode device envelope: %v", err)
	}
	if input.ID != "device-123" || input.Ref != "legacy-ref-1" || input.SerialNumber != "SN-42" {
		t.Fatalf("expected identifiers to be decoded, got %+v", input)
	}
	if len(input.Bindings) != 1 || input.Bindings[0].Client != "client-1" {
		t.Fatalf("expected bindings to be decoded, got %+v", input.Bindings)
	}
	if !input.ConnectedToLis {
		t.Fatal("expected connectedToLis to be preserved")
	}
	if string(input.Properties) != `{"rack":2}` {
		t.Fatalf("expected properties raw json preserved, got %s", string(input.Properties))
	}
}

func TestDecodeContactSyncEnvelopeData(t *testing.T) {
	t.Parallel()

	input, err := decodeContactSyncEnvelopeData(json.RawMessage(`{
		"id":"ab55b2b2-3d63-11f1-814b-40b0765b1e01",
		"ref":"de7a4716-169c-11e6-a438-001a64d22812",
		"firstName":"Арина",
		"middleName":" ",
		"lastName":"Иванова",
		"position":"Врач",
		"phone":"89883969704",
		"email":"ARINA@example.com",
		"disableNotification":true,
		"sendAllNotifications":false
	}`))
	if err != nil {
		t.Fatalf("decode contact envelope: %v", err)
	}
	if input.ID != "ab55b2b2-3d63-11f1-814b-40b0765b1e01" || input.Ref != "de7a4716-169c-11e6-a438-001a64d22812" {
		t.Fatalf("expected identifiers to be decoded, got %+v", input)
	}
	if input.FirstName != "Арина" || input.LastName != "Иванова" {
		t.Fatalf("expected name fields to be decoded, got %+v", input)
	}
	if input.Position != "Врач" || input.Phone != "89883969704" || input.Email != "ARINA@example.com" {
		t.Fatalf("expected contact details to be decoded, got %+v", input)
	}
	if !input.DisableNotification || input.SendAllNotifications {
		t.Fatalf("expected sync flags to be preserved, got %+v", input)
	}
}

func TestProcessContactSyncRequestCreatesContact(t *testing.T) {
	t.Parallel()

	contactID := mustUUID(t, "ab55b2b2-3d63-11f1-814b-40b0765b1e01")
	clientID := mustUUID(t, "de7a4716-169c-11e6-a438-001a64d22812")

	var (
		gotContactID pgtype.UUID
		gotName      string
		gotPosition  string
		gotPhone     string
		gotEmail     string
		gotClientID  pgtype.UUID
	)

	srv := &Server{
		syncClientExists: func(_ context.Context, id pgtype.UUID) (bool, error) {
			if id != clientID {
				t.Fatalf("expected client lookup id %s, got %s", clientID.String(), id.String())
			}
			return true, nil
		},
		syncContactUpserter: func(_ context.Context, id pgtype.UUID, name string, position string, phone string, email string, linkedClientID pgtype.UUID) (bool, error) {
			gotContactID = id
			gotName = name
			gotPosition = position
			gotPhone = phone
			gotEmail = email
			gotClientID = linkedClientID
			return true, nil
		},
	}

	statusCode, response, err := srv.processContactSyncRequest(context.Background(), "172.30.240.4:49232", syncContactRequest{
		ID:        contactID.String(),
		Ref:       clientID.String(),
		FirstName: " Арина ",
		LastName:  " Иванова ",
		Position:  " Врач ",
		Phone:     " 89883969704 ",
		Email:     " ARINA@example.com ",
	})
	if err != nil {
		t.Fatalf("process contact sync: %v", err)
	}
	if statusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, statusCode)
	}
	if gotContactID != contactID || gotClientID != clientID {
		t.Fatal("expected upserter to receive contact and client ids")
	}
	if gotName != "Арина Иванова" {
		t.Fatalf("expected normalized contact name, got %q", gotName)
	}
	if gotPosition != "Врач" || gotPhone != "89883969704" || gotEmail != "arina@example.com" {
		t.Fatalf("expected trimmed normalized fields, got %q / %q / %q", gotPosition, gotPhone, gotEmail)
	}
	if response.ID != contactID.String() || response.Client != clientID.String() {
		t.Fatalf("expected response identifiers, got %+v", response)
	}
	if !response.Created || response.Name != "Арина Иванова" {
		t.Fatalf("expected created response with normalized name, got %+v", response)
	}
}

func TestDeterministicSyncAgreementIDIsStable(t *testing.T) {
	t.Parallel()

	first := deterministicSyncAgreementID("device-1", "client-1")
	second := deterministicSyncAgreementID("device-1", "client-1")
	other := deterministicSyncAgreementID("device-1", "client-2")
	if first != second {
		t.Fatalf("expected deterministic id, got %q and %q", first, second)
	}
	if first == other {
		t.Fatalf("expected different client to change id, got %q", first)
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

func TestChangePasswordEndpointRejectsMissingToken(t *testing.T) {
	t.Parallel()

	srv := &Server{
		queries: &fakeAccountStore{},
		auth:    testAuthConfig(),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", strings.NewReader(`{"currentPassword":"secret","newPassword":"new-secret"}`))
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestChangePasswordEndpointRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	secret := testAuthConfig().jwtSecret
	token, err := signJWT(secret, accessTokenClaims{
		Subject:   "11111111-1111-1111-1111-111111111111",
		Username:  "alice",
		TokenType: accessTokenType,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	srv := &Server{
		queries: &fakeAccountStore{},
		auth:    testAuthConfig(),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", strings.NewReader(`{"currentPassword":"secret","newPassword":"new-secret","extra":true}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestChangePasswordEndpointValidatesInput(t *testing.T) {
	t.Parallel()

	secret := testAuthConfig().jwtSecret
	token, err := signJWT(secret, accessTokenClaims{
		Subject:   "11111111-1111-1111-1111-111111111111",
		Username:  "alice",
		TokenType: accessTokenType,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	srv := &Server{
		queries: &fakeAccountStore{},
		auth:    testAuthConfig(),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", strings.NewReader(`{"currentPassword":" ","newPassword":"short"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestChangePasswordEndpointRejectsWrongCurrentPassword(t *testing.T) {
	t.Parallel()

	passwordHash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	secret := testAuthConfig().jwtSecret
	token, err := signJWT(secret, accessTokenClaims{
		Subject:   "11111111-1111-1111-1111-111111111111",
		Username:  "alice",
		TokenType: accessTokenType,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	srv := &Server{
		queries: &fakeAccountStore{
			accountByUserID: appdb.Account{
				UserID:       pgtype.UUID{Bytes: [16]byte{1, 1, 1}, Valid: true},
				Username:     "alice",
				PasswordHash: passwordHash,
			},
		},
		auth: testAuthConfig(),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", strings.NewReader(`{"currentPassword":"wrong","newPassword":"new-secret"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestChangePasswordEndpointUpdatesPasswordHash(t *testing.T) {
	t.Parallel()

	passwordHash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	secret := testAuthConfig().jwtSecret
	token, err := signJWT(secret, accessTokenClaims{
		Subject:   "11111111-1111-1111-1111-111111111111",
		Username:  "alice",
		TokenType: accessTokenType,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	store := &fakeAccountStore{
		accountByUserID: appdb.Account{
			UserID:       pgtype.UUID{Bytes: [16]byte{1, 1, 1}, Valid: true},
			Username:     "alice",
			PasswordHash: passwordHash,
		},
		updatePasswordRows: 1,
	}
	srv := &Server{
		queries: store,
		auth:    testAuthConfig(),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/change-password", strings.NewReader(`{"currentPassword":" secret ","newPassword":" fresh-secret "}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if !store.getByIDCalled {
		t.Fatal("expected GetAccountByUserID to be called")
	}
	if !store.updatePasswordCalled {
		t.Fatal("expected UpdateAccountPasswordHash to be called")
	}
	if store.updatePasswordArgs.PasswordHash == "" || store.updatePasswordArgs.PasswordHash == "fresh-secret" {
		t.Fatalf("expected hashed password, got %q", store.updatePasswordArgs.PasswordHash)
	}
	if !strings.HasPrefix(store.updatePasswordArgs.PasswordHash, "$2") {
		t.Fatalf("expected bcrypt password hash, got %q", store.updatePasswordArgs.PasswordHash)
	}
	if !verifyPassword("fresh-secret", store.updatePasswordArgs.PasswordHash) {
		t.Fatal("expected updated password hash to verify")
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

func TestEditorClientsEndpointRejectsMissingToken(t *testing.T) {
	t.Parallel()

	srv := &Server{auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestEditorClientsEndpointRequiresDatabase(t *testing.T) {
	t.Parallel()

	secret := testAuthConfig().jwtSecret
	token, err := signJWT(secret, accessTokenClaims{
		Subject:   "11111111-1111-1111-1111-111111111111",
		Username:  "coordinator",
		TokenType: accessTokenType,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	srv := &Server{auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
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
		`"disabled":false`,
		`"name":"Maya Hernandez"`,
		`"email":"maya.hernandez@foxygen.dev"`,
		`"department":"Mobile Engineering"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}

func TestProfileDisabledEndpointRejectsSelfDisable(t *testing.T) {
	t.Parallel()

	requesterID := mustUUID(t, "11111111-1111-1111-1111-111111111111")
	srv := &Server{
		auth: testAuthConfig(),
		editorAccessCheck: func(_ http.ResponseWriter, _ *http.Request) (pgtype.UUID, bool) {
			return requesterID, true
		},
		profileAccessCheck: func(_ context.Context, _, _ pgtype.UUID) (bool, error) {
			return true, nil
		},
		accountDisabledUpdater: func(_ context.Context, _ pgtype.UUID, _ bool) (bool, error) {
			t.Fatal("account disabled updater should not be called for self-disable")
			return false, nil
		},
	}

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/profile/11111111-1111-1111-1111-111111111111/disabled",
		strings.NewReader(`{"disabled":true}`),
	)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "cannot disable yourself") {
		t.Fatalf("expected self-disable error, got %q", body)
	}
}

func TestProfileDisabledEndpointUpdatesAccountFlag(t *testing.T) {
	t.Parallel()

	requesterID := mustUUID(t, "11111111-1111-1111-1111-111111111111")
	targetID := mustUUID(t, "22222222-2222-2222-2222-222222222222")

	var (
		gotTargetID pgtype.UUID
		gotDisabled bool
	)

	srv := &Server{
		auth: testAuthConfig(),
		editorAccessCheck: func(_ http.ResponseWriter, _ *http.Request) (pgtype.UUID, bool) {
			return requesterID, true
		},
		profileAccessCheck: func(_ context.Context, gotRequesterID, gotProfileID pgtype.UUID) (bool, error) {
			if gotRequesterID != requesterID {
				t.Fatalf("expected requester id %s, got %s", requesterID.String(), gotRequesterID.String())
			}
			if gotProfileID != targetID {
				t.Fatalf("expected target id %s, got %s", targetID.String(), gotProfileID.String())
			}
			return true, nil
		},
		accountDisabledUpdater: func(_ context.Context, userID pgtype.UUID, disabled bool) (bool, error) {
			gotTargetID = userID
			gotDisabled = disabled
			return true, nil
		},
	}

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/profile/22222222-2222-2222-2222-222222222222/disabled",
		strings.NewReader(`{"disabled":true}`),
	)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if gotTargetID != targetID {
		t.Fatalf("expected disabled update for %s, got %s", targetID.String(), gotTargetID.String())
	}
	if !gotDisabled {
		t.Fatal("expected disabled flag to be set to true")
	}
	body := rec.Body.String()
	for _, want := range []string{
		`"user_id":"22222222-2222-2222-2222-222222222222"`,
		`"disabled":true`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}

func TestProfileAvatarDownloadEndpointUsesPublicRoute(t *testing.T) {
	t.Parallel()

	srv := &Server{auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodGet, "/api/profile/11111111-1111-1111-1111-111111111111/avatar", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestProfileAvatarUploadEndpointRejectsMissingToken(t *testing.T) {
	t.Parallel()

	srv := &Server{auth: testAuthConfig()}
	req := httptest.NewRequest(http.MethodPost, "/api/profile/avatar", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
