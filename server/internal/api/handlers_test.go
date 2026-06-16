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

func TestResolveKnownTicketReasonIDPreservesKnownReason(t *testing.T) {
	t.Parallel()

	got := resolveKnownTicketReasonID("maintenance", func(candidate string) bool {
		return candidate == "maintenance"
	})
	if got != "maintenance" {
		t.Fatalf("expected known reason to be preserved, got %q", got)
	}
}

func TestResolveKnownTicketReasonIDFallsBackToLegacyMaintenanceID(t *testing.T) {
	t.Parallel()

	got := resolveKnownTicketReasonID("maintenance", func(candidate string) bool {
		return candidate == "maintanence"
	})
	if got != "maintanence" {
		t.Fatalf("expected maintenance to resolve to maintanence, got %q", got)
	}
}

func TestResolveKnownTicketReasonIDFallsBackToCanonicalMaintenanceID(t *testing.T) {
	t.Parallel()

	got := resolveKnownTicketReasonID("maintanence", func(candidate string) bool {
		return candidate == "maintenance"
	})
	if got != "maintenance" {
		t.Fatalf("expected maintanence to resolve to maintenance, got %q", got)
	}
}

func TestResolveKnownTicketReasonIDDropsUnknownReason(t *testing.T) {
	t.Parallel()

	got := resolveKnownTicketReasonID("maintenance", func(string) bool {
		return false
	})
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

func TestDecodeTicketSyncRequestsMapsAssignedUsersFromEnvelope(t *testing.T) {
	t.Parallel()

	requests, wrapped, err := decodeTicketSyncRequests([]byte(`{
		"type":"tickets",
		"data":{
			"id":"ticket-456",
			"client":{"id":"client-1"},
			"department":{"id":"department-1","title":"Service Department"},
			"description":"Needs assignment",
			"device":{"id":"device-1"},
			"executor":{
				"id":"2e3358a6-6613-11f0-8136-40b0765b1e01",
				"login":"Albert.Andrianov",
				"password":"wtqWVoYP",
				"title":"Альберт Андрианов",
				"department":{"id":"2e3358a7-6613-11f0-8136-40b0765b1e01","title":"Сервис КПД"},
				"email":"AA@KPD.com.ru"
			},
			"assignedBy":{
				"id":"6f1a8d46-6613-11f0-8136-40b0765b1e01",
				"login":"Anna.Ivanova",
				"password":"Secret123",
				"title":"Анна Иванова",
				"department":{"id":"2e3358a7-6613-11f0-8136-40b0765b1e01","title":"Сервис КПД"},
				"email":"ANNA@example.com"
			},
			"reason":"diagnostic"
		}
	}`))
	if err != nil {
		t.Fatalf("decode envelope with assignees: %v", err)
	}
	if !wrapped {
		t.Fatal("expected wrapped tickets payload")
	}
	if len(requests) != 1 {
		t.Fatalf("expected one request, got %d", len(requests))
	}

	got := requests[0]
	if got.Status != "assigned" {
		t.Fatalf("expected assigned status derived from assignee payload, got %q", got.Status)
	}
	if got.Executor == nil {
		t.Fatal("expected executor to be mapped")
	}
	if got.Executor.ID != "2e3358a6-6613-11f0-8136-40b0765b1e01" ||
		got.Executor.Login != "Albert.Andrianov" ||
		got.Executor.Password != "wtqWVoYP" ||
		got.Executor.Title != "Альберт Андрианов" ||
		got.Executor.Email != "aa@kpd.com.ru" {
		t.Fatalf("unexpected executor mapping: %+v", got.Executor)
	}
	if got.Executor.Department == nil ||
		got.Executor.Department.ID != "2e3358a7-6613-11f0-8136-40b0765b1e01" ||
		got.Executor.Department.Title != "Сервис КПД" {
		t.Fatalf("unexpected executor department mapping: %+v", got.Executor.Department)
	}
	if got.AssignedBy == nil {
		t.Fatal("expected assignedBy to be mapped")
	}
	if got.AssignedBy.ID != "6f1a8d46-6613-11f0-8136-40b0765b1e01" ||
		got.AssignedBy.Login != "Anna.Ivanova" ||
		got.AssignedBy.Password != "Secret123" ||
		got.AssignedBy.Title != "Анна Иванова" ||
		got.AssignedBy.Email != "anna@example.com" {
		t.Fatalf("unexpected assignedBy mapping: %+v", got.AssignedBy)
	}
	if got.AssignedBy.Department == nil ||
		got.AssignedBy.Department.ID != "2e3358a7-6613-11f0-8136-40b0765b1e01" ||
		got.AssignedBy.Department.Title != "Сервис КПД" {
		t.Fatalf("unexpected assignedBy department mapping: %+v", got.AssignedBy.Department)
	}
}

func TestNormalizeSyncTicketStatusDefaultsToAssignedWhenExecutorPresent(t *testing.T) {
	t.Parallel()

	status := normalizeSyncTicketStatus("", &syncTicketUser{ID: "executor-1"}, nil)
	if status != "assigned" {
		t.Fatalf("expected executor to imply assigned status, got %q", status)
	}

	createdStatus := normalizeSyncTicketStatus("", nil, nil)
	if createdStatus != "created" {
		t.Fatalf("expected empty assignee payload to default to created, got %q", createdStatus)
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

func TestDecodeClientSyncEnvelopeData(t *testing.T) {
	t.Parallel()

	input, err := decodeClientSyncEnvelopeData(json.RawMessage(`{
		"id":"256406ac-59a2-11f1-814e-40b0765b1e01",
		"ref":null,
		"title":"ММЦ ВТ, Белоостров",
		"address":"Лен.обл, Всеволожский р-н, с.п.Юкковское, тер. Клиника Белоостров, зд. 1, к. 1",
		"region":null,
		"location":{"lat":60.123,"lon":30.456},
		"laboratorySystem":null
	}`))
	if err != nil {
		t.Fatalf("decode client envelope: %v", err)
	}
	if input.ID != "256406ac-59a2-11f1-814e-40b0765b1e01" {
		t.Fatalf("expected client id to be decoded, got %+v", input)
	}
	if input.Title != "ММЦ ВТ, Белоостров" || input.Address == "" {
		t.Fatalf("expected client details to be decoded, got %+v", input)
	}
	if compactSyncLogPayload(input.Location) != `{"lat":60.123,"lon":30.456}` {
		t.Fatalf("unexpected location payload: %s", compactSyncLogPayload(input.Location))
	}
}

func TestDecodeClassificatorSyncEnvelopeData(t *testing.T) {
	t.Parallel()

	input, err := decodeClassificatorSyncEnvelopeData(json.RawMessage(`{
		"id":"2c161c29-4dc9-11f1-814d-40b0765b1e01",
		"refs":[],
		"title":"TG16B-21 Центрифуга TG16B с №21 ротором угловым для пробирок 12-местным",
		"manufacturer":null,
		"researchType":null,
		"registrationCertificate":{"number":"","date":"0001-01-01T00:00:00Z","issueDate":"0001-01-01T00:00:00Z"},
		"maintenanceRegulations":[],
		"maintenanceMaterialsList":[],
		"attachments":[],
		"notes":[""],
		"images":[]
	}`))
	if err != nil {
		t.Fatalf("decode classificator envelope: %v", err)
	}
	if input.ID != "2c161c29-4dc9-11f1-814d-40b0765b1e01" {
		t.Fatalf("expected classificator id to be decoded, got %+v", input)
	}
	if input.Title == "" {
		t.Fatalf("expected classificator title to be decoded, got %+v", input)
	}
	if compactSyncLogPayload(input.RegistrationCertificate) != `{"number":"","date":"0001-01-01T00:00:00Z","issueDate":"0001-01-01T00:00:00Z"}` {
		t.Fatalf("unexpected registration certificate payload: %s", compactSyncLogPayload(input.RegistrationCertificate))
	}
	if compactSyncLogPayload(input.MaintenanceRegulations) != `[]` {
		t.Fatalf("unexpected maintenance regulations payload: %s", compactSyncLogPayload(input.MaintenanceRegulations))
	}
	if len(input.Attachments) != 0 || len(input.Images) != 0 {
		t.Fatalf("expected empty media lists, got attachments=%v images=%v", input.Attachments, input.Images)
	}
}

func TestProcessClientSyncRequestCreatesClient(t *testing.T) {
	t.Parallel()

	clientID := mustUUID(t, "256406ac-59a2-11f1-814e-40b0765b1e01")
	regionID := mustUUID(t, "11111111-1111-1111-1111-111111111111")
	laboratorySystemID := mustUUID(t, "22222222-2222-2222-2222-222222222222")

	var (
		gotClientID         pgtype.UUID
		gotTitle            string
		gotAddress          string
		gotRegion           any
		gotLocation         any
		gotLaboratorySystem any
	)

	srv := &Server{
		editorRegionExists: func(_ context.Context, id pgtype.UUID) (bool, error) {
			if id != regionID {
				t.Fatalf("expected region lookup id %s, got %s", regionID.String(), id.String())
			}
			return true, nil
		},
		syncClientUpserter: func(_ context.Context, id pgtype.UUID, title string, address string, region any, location any, laboratorySystem any) (bool, error) {
			gotClientID = id
			gotTitle = title
			gotAddress = address
			gotRegion = region
			gotLocation = location
			gotLaboratorySystem = laboratorySystem
			return true, nil
		},
	}

	statusCode, response, err := srv.processClientSyncRequest(context.Background(), "172.30.240.4:38570", syncClientRequest{
		ID:               clientID.String(),
		Title:            "  ММЦ ВТ,   Белоостров  ",
		Address:          " Лен.обл ",
		Region:           json.RawMessage(`{"id":"` + regionID.String() + `","title":"Ленинградская область"}`),
		Location:         json.RawMessage(`{"lat":60.123,"lon":30.456}`),
		LaboratorySystem: json.RawMessage(`"` + laboratorySystemID.String() + `"`),
	})
	if err != nil {
		t.Fatalf("process client sync: %v", err)
	}
	if statusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, statusCode)
	}
	if gotClientID != clientID {
		t.Fatalf("expected upserter to receive client id %s, got %s", clientID.String(), gotClientID.String())
	}
	if gotTitle != "ММЦ ВТ, Белоостров" || gotAddress != "Лен.обл" {
		t.Fatalf("expected normalized client fields, got %q / %q", gotTitle, gotAddress)
	}
	if gotRegionID, ok := gotRegion.(pgtype.UUID); !ok || gotRegionID != regionID {
		t.Fatalf("expected region uuid, got %#v", gotRegion)
	}
	if gotLaboratorySystemID, ok := gotLaboratorySystem.(pgtype.UUID); !ok || gotLaboratorySystemID != laboratorySystemID {
		t.Fatalf("expected laboratory system uuid, got %#v", gotLaboratorySystem)
	}
	location, ok := gotLocation.(json.RawMessage)
	if !ok || compactSyncLogPayload(location) != `{"lat":60.123,"lon":30.456}` {
		t.Fatalf("unexpected location: %#v", gotLocation)
	}
	if !response.Created || response.ID != clientID.String() || response.Title != "ММЦ ВТ, Белоостров" {
		t.Fatalf("unexpected client response: %+v", response)
	}
	if response.Region == nil || *response.Region != regionID.String() {
		t.Fatalf("expected region id in response, got %+v", response)
	}
	if response.LaboratorySystem == nil || *response.LaboratorySystem != laboratorySystemID.String() {
		t.Fatalf("expected laboratory system id in response, got %+v", response)
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

func TestProcessClassificatorSyncRequestCreatesClassificator(t *testing.T) {
	t.Parallel()

	classificatorID := mustUUID(t, "2c161c29-4dc9-11f1-814d-40b0765b1e01")
	manufacturerID := mustUUID(t, "11111111-1111-1111-1111-111111111111")
	researchTypeID := mustUUID(t, "22222222-2222-2222-2222-222222222222")

	var (
		gotClassificatorID         pgtype.UUID
		gotTitle                   string
		gotManufacturer            any
		gotResearchType            any
		gotRegistrationCertificate json.RawMessage
		gotMaintenanceRegulations  json.RawMessage
		gotAttachments             []string
		gotImages                  []string
	)

	srv := &Server{
		editorManufacturerExists: func(_ context.Context, id pgtype.UUID) (bool, error) {
			if id != manufacturerID {
				t.Fatalf("expected manufacturer lookup id %s, got %s", manufacturerID.String(), id.String())
			}
			return true, nil
		},
		editorResearchTypeExists: func(_ context.Context, id pgtype.UUID) (bool, error) {
			if id != researchTypeID {
				t.Fatalf("expected research type lookup id %s, got %s", researchTypeID.String(), id.String())
			}
			return true, nil
		},
		syncClassificatorUpserter: func(
			_ context.Context,
			id pgtype.UUID,
			title string,
			manufacturer any,
			researchType any,
			registrationCertificate json.RawMessage,
			maintenanceRegulations json.RawMessage,
			attachments []string,
			images []string,
		) (bool, error) {
			gotClassificatorID = id
			gotTitle = title
			gotManufacturer = manufacturer
			gotResearchType = researchType
			gotRegistrationCertificate = registrationCertificate
			gotMaintenanceRegulations = maintenanceRegulations
			gotAttachments = attachments
			gotImages = images
			return true, nil
		},
	}

	statusCode, response, err := srv.processClassificatorSyncRequest(context.Background(), "172.30.240.4:50520", syncClassificatorRequest{
		ID:                      classificatorID.String(),
		Title:                   "  TG16B-21 Центрифуга  ",
		Manufacturer:            json.RawMessage(`{"id":"` + manufacturerID.String() + `","title":"Ignored"}`),
		ResearchType:            json.RawMessage(`"` + researchTypeID.String() + `"`),
		RegistrationCertificate: json.RawMessage(`{"number":"RU-1"}`),
		MaintenanceRegulations:  json.RawMessage(`[{"kind":"yearly"}]`),
		Attachments:             []string{" manual.pdf ", "", "spec.docx"},
		Images:                  []string{" photo-1.jpg ", " "},
	})
	if err != nil {
		t.Fatalf("process classificator sync: %v", err)
	}
	if statusCode != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, statusCode)
	}
	if gotClassificatorID != classificatorID {
		t.Fatalf("expected upserter to receive classificator id %s, got %s", classificatorID.String(), gotClassificatorID.String())
	}
	if gotTitle != "TG16B-21 Центрифуга" {
		t.Fatalf("expected trimmed title, got %q", gotTitle)
	}
	if gotManufacturerID, ok := gotManufacturer.(pgtype.UUID); !ok || gotManufacturerID != manufacturerID {
		t.Fatalf("expected manufacturer uuid, got %#v", gotManufacturer)
	}
	if gotResearchTypeID, ok := gotResearchType.(pgtype.UUID); !ok || gotResearchTypeID != researchTypeID {
		t.Fatalf("expected research type uuid, got %#v", gotResearchType)
	}
	if compactSyncLogPayload(gotRegistrationCertificate) != `{"number":"RU-1"}` {
		t.Fatalf("unexpected registration certificate payload: %s", compactSyncLogPayload(gotRegistrationCertificate))
	}
	if compactSyncLogPayload(gotMaintenanceRegulations) != `[{"kind":"yearly"}]` {
		t.Fatalf("unexpected maintenance regulations payload: %s", compactSyncLogPayload(gotMaintenanceRegulations))
	}
	if len(gotAttachments) != 2 || gotAttachments[0] != "manual.pdf" || gotAttachments[1] != "spec.docx" {
		t.Fatalf("unexpected attachments: %#v", gotAttachments)
	}
	if len(gotImages) != 1 || gotImages[0] != "photo-1.jpg" {
		t.Fatalf("unexpected images: %#v", gotImages)
	}
	if !response.Created || response.ID != classificatorID.String() || response.Title != "TG16B-21 Центрифуга" {
		t.Fatalf("unexpected classificator response: %+v", response)
	}
	if response.Manufacturer == nil || *response.Manufacturer != manufacturerID.String() {
		t.Fatalf("expected manufacturer id in response, got %+v", response)
	}
	if response.ResearchType == nil || *response.ResearchType != researchTypeID.String() {
		t.Fatalf("expected research type id in response, got %+v", response)
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

func TestShouldRestrictRequesterToExecutorTickets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		role string
		want bool
	}{
		{name: "engineer user role is restricted", role: "user", want: true},
		{name: "role comparison is case insensitive", role: " User ", want: true},
		{name: "coordinator keeps broad visibility", role: "coordinator", want: false},
		{name: "admin keeps broad visibility", role: "admin", want: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldRestrictRequesterToExecutorTickets(tc.role); got != tc.want {
				t.Fatalf("expected %t for role %q, got %t", tc.want, tc.role, got)
			}
		})
	}
}

