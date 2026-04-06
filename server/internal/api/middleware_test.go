package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShouldLogRequestAlwaysLogsNonHealthRequests(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync", nil)

	if !shouldLogRequest(req, http.StatusCreated, time.Now()) {
		t.Fatal("expected non-health request to be logged")
	}
}

func TestShouldLogRequestAlwaysLogsUnhealthyHealthChecks(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)

	if !shouldLogRequest(req, http.StatusServiceUnavailable, time.Now()) {
		t.Fatal("expected unhealthy health check to be logged")
	}
}

func TestThrottledRequestLoggerAllowsOnlyOneHealthyHealthCheckPerInterval(t *testing.T) {
	t.Parallel()

	logger := &throttledRequestLogger{}
	now := time.Date(2026, time.April, 6, 12, 0, 0, 0, time.UTC)

	if !logger.ShouldLog(now, time.Hour) {
		t.Fatal("expected first healthy health check to be logged")
	}
	if logger.ShouldLog(now.Add(59*time.Minute), time.Hour) {
		t.Fatal("expected healthy health check inside the interval to be suppressed")
	}
	if !logger.ShouldLog(now.Add(time.Hour), time.Hour) {
		t.Fatal("expected healthy health check at the interval boundary to be logged")
	}
}
