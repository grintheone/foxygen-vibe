package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStorageConfigReturnsDisabledWhenUnset(t *testing.T) {
	dir := t.TempDir()
	writeConfigTestEnv(t, dir, "")

	restore := changeWorkingDirectory(t, dir)
	defer restore()

	cfg, err := resolveStorageConfig()
	if err != nil {
		t.Fatalf("resolve storage config: %v", err)
	}

	if cfg.Enabled() {
		t.Fatalf("expected storage to be disabled, got %+v", cfg)
	}
}

func TestResolveStorageConfigRequiresCompleteMinIOSettings(t *testing.T) {
	dir := t.TempDir()
	writeConfigTestEnv(t, dir, "MINIO_ENDPOINT=localhost:9000\nMINIO_ACCESS_KEY=minioadmin\n")

	restore := changeWorkingDirectory(t, dir)
	defer restore()

	if _, err := resolveStorageConfig(); err == nil {
		t.Fatal("expected resolveStorageConfig to fail for incomplete settings")
	}
}

func TestResolveStorageConfigParsesMinIOSettings(t *testing.T) {
	dir := t.TempDir()
	writeConfigTestEnv(t, dir, "MINIO_ENDPOINT=localhost:9000\nMINIO_ACCESS_KEY=minioadmin\nMINIO_SECRET_KEY=minioadmin\nMINIO_BUCKET=foxygen-vibe\nMINIO_USE_SSL=true\nMINIO_REGION=us-east-1\n")

	restore := changeWorkingDirectory(t, dir)
	defer restore()

	cfg, err := resolveStorageConfig()
	if err != nil {
		t.Fatalf("resolve storage config: %v", err)
	}

	if cfg.Endpoint != "localhost:9000" {
		t.Fatalf("expected endpoint localhost:9000, got %q", cfg.Endpoint)
	}
	if cfg.AccessKeyID != "minioadmin" {
		t.Fatalf("expected access key minioadmin, got %q", cfg.AccessKeyID)
	}
	if cfg.SecretAccessKey != "minioadmin" {
		t.Fatalf("expected secret key minioadmin, got %q", cfg.SecretAccessKey)
	}
	if cfg.Bucket != "foxygen-vibe" {
		t.Fatalf("expected bucket foxygen-vibe, got %q", cfg.Bucket)
	}
	if !cfg.UseSSL {
		t.Fatal("expected storage config to enable SSL")
	}
	if cfg.Region != "us-east-1" {
		t.Fatalf("expected region us-east-1, got %q", cfg.Region)
	}
}

func changeWorkingDirectory(t *testing.T, dir string) func() {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	return func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}
}

func writeConfigTestEnv(t *testing.T, dir string, content string) {
	t.Helper()

	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
