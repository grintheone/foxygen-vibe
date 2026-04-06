package api

import (
	"log"
	"net/http"
	"sync"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

const healthRequestLogInterval = time.Hour

var healthRequestLogLimiter = throttledRequestLogger{}

type throttledRequestLogger struct {
	mu           sync.Mutex
	lastLoggedAt time.Time
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		if !shouldLogRequest(r, recorder.statusCode, startedAt) {
			return
		}

		log.Printf("%s %s %d %s", r.Method, r.URL.Path, recorder.statusCode, time.Since(startedAt))
	})
}

func shouldLogRequest(r *http.Request, statusCode int, now time.Time) bool {
	if r.Method != http.MethodGet || r.URL.Path != "/api/health" {
		return true
	}
	if statusCode != http.StatusOK {
		return true
	}

	return healthRequestLogLimiter.ShouldLog(now, healthRequestLogInterval)
}

func (l *throttledRequestLogger) ShouldLog(now time.Time, interval time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastLoggedAt.IsZero() || now.Sub(l.lastLoggedAt) >= interval {
		l.lastLoggedAt = now
		return true
	}

	return false
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Sync-Secret")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
