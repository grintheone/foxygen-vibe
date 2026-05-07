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
