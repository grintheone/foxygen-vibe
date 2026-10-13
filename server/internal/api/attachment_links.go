package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func resolvePublicBaseURL() (string, error) {
	fileEnv, err := loadDotEnv(".env")
	if err != nil {
		return "", err
	}
	value := strings.TrimRight(getConfigValue(fileEnv, "PUBLIC_BASE_URL"), "/")
	if value == "" {
		return "", nil
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("PUBLIC_BASE_URL must be an HTTP(S) URL without credentials, query or fragment")
	}
	return value, nil
}

func (s *Server) attachmentDownloadSignature(ticketID, attachmentID string) string {
	mac := hmac.New(sha256.New, s.auth.jwtSecret)
	mac.Write([]byte("foxygen:attachment-download:v1\x00" + ticketID + "\x00" + attachmentID))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) ticketAttachmentEventURL(ticketID, attachmentID string) (string, error) {
	if s.publicBaseURL == "" || len(s.auth.jwtSecret) == 0 {
		return "", errors.New("PUBLIC_BASE_URL and JWT_SECRET are required for attachment event URLs")
	}
	return s.publicBaseURL + "/api/attachment-download/" + url.PathEscape(ticketID) + "/" + url.PathEscape(attachmentID) +
		"?signature=" + s.attachmentDownloadSignature(ticketID, attachmentID), nil
}

// A signed URL grants read access to one attachment and has no time-based expiry.
func (s *Server) handleSharedTicketAttachmentDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/attachment-download/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.NotFound(w, r)
		return
	}
	var ticketID pgtype.UUID
	if err := ticketID.Scan(parts[0]); err != nil {
		http.NotFound(w, r)
		return
	}
	signature := r.URL.Query().Get("signature")
	if len(s.auth.jwtSecret) == 0 || !hmac.Equal([]byte(signature), []byte(s.attachmentDownloadSignature(parts[0], parts[1]))) {
		http.Error(w, "invalid download signature", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "private, no-store")
	s.handleTicketAttachmentDownload(w, r, ticketID, parts[1])
}
