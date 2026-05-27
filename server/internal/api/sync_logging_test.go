package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSyncLoggerCreatesFileAndWrites(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "sync.log")

	logger, closer, err := newSyncLogger(path)
	if err != nil {
		t.Fatalf("new sync logger: %v", err)
	}

	logger.Printf("sync test message id=%d", 42)

	if closer == nil {
		t.Fatal("expected sync logger closer")
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("close sync logger: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sync log file: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "sync test message id=42") {
		t.Fatalf("expected sync log file to contain message, got %q", text)
	}
}

func TestCompactSyncLogPayloadRedactsPasswordFields(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"executor": map[string]any{
			"login":    "Andrej.Suvorov",
			"password": "hMWBaUsQ",
		},
		"assignedBy": map[string]any{
			"newPassword": "fresh-secret",
		},
		"events": []any{
			map[string]any{"password_hash": "stored-secret"},
		},
	}

	got := compactSyncLogPayload(payload)
	if strings.Contains(got, "hMWBaUsQ") ||
		strings.Contains(got, "fresh-secret") ||
		strings.Contains(got, "stored-secret") {
		t.Fatalf("expected password values to be redacted, got %s", got)
	}
	for _, want := range []string{
		`"password":"[redacted]"`,
		`"newPassword":"[redacted]"`,
		`"password_hash":"[redacted]"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected redacted payload to contain %s, got %s", want, got)
		}
	}
}
