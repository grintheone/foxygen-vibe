package moleculer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewReturnsNilWhenDisabled(t *testing.T) {
	client, err := New(Config{})
	if err != nil {
		t.Fatalf("create disabled client: %v", err)
	}
	if client != nil {
		t.Fatal("expected disabled client to be nil")
	}
}

func TestNewValidatesEnabledConfig(t *testing.T) {
	_, err := New(Config{Enabled: true, URL: "nats://localhost:4222", Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "http or https") {
		t.Fatalf("expected HTTP URL validation error, got %v", err)
	}
}

func TestProbeRegistryCountsVisibleNodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != registryNodesPath {
			t.Fatalf("expected path %s, got %s", registryNodesPath, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"sidecar"},{"id":"remote-service"}]`))
	}))
	defer server.Close()

	client, err := New(Config{Enabled: true, URL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	status, err := client.ProbeRegistry(context.Background())
	if err != nil {
		t.Fatalf("probe registry: %v", err)
	}
	if status.NodeCount != 2 {
		t.Fatalf("expected 2 visible nodes, got %d", status.NodeCount)
	}
}

func TestProbeRegistryReturnsSidecarError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "registry unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client, err := New(Config{Enabled: true, URL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	_, err = client.ProbeRegistry(context.Background())
	if err == nil || !strings.Contains(err.Error(), "registry unavailable") {
		t.Fatalf("expected registry error, got %v", err)
	}
}