func TestRequesterTicketExecutorFilterRestrictsUserRole(t *testing.T) {
	t.Parallel()

	requesterID := mustUUID(t, "11111111-1111-1111-1111-111111111111")
	srv := &Server{
		requesterRoleLookup: func(_ context.Context, gotRequesterID pgtype.UUID) (string, error) {
			if gotRequesterID != requesterID {
				t.Fatalf("expected requester id %s, got %s", requesterID.String(), gotRequesterID.String())
			}
			return "user", nil
		},
	}

	filter, err := srv.requesterTicketExecutorFilter(context.Background(), requesterID)
	if err != nil {
		t.Fatalf("requesterTicketExecutorFilter returned error: %v", err)
	}
	if filter == nil {
		t.Fatal("expected executor filter for user role")
	}
	if *filter != requesterID {
		t.Fatalf("expected executor filter %s, got %s", requesterID.String(), filter.String())
	}
}

func TestRequesterTicketExecutorFilterSkipsCoordinatorRole(t *testing.T) {
	t.Parallel()

	requesterID := mustUUID(t, "11111111-1111-1111-1111-111111111111")
	srv := &Server{
		requesterRoleLookup: func(_ context.Context, _ pgtype.UUID) (string, error) {
			return "coordinator", nil
		},
	}

	filter, err := srv.requesterTicketExecutorFilter(context.Background(), requesterID)
	if err != nil {
		t.Fatalf("requesterTicketExecutorFilter returned error: %v", err)
	}
	if filter != nil {
		t.Fatalf("expected no executor filter for coordinator role, got %s", filter.String())
	}
}
