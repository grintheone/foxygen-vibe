package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func allowEditorAccess(_ http.ResponseWriter, _ *http.Request) (pgtype.UUID, bool) {
	return pgtype.UUID{Bytes: [16]byte{1}, Valid: true}, true
}

func editorAccessToken(t *testing.T, subject string) string {
	t.Helper()

	token, err := signJWT(testAuthConfig().jwtSecret, accessTokenClaims{
		Subject:   subject,
		Username:  "editor.user",
		TokenType: accessTokenType,
		ExpiresAt: 4102444800, // January 1, 2100 UTC
		IssuedAt:  1704067200, // January 1, 2024 UTC
	})
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	return token
}

func mustUUID(t *testing.T, raw string) pgtype.UUID {
	t.Helper()

	var value pgtype.UUID
	if err := value.Scan(raw); err != nil {
		t.Fatalf("scan uuid: %v", err)
	}

	return value
}

func stringPointer(value string) *string {
	return &value
}

func TestRequireEditorAccessAllowsCoordinator(t *testing.T) {
	t.Parallel()

	var gotUserID pgtype.UUID
	srv := &Server{
		auth: testAuthConfig(),
		editorRoleLookup: func(_ context.Context, userID pgtype.UUID) (string, error) {
			gotUserID = userID
			return "coordinator", nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients", nil)
	req.Header.Set("Authorization", "Bearer "+editorAccessToken(t, "11111111-1111-1111-1111-111111111111"))
	rec := httptest.NewRecorder()

	requesterID, ok := srv.requireEditorAccess(rec, req)

	if !ok {
		t.Fatal("expected editor access to be granted")
	}
	if !requesterID.Valid {
		t.Fatal("expected requester id to be returned")
	}
	if requesterID != gotUserID {
		t.Fatal("expected role lookup to receive the parsed token subject")
	}
}

func TestRequireEditorAccessAllowsAdmin(t *testing.T) {
	t.Parallel()

	srv := &Server{
		auth: testAuthConfig(),
		editorRoleLookup: func(_ context.Context, _ pgtype.UUID) (string, error) {
			return "admin", nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients", nil)
	req.Header.Set("Authorization", "Bearer "+editorAccessToken(t, "11111111-1111-1111-1111-111111111111"))
	rec := httptest.NewRecorder()

	_, ok := srv.requireEditorAccess(rec, req)

	if !ok {
		t.Fatal("expected admin editor access to be granted")
	}
}

func TestRequireEditorAccessRejectsForbiddenRole(t *testing.T) {
	t.Parallel()

	srv := &Server{
		auth: testAuthConfig(),
		editorRoleLookup: func(_ context.Context, _ pgtype.UUID) (string, error) {
			return "user", nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients", nil)
	req.Header.Set("Authorization", "Bearer "+editorAccessToken(t, "11111111-1111-1111-1111-111111111111"))
	rec := httptest.NewRecorder()

	_, ok := srv.requireEditorAccess(rec, req)

	if ok {
		t.Fatal("expected editor access to be rejected")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "forbidden") {
		t.Fatalf("expected forbidden error, got %q", body)
	}
}

func TestRequireEditorAccessRejectsUnknownProfile(t *testing.T) {
	t.Parallel()

	srv := &Server{
		auth: testAuthConfig(),
		editorRoleLookup: func(_ context.Context, _ pgtype.UUID) (string, error) {
			return "", pgx.ErrNoRows
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients", nil)
	req.Header.Set("Authorization", "Bearer "+editorAccessToken(t, "11111111-1111-1111-1111-111111111111"))
	rec := httptest.NewRecorder()

	_, ok := srv.requireEditorAccess(rec, req)

	if ok {
		t.Fatal("expected editor access to be rejected")
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "profile not found") {
		t.Fatalf("expected missing profile error, got %q", body)
	}
}

func TestRequireEditorAccessHandlesRoleLookupFailure(t *testing.T) {
	t.Parallel()

	srv := &Server{
		auth: testAuthConfig(),
		editorRoleLookup: func(_ context.Context, _ pgtype.UUID) (string, error) {
			return "", errors.New("boom")
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients", nil)
	req.Header.Set("Authorization", "Bearer "+editorAccessToken(t, "11111111-1111-1111-1111-111111111111"))
	rec := httptest.NewRecorder()

	_, ok := srv.requireEditorAccess(rec, req)

	if ok {
		t.Fatal("expected editor access to be rejected")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "failed to verify editor access") {
		t.Fatalf("expected role lookup failure error, got %q", body)
	}
}

func TestRequireEditorAccessRejectsInvalidSubjectUUID(t *testing.T) {
	t.Parallel()

	srv := &Server{
		auth: testAuthConfig(),
		editorRoleLookup: func(_ context.Context, _ pgtype.UUID) (string, error) {
			t.Fatal("editor role lookup should not be called for invalid subjects")
			return "", nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients", nil)
	req.Header.Set("Authorization", "Bearer "+editorAccessToken(t, "not-a-uuid"))
	rec := httptest.NewRecorder()

	_, ok := srv.requireEditorAccess(rec, req)

	if ok {
		t.Fatal("expected editor access to be rejected")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "invalid access token") {
		t.Fatalf("expected invalid token error, got %q", body)
	}
}

func TestEditorClientsEndpointRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients?limit=abc", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "limit must be a positive integer") {
		t.Fatalf("expected invalid limit error, got %q", body)
	}
}

func TestEditorClientsEndpointRejectsNonPositiveLimit(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients?limit=0", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "limit must be a positive integer") {
		t.Fatalf("expected invalid limit error, got %q", body)
	}
}

func TestEditorClientByIDRejectsNestedPath(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients/11111111-1111-1111-1111-111111111111/extra", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorClientByIDRejectsInvalidID(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "invalid client id") {
		t.Fatalf("expected invalid client id error, got %q", body)
	}
}

func TestEditorClientByIDRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodDelete, "/api/editor/clients/11111111-1111-1111-1111-111111111111", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestEditorClientPatchRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/clients/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"title":"Acme Labs","extra":true}`))
	rec := httptest.NewRecorder()

	srv.handleEditorClientPatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestEditorClientPatchRequiresTitle(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/clients/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"title":" ","address":"Somewhere"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorClientPatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "title is required") {
		t.Fatalf("expected missing title error, got %q", body)
	}
}

func TestEditorClientPatchRejectsInvalidRegionUUID(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/clients/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"title":"Acme Labs","region":"not-a-uuid"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorClientPatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "region must be a valid UUID") {
		t.Fatalf("expected invalid region error, got %q", body)
	}
}

func TestEditorClientPatchRejectsInvalidLocationJSON(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/clients/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"title":"Acme Labs","location":"{"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorClientPatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "location must be valid JSON") {
		t.Fatalf("expected invalid location error, got %q", body)
	}
}

func TestEditorClientPatchReturnsUpdatedClient(t *testing.T) {
	t.Parallel()

	clientID := mustUUID(t, "11111111-1111-1111-1111-111111111111")
	regionID := mustUUID(t, "22222222-2222-2222-2222-222222222222")

	var (
		gotClientID pgtype.UUID
		gotTitle    string
		gotAddress  string
		gotRegion   any
		gotLocation any
	)

	srv := &Server{
		editorRegionExists: func(_ context.Context, id pgtype.UUID) (bool, error) {
			if id != regionID {
				t.Fatalf("expected region lookup id %s, got %s", regionID.String(), id.String())
			}
			return true, nil
		},
		editorClientUpdater: func(_ context.Context, id pgtype.UUID, title string, address string, region any, location any) (int64, error) {
			gotClientID = id
			gotTitle = title
			gotAddress = address
			gotRegion = region
			gotLocation = location
			return 1, nil
		},
		editorClientDetailLoader: func(_ context.Context, id pgtype.UUID) (editorClientDetailResponse, bool, error) {
			if id != clientID {
				t.Fatalf("expected reloaded client id %s, got %s", clientID.String(), id.String())
			}
			return editorClientDetailResponse{
				ID:          clientID.String(),
				Title:       "Acme Labs",
				Address:     "Bangkok",
				Region:      stringPointer(regionID.String()),
				RegionTitle: "Central",
				Location:    json.RawMessage(`{"lat":13.7}`),
			}, true, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/clients/"+clientID.String(), strings.NewReader(`{"title":"  Acme Labs  ","address":"  Bangkok  ","region":"22222222-2222-2222-2222-222222222222","location":"{\"lat\":13.7}"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorClientPatch(rec, req, clientID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if gotClientID != clientID {
		t.Fatal("expected updater to receive target client id")
	}
	if gotTitle != "Acme Labs" || gotAddress != "Bangkok" {
		t.Fatalf("expected trimmed title/address, got %q / %q", gotTitle, gotAddress)
	}

	gotRegionID, ok := gotRegion.(pgtype.UUID)
	if !ok || gotRegionID != regionID {
		t.Fatalf("expected updater region to be parsed uuid, got %#v", gotRegion)
	}

	gotLocationJSON, ok := gotLocation.(json.RawMessage)
	if !ok || string(gotLocationJSON) != `{"lat":13.7}` {
		t.Fatalf("expected updater location JSON, got %#v", gotLocation)
	}

	body := rec.Body.String()
	for _, want := range []string{`"title":"Acme Labs"`, `"address":"Bangkok"`, `"regionTitle":"Central"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}

func TestEditorClientDetailReturnsClient(t *testing.T) {
	t.Parallel()

	clientID := mustUUID(t, "77777777-7777-7777-7777-777777777777")
	srv := &Server{
		editorClientDetailLoader: func(_ context.Context, id pgtype.UUID) (editorClientDetailResponse, bool, error) {
			if id != clientID {
				t.Fatalf("expected client detail id %s, got %s", clientID.String(), id.String())
			}
			return editorClientDetailResponse{
				ID:          clientID.String(),
				Title:       "North Lab",
				Address:     "Chiang Mai",
				RegionTitle: "North",
			}, true, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients/"+clientID.String(), nil)
	rec := httptest.NewRecorder()

	srv.handleEditorClientDetail(rec, req, clientID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"title":"North Lab"`) {
		t.Fatalf("expected response body to contain client title, got %s", body)
	}
}

func TestEditorClientDetailReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv := &Server{
		editorClientDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorClientDetailResponse, bool, error) {
			return editorClientDetailResponse{}, false, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients/77777777-7777-7777-7777-777777777777", nil)
	rec := httptest.NewRecorder()

	srv.handleEditorClientDetail(rec, req, mustUUID(t, "77777777-7777-7777-7777-777777777777"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorClientDetailHandlesLoaderFailure(t *testing.T) {
	t.Parallel()

	srv := &Server{
		editorClientDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorClientDetailResponse, bool, error) {
			return editorClientDetailResponse{}, false, errors.New("boom")
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/clients/77777777-7777-7777-7777-777777777777", nil)
	rec := httptest.NewRecorder()

	srv.handleEditorClientDetail(rec, req, mustUUID(t, "77777777-7777-7777-7777-777777777777"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestEditorClientPatchReturnsNotFoundWhenUpdaterAffectsNoRows(t *testing.T) {
	t.Parallel()

	clientID := mustUUID(t, "88888888-8888-8888-8888-888888888888")
	srv := &Server{
		editorClientUpdater: func(_ context.Context, _ pgtype.UUID, _ string, _ string, _ any, _ any) (int64, error) {
			return 0, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/clients/"+clientID.String(), strings.NewReader(`{"title":"Acme Labs"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorClientPatch(rec, req, clientID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "client not found") {
		t.Fatalf("expected not found error, got %q", body)
	}
}

func TestEditorClientPatchReturnsNotFoundWhenReloadMisses(t *testing.T) {
	t.Parallel()

	clientID := mustUUID(t, "88888888-8888-8888-8888-888888888888")
	srv := &Server{
		editorClientUpdater: func(_ context.Context, _ pgtype.UUID, _ string, _ string, _ any, _ any) (int64, error) {
			return 1, nil
		},
		editorClientDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorClientDetailResponse, bool, error) {
			return editorClientDetailResponse{}, false, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/clients/"+clientID.String(), strings.NewReader(`{"title":"Acme Labs"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorClientPatch(rec, req, clientID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorClientPatchHandlesReloadFailure(t *testing.T) {
	t.Parallel()

	clientID := mustUUID(t, "88888888-8888-8888-8888-888888888888")
	srv := &Server{
		editorClientUpdater: func(_ context.Context, _ pgtype.UUID, _ string, _ string, _ any, _ any) (int64, error) {
			return 1, nil
		},
		editorClientDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorClientDetailResponse, bool, error) {
			return editorClientDetailResponse{}, false, errors.New("boom")
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/clients/"+clientID.String(), strings.NewReader(`{"title":"Acme Labs"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorClientPatch(rec, req, clientID)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestEditorContactPatchRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/contacts/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"name":"Alice","client":"11111111-1111-1111-1111-111111111111","extra":true}`))
	rec := httptest.NewRecorder()

	srv.handleEditorContactPatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestEditorContactPatchRequiresName(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/contacts/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"name":" ","client":"11111111-1111-1111-1111-111111111111"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorContactPatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "name is required") {
		t.Fatalf("expected missing name error, got %q", body)
	}
}

func TestEditorContactPatchRequiresClient(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/contacts/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"name":"Alice","client":" "}`))
	rec := httptest.NewRecorder()

	srv.handleEditorContactPatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "client is required") {
		t.Fatalf("expected missing client error, got %q", body)
	}
}

func TestEditorContactPatchRejectsInvalidClientUUID(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/contacts/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"name":"Alice","client":"not-a-uuid"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorContactPatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "client must be a valid UUID") {
		t.Fatalf("expected invalid client error, got %q", body)
	}
}

func TestEditorContactPatchReturnsUpdatedContact(t *testing.T) {
	t.Parallel()

	contactID := mustUUID(t, "33333333-3333-3333-3333-333333333333")
	clientID := mustUUID(t, "44444444-4444-4444-4444-444444444444")

	var (
		gotContactID pgtype.UUID
		gotName      string
		gotPosition  string
		gotPhone     string
		gotEmail     string
		gotClientID  pgtype.UUID
	)

	srv := &Server{
		editorClientExists: func(_ context.Context, id pgtype.UUID) (bool, error) {
			if id != clientID {
				t.Fatalf("expected client lookup id %s, got %s", clientID.String(), id.String())
			}
			return true, nil
		},
		editorContactUpdater: func(_ context.Context, id pgtype.UUID, name string, position string, phone string, email string, linkedClientID pgtype.UUID) (int64, error) {
			gotContactID = id
			gotName = name
			gotPosition = position
			gotPhone = phone
			gotEmail = email
			gotClientID = linkedClientID
			return 1, nil
		},
		editorContactDetailLoader: func(_ context.Context, id pgtype.UUID) (editorContactDetailResponse, bool, error) {
			if id != contactID {
				t.Fatalf("expected reloaded contact id %s, got %s", contactID.String(), id.String())
			}
			return editorContactDetailResponse{
				ID:         contactID.String(),
				Name:       "Alice Chen",
				Position:   "Lab Lead",
				Phone:      "+66 1234",
				Email:      "alice@example.com",
				Client:     stringPointer(clientID.String()),
				ClientName: "Acme Labs",
			}, true, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/contacts/"+contactID.String(), strings.NewReader(`{"name":"  Alice Chen  ","position":"  Lab Lead  ","phone":"  +66 1234  ","email":"  alice@example.com  ","client":"44444444-4444-4444-4444-444444444444"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorContactPatch(rec, req, contactID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if gotContactID != contactID || gotClientID != clientID {
		t.Fatal("expected updater to receive contact and client ids")
	}
	if gotName != "Alice Chen" || gotPosition != "Lab Lead" || gotPhone != "+66 1234" || gotEmail != "alice@example.com" {
		t.Fatalf("expected trimmed contact fields, got %q / %q / %q / %q", gotName, gotPosition, gotPhone, gotEmail)
	}

	body := rec.Body.String()
	for _, want := range []string{`"name":"Alice Chen"`, `"clientName":"Acme Labs"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}

func TestEditorContactDetailReturnsContact(t *testing.T) {
	t.Parallel()

	contactID := mustUUID(t, "99999999-9999-9999-9999-999999999999")
	srv := &Server{
		editorContactDetailLoader: func(_ context.Context, id pgtype.UUID) (editorContactDetailResponse, bool, error) {
			if id != contactID {
				t.Fatalf("expected contact detail id %s, got %s", contactID.String(), id.String())
			}
			return editorContactDetailResponse{
				ID:         contactID.String(),
				Name:       "Nina Park",
				ClientName: "North Lab",
			}, true, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/contacts/"+contactID.String(), nil)
	rec := httptest.NewRecorder()

	srv.handleEditorContactDetail(rec, req, contactID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"name":"Nina Park"`) {
		t.Fatalf("expected response body to contain contact name, got %s", body)
	}
}

func TestEditorContactDetailReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv := &Server{
		editorContactDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorContactDetailResponse, bool, error) {
			return editorContactDetailResponse{}, false, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/contacts/99999999-9999-9999-9999-999999999999", nil)
	rec := httptest.NewRecorder()

	srv.handleEditorContactDetail(rec, req, mustUUID(t, "99999999-9999-9999-9999-999999999999"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorContactDetailHandlesLoaderFailure(t *testing.T) {
	t.Parallel()

	srv := &Server{
		editorContactDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorContactDetailResponse, bool, error) {
			return editorContactDetailResponse{}, false, errors.New("boom")
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/contacts/99999999-9999-9999-9999-999999999999", nil)
	rec := httptest.NewRecorder()

	srv.handleEditorContactDetail(rec, req, mustUUID(t, "99999999-9999-9999-9999-999999999999"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestEditorContactPatchReturnsNotFoundWhenUpdaterAffectsNoRows(t *testing.T) {
	t.Parallel()

	contactID := mustUUID(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	clientID := mustUUID(t, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	srv := &Server{
		editorClientExists: func(_ context.Context, _ pgtype.UUID) (bool, error) {
			return true, nil
		},
		editorContactUpdater: func(_ context.Context, _ pgtype.UUID, _ string, _ string, _ string, _ string, _ pgtype.UUID) (int64, error) {
			return 0, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/contacts/"+contactID.String(), strings.NewReader(`{"name":"Alice","client":"`+clientID.String()+`"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorContactPatch(rec, req, contactID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorContactPatchReturnsNotFoundWhenReloadMisses(t *testing.T) {
	t.Parallel()

	contactID := mustUUID(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	clientID := mustUUID(t, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	srv := &Server{
		editorClientExists: func(_ context.Context, _ pgtype.UUID) (bool, error) {
			return true, nil
		},
		editorContactUpdater: func(_ context.Context, _ pgtype.UUID, _ string, _ string, _ string, _ string, _ pgtype.UUID) (int64, error) {
			return 1, nil
		},
		editorContactDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorContactDetailResponse, bool, error) {
			return editorContactDetailResponse{}, false, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/contacts/"+contactID.String(), strings.NewReader(`{"name":"Alice","client":"`+clientID.String()+`"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorContactPatch(rec, req, contactID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorContactPatchHandlesReloadFailure(t *testing.T) {
	t.Parallel()

	contactID := mustUUID(t, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	clientID := mustUUID(t, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	srv := &Server{
		editorClientExists: func(_ context.Context, _ pgtype.UUID) (bool, error) {
			return true, nil
		},
		editorContactUpdater: func(_ context.Context, _ pgtype.UUID, _ string, _ string, _ string, _ string, _ pgtype.UUID) (int64, error) {
			return 1, nil
		},
		editorContactDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorContactDetailResponse, bool, error) {
			return editorContactDetailResponse{}, false, errors.New("boom")
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/contacts/"+contactID.String(), strings.NewReader(`{"name":"Alice","client":"`+clientID.String()+`"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorContactPatch(rec, req, contactID)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestEditorContactByIDRejectsNestedPath(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/contacts/11111111-1111-1111-1111-111111111111/extra", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorContactByIDRejectsInvalidID(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/contacts/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "invalid contact id") {
		t.Fatalf("expected invalid contact id error, got %q", body)
	}
}

func TestEditorContactByIDRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodDelete, "/api/editor/contacts/11111111-1111-1111-1111-111111111111", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestEditorDevicePatchRejectsInvalidBody(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/devices/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"serialNumber":"SN-1","extra":true}`))
	rec := httptest.NewRecorder()

	srv.handleEditorDevicePatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestEditorDevicePatchRejectsInvalidClassificatorUUID(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/devices/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"classificator":"not-a-uuid","serialNumber":"SN-1"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorDevicePatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "classificator must be a valid UUID") {
		t.Fatalf("expected invalid classificator error, got %q", body)
	}
}

func TestEditorDevicePatchRejectsInvalidPropertiesJSON(t *testing.T) {
	t.Parallel()

	srv := &Server{}
	req := httptest.NewRequest(http.MethodPatch, "/api/editor/devices/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"serialNumber":"SN-1","properties":"{"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorDevicePatch(rec, req, pgtype.UUID{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "properties must be valid JSON") {
		t.Fatalf("expected invalid properties error, got %q", body)
	}
}

func TestEditorDevicePatchReturnsUpdatedDevice(t *testing.T) {
	t.Parallel()

	deviceID := mustUUID(t, "55555555-5555-5555-5555-555555555555")
	classificatorID := mustUUID(t, "66666666-6666-6666-6666-666666666666")

	var (
		gotDeviceID       pgtype.UUID
		gotClassificator  any
		gotSerialNumber   string
		gotProperties     json.RawMessage
		gotConnectedToLis bool
		gotIsUsed         bool
	)

	srv := &Server{
		editorClassificatorExists: func(_ context.Context, id pgtype.UUID) (bool, error) {
			if id != classificatorID {
				t.Fatalf("expected classificator lookup id %s, got %s", classificatorID.String(), id.String())
			}
			return true, nil
		},
		editorDeviceUpdater: func(_ context.Context, id pgtype.UUID, classificator any, serialNumber string, properties json.RawMessage, connectedToLis bool, isUsed bool) (int64, error) {
			gotDeviceID = id
			gotClassificator = classificator
			gotSerialNumber = serialNumber
			gotProperties = properties
			gotConnectedToLis = connectedToLis
			gotIsUsed = isUsed
			return 1, nil
		},
		editorDeviceDetailLoader: func(_ context.Context, id pgtype.UUID) (editorDeviceDetailResponse, bool, error) {
			if id != deviceID {
				t.Fatalf("expected reloaded device id %s, got %s", deviceID.String(), id.String())
			}
			return editorDeviceDetailResponse{
				ID:             deviceID.String(),
				Title:          "Analyzer X",
				SerialNumber:   "SN-42",
				Classificator:  stringPointer(classificatorID.String()),
				ConnectedToLis: true,
				IsUsed:         true,
				Properties:     json.RawMessage(`{"rack":2}`),
			}, true, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/devices/"+deviceID.String(), strings.NewReader(`{"classificator":"66666666-6666-6666-6666-666666666666","serialNumber":"  SN-42  ","properties":"{\"rack\":2}","connectedToLis":true,"isUsed":true}`))
	rec := httptest.NewRecorder()

	srv.handleEditorDevicePatch(rec, req, deviceID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if gotDeviceID != deviceID {
		t.Fatal("expected updater to receive target device id")
	}
	gotClassificatorID, ok := gotClassificator.(pgtype.UUID)
	if !ok || gotClassificatorID != classificatorID {
		t.Fatalf("expected updater classificator to be parsed uuid, got %#v", gotClassificator)
	}
	if gotSerialNumber != "SN-42" {
		t.Fatalf("expected trimmed serial number, got %q", gotSerialNumber)
	}
	if string(gotProperties) != `{"rack":2}` || !gotConnectedToLis || !gotIsUsed {
		t.Fatalf("unexpected device updater payload: properties=%s connected=%v used=%v", string(gotProperties), gotConnectedToLis, gotIsUsed)
	}

	body := rec.Body.String()
	for _, want := range []string{`"title":"Analyzer X"`, `"serialNumber":"SN-42"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected response body to contain %q, got %s", want, body)
		}
	}
}

func TestEditorDeviceDetailReturnsDevice(t *testing.T) {
	t.Parallel()

	deviceID := mustUUID(t, "cccccccc-cccc-cccc-cccc-cccccccccccc")
	srv := &Server{
		editorDeviceDetailLoader: func(_ context.Context, id pgtype.UUID) (editorDeviceDetailResponse, bool, error) {
			if id != deviceID {
				t.Fatalf("expected device detail id %s, got %s", deviceID.String(), id.String())
			}
			return editorDeviceDetailResponse{
				ID:           deviceID.String(),
				Title:        "Analyzer Q",
				SerialNumber: "SN-Q",
			}, true, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/devices/"+deviceID.String(), nil)
	rec := httptest.NewRecorder()

	srv.handleEditorDeviceDetail(rec, req, deviceID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"title":"Analyzer Q"`) {
		t.Fatalf("expected response body to contain device title, got %s", body)
	}
}

func TestEditorDeviceDetailReturnsNotFound(t *testing.T) {
	t.Parallel()

	srv := &Server{
		editorDeviceDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorDeviceDetailResponse, bool, error) {
			return editorDeviceDetailResponse{}, false, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/devices/cccccccc-cccc-cccc-cccc-cccccccccccc", nil)
	rec := httptest.NewRecorder()

	srv.handleEditorDeviceDetail(rec, req, mustUUID(t, "cccccccc-cccc-cccc-cccc-cccccccccccc"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorDeviceDetailHandlesLoaderFailure(t *testing.T) {
	t.Parallel()

	srv := &Server{
		editorDeviceDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorDeviceDetailResponse, bool, error) {
			return editorDeviceDetailResponse{}, false, errors.New("boom")
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/editor/devices/cccccccc-cccc-cccc-cccc-cccccccccccc", nil)
	rec := httptest.NewRecorder()

	srv.handleEditorDeviceDetail(rec, req, mustUUID(t, "cccccccc-cccc-cccc-cccc-cccccccccccc"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestEditorDevicePatchReturnsNotFoundWhenUpdaterAffectsNoRows(t *testing.T) {
	t.Parallel()

	deviceID := mustUUID(t, "dddddddd-dddd-dddd-dddd-dddddddddddd")
	srv := &Server{
		editorDeviceUpdater: func(_ context.Context, _ pgtype.UUID, _ any, _ string, _ json.RawMessage, _ bool, _ bool) (int64, error) {
			return 0, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/devices/"+deviceID.String(), strings.NewReader(`{"serialNumber":"SN-1"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorDevicePatch(rec, req, deviceID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorDevicePatchReturnsNotFoundWhenReloadMisses(t *testing.T) {
	t.Parallel()

	deviceID := mustUUID(t, "dddddddd-dddd-dddd-dddd-dddddddddddd")
	srv := &Server{
		editorDeviceUpdater: func(_ context.Context, _ pgtype.UUID, _ any, _ string, _ json.RawMessage, _ bool, _ bool) (int64, error) {
			return 1, nil
		},
		editorDeviceDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorDeviceDetailResponse, bool, error) {
			return editorDeviceDetailResponse{}, false, nil
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/devices/"+deviceID.String(), strings.NewReader(`{"serialNumber":"SN-1"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorDevicePatch(rec, req, deviceID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorDevicePatchHandlesReloadFailure(t *testing.T) {
	t.Parallel()

	deviceID := mustUUID(t, "dddddddd-dddd-dddd-dddd-dddddddddddd")
	srv := &Server{
		editorDeviceUpdater: func(_ context.Context, _ pgtype.UUID, _ any, _ string, _ json.RawMessage, _ bool, _ bool) (int64, error) {
			return 1, nil
		},
		editorDeviceDetailLoader: func(_ context.Context, _ pgtype.UUID) (editorDeviceDetailResponse, bool, error) {
			return editorDeviceDetailResponse{}, false, errors.New("boom")
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/editor/devices/"+deviceID.String(), strings.NewReader(`{"serialNumber":"SN-1"}`))
	rec := httptest.NewRecorder()

	srv.handleEditorDevicePatch(rec, req, deviceID)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestEditorDeviceByIDRejectsNestedPath(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/devices/11111111-1111-1111-1111-111111111111/extra", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestEditorDeviceByIDRejectsInvalidID(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodGet, "/api/editor/devices/not-a-uuid", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "invalid device id") {
		t.Fatalf("expected invalid device id error, got %q", body)
	}
}

func TestEditorDeviceByIDRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	srv := &Server{editorAccessCheck: allowEditorAccess}
	req := httptest.NewRequest(http.MethodDelete, "/api/editor/devices/11111111-1111-1111-1111-111111111111", nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}
