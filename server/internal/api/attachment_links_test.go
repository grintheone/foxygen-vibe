package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSharedAttachmentLinksRequireMatchingSignature(t *testing.T) {
	t.Parallel()
	srv := &Server{publicBaseURL: "https://foxygen.example.com", auth: authConfig{jwtSecret: []byte("test-secret")}}
	ticketID := "11111111-1111-1111-1111-111111111111"
	link, err := srv.ticketAttachmentEventURL(ticketID, "attachment-1")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "https" || u.Host != "foxygen.example.com" || len(u.Query()) != 1 || u.Query().Get("signature") == "" {
		t.Fatalf("expected absolute signed URL without expiration: %s", link)
	}
	for _, tc := range []struct {
		name, method, target string
		want                 int
	}{
		// A valid link passes authorization and reaches the unconfigured database.
		{"valid without login", http.MethodGet, link, http.StatusServiceUnavailable},
		{"missing signature", http.MethodGet, strings.Split(link, "?")[0], http.StatusUnauthorized},
		{"changed signature", http.MethodGet, link + "0", http.StatusUnauthorized},
		{"other attachment", http.MethodGet, strings.Replace(link, "attachment-1", "attachment-2", 1), http.StatusUnauthorized},
		{"other ticket", http.MethodGet, strings.Replace(link, ticketID, "22222222-2222-2222-2222-222222222222", 1), http.StatusUnauthorized},
		{"write prohibited", http.MethodPost, link, http.StatusMethodNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			srv.handleSharedTicketAttachmentDownload(rec, httptest.NewRequest(tc.method, tc.target, nil))
			if rec.Code != tc.want {
				t.Fatalf("got %d, want %d: %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
	// Links are stable across restarts, and key rotation revokes them.
	restarted := &Server{publicBaseURL: srv.publicBaseURL, auth: srv.auth}
	if again, _ := restarted.ticketAttachmentEventURL(ticketID, "attachment-1"); again != link {
		t.Fatal("link changed across restart")
	}
	restarted.auth.jwtSecret = []byte("rotated-secret")
	rec := httptest.NewRecorder()
	restarted.handleSharedTicketAttachmentDownload(rec, httptest.NewRequest(http.MethodGet, link, nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatal("old link accepted after key rotation")
	}
}

func TestResolvePublicBaseURL(t *testing.T) {
	dir := t.TempDir()
	writeConfigTestEnv(t, dir, "")
	restore := changeWorkingDirectory(t, dir)
	defer restore()
	for _, tc := range []struct {
		value, want string
		invalid     bool
	}{
		{"", "", false},
		{"https://foxygen.example.com/", "https://foxygen.example.com", false},
		{"http://localhost:8080", "http://localhost:8080", false},
		{"/relative", "", true},
		{"ftp://example.com", "", true},
		{"https://user:pass@example.com", "", true},
		{"https://example.com?query=1", "", true},
		{"https://example.com#fragment", "", true},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("PUBLIC_BASE_URL", tc.value)
			got, err := resolvePublicBaseURL()
			if (err != nil) != tc.invalid || got != tc.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}
