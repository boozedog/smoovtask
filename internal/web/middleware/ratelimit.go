package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig holds rate limiter settings.
type RateLimitConfig struct {
	Rate       rate.Limit
	Burst      int
	StaleAfter time.Duration
	CleanEvery time.Duration
	ExemptPath string
}

// DefaultRateLimitConfig returns sensible defaults for a local dev tool.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Rate:       10,
		Burst:      30,
		StaleAfter: 5 * time.Minute,
		CleanEvery: 3 * time.Minute,
		ExemptPath: "/events",
	}
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit returns per-IP rate limiting middleware.
// The ctx parameter controls the lifetime of the background cleanup goroutine.
func RateLimit(ctx context.Context, cfg RateLimitConfig) func(http.Handler) http.Handler {
	var (
		mu       sync.Mutex
		visitors = make(map[string]*visitor)
	)

	// Background cleanup of stale entries.
	go func() {
		ticker := time.NewTicker(cfg.CleanEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				for ip, v := range visitors {
					if time.Since(v.lastSeen) > cfg.StaleAfter {
						delete(visitors, ip)
					}
				}
				mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == cfg.ExemptPath {
				next.ServeHTTP(w, r)
				return
			}

			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			mu.Lock()
			v, ok := visitors[ip]
			if !ok {
				v = &visitor{limiter: rate.NewLimiter(cfg.Rate, cfg.Burst)}
				visitors[ip] = v
			}
			v.lastSeen = time.Now()
			mu.Unlock()

			if !v.limiter.Allow() {
				slog.Warn("rate limit exceeded", "ip", ip, "path", r.URL.Path) //nolint:gosec // ip is from net.SplitHostPort, not user input
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
