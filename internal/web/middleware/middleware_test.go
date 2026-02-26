package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestChainOrdering(t *testing.T) {
	var order []string

	mw := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}

	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), mw("first"), mw("second"))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	if len(order) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(order))
	}
	if order[0] != "first" || order[1] != "second" || order[2] != "handler" {
		t.Fatalf("unexpected order: %v", order)
	}
}

func TestCORSHeaders(t *testing.T) {
	h := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Allow-Origin *, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, OPTIONS" {
		t.Errorf("expected Allow-Methods 'GET, OPTIONS', got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "HX-Request, HX-Target, HX-Trigger" {
		t.Errorf("expected Allow-Headers with htmx headers, got %q", got)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	called := false
	h := CORS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("OPTIONS", "/", nil))

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204 on preflight, got %d", rec.Code)
	}
	if called {
		t.Error("downstream handler should not be called on preflight")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Allow-Origin * on preflight, got %q", got)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "3600" {
		t.Errorf("expected Access-Control-Max-Age 3600, got %q", got)
	}
}

func TestRateLimitAllows(t *testing.T) {
	cfg := RateLimitConfig{
		Rate:       10,
		Burst:      10,
		StaleAfter: time.Minute,
		CleanEvery: time.Minute,
		ExemptPath: "/events",
	}

	h := RateLimit(context.Background(), cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := range 10 {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}
}

func TestRateLimitRejects(t *testing.T) {
	cfg := RateLimitConfig{
		Rate:       rate.Limit(1),
		Burst:      1,
		StaleAfter: time.Minute,
		CleanEvery: time.Minute,
		ExemptPath: "/events",
	}

	h := RateLimit(context.Background(), cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should succeed (uses burst).
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", rec.Code)
	}

	// Immediate second request should be rejected.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rec.Code)
	}
}

func TestRateLimitSSEExempt(t *testing.T) {
	cfg := RateLimitConfig{
		Rate:       rate.Limit(1),
		Burst:      1,
		StaleAfter: time.Minute,
		CleanEvery: time.Minute,
		ExemptPath: "/events",
	}

	h := RateLimit(context.Background(), cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the limiter on a normal path.
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on normal path, got %d", rec.Code)
	}

	// SSE endpoint should still work.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/events", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("SSE request: expected 200, got %d", rec.Code)
	}
}

func TestRateLimitCleanupStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := RateLimitConfig{
		Rate:       10,
		Burst:      10,
		StaleAfter: time.Millisecond,
		CleanEvery: time.Millisecond,
		ExemptPath: "/events",
	}

	before := runtime.NumGoroutine()
	_ = RateLimit(ctx, cfg)

	// Wait for the goroutine to start.
	time.Sleep(10 * time.Millisecond)
	afterStart := runtime.NumGoroutine()
	if afterStart <= before {
		t.Fatal("expected cleanup goroutine to be running")
	}

	cancel()
	// Wait for the goroutine to exit.
	time.Sleep(50 * time.Millisecond)
	afterCancel := runtime.NumGoroutine()
	if afterCancel > before {
		t.Errorf("expected cleanup goroutine to stop after cancel: before=%d, after=%d", before, afterCancel)
	}
}
