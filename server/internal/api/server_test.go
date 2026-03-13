package api

import (
	"context"
	"errors"
	"testing"

	"foxygen-vibe/server/internal/storage"
)

func TestConnectStorageContinuesWhenMinIOUnavailable(t *testing.T) {
	previous := newMinIOClient
	newMinIOClient = func(context.Context, storage.Config) (*storage.Client, error) {
		return nil, errors.New(`check bucket "mobile-engineer": Access Denied`)
	}
	defer func() {
		newMinIOClient = previous
	}()

	srv := &Server{storageConfigured: true}
	srv.connectStorage(context.Background(), storage.Config{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		Bucket:          "mobile-engineer",
	})

	if srv.storage != nil {
		t.Fatal("expected server to continue without storage client")
	}
	if !srv.storageConfigured {
		t.Fatal("expected storage to remain marked as configured")
	}
}
