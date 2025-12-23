package security

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/web"
	"golang.org/x/time/rate"
)

// RateLimitConfig configures rate limiting
type RateLimitConfig struct {
	// RequestsPerMinute is the maximum number of requests per minute per client
	RequestsPerMinute int

	// RequestsPerSecond is the maximum number of requests per second per client (alternative to RequestsPerMinute)
	RequestsPerSecond int

	// Burst is the maximum burst size allowed.
	// If 0, a reasonable default is selected based on the configured rate.
	Burst int

	// KeyFunc extracts a key from the request to identify the client
	// Default: uses IP address
	KeyFunc func(ctx *web.FastRequestContext) string

	// SkipFunc allows skipping rate limiting for some requests (e.g., health checks).
	// If it returns true, the request is not rate limited.
	SkipFunc func(ctx *web.FastRequestContext) bool

	// EntryTTL controls how long an idle client limiter is kept in memory.
	// If 0, defaults to 10 minutes.
	EntryTTL time.Duration

	// CleanupInterval controls how often idle entries are evicted (best-effort, on-request).
	// If 0, defaults to 1 minute.
	CleanupInterval time.Duration

	// DisableRetryAfter disables setting `Retry-After` on 429 responses.
	// Default: false (i.e., Retry-After is included).
	DisableRetryAfter bool

	// OnLimitReached is called when rate limit is exceeded
	// If nil, returns 429 Too Many Requests
	OnLimitReached func(ctx *web.FastRequestContext) error
}

// DefaultRateLimitConfig returns a default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 100,
		KeyFunc: func(ctx *web.FastRequestContext) string {
			// Use IP address as key
			return ctx.RequestCtx.RemoteIP().String()
		},
	}
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// limiterStore keeps per-client limiters for a single middleware instance (i.e., per-route config).
type limiterStore struct {
	mu              sync.Mutex
	clients         map[string]*clientLimiter
	lastCleanup     time.Time
	ttl             time.Duration
	cleanupInterval time.Duration
	limit           rate.Limit
	burst           int
}

func newLimiterStore(limit rate.Limit, burst int, ttl, cleanupInterval time.Duration) *limiterStore {
	return &limiterStore{
		clients:         make(map[string]*clientLimiter),
		lastCleanup:     time.Now(),
		ttl:             ttl,
		cleanupInterval: cleanupInterval,
		limit:           limit,
		burst:           burst,
	}
}

func (s *limiterStore) get(key string) *clientLimiter {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Best-effort cleanup on request
	if now.Sub(s.lastCleanup) >= s.cleanupInterval {
		s.lastCleanup = now
		for k, v := range s.clients {
			if now.Sub(v.lastSeen) >= s.ttl {
				delete(s.clients, k)
			}
		}
	}

	if cl, ok := s.clients[key]; ok {
		cl.lastSeen = now
		return cl
	}

	cl := &clientLimiter{
		limiter:  rate.NewLimiter(s.limit, s.burst),
		lastSeen: now,
	}
	s.clients[key] = cl
	return cl
}

// RateLimit middleware enforces rate limiting
func RateLimit(config RateLimitConfig) web.FastMiddleware {
	// Determine rate (tokens/second).
	var tokensPerSecond float64
	switch {
	case config.RequestsPerSecond > 0:
		tokensPerSecond = float64(config.RequestsPerSecond)
	case config.RequestsPerMinute > 0:
		tokensPerSecond = float64(config.RequestsPerMinute) / 60.0
	default:
		tokensPerSecond = float64(DefaultRateLimitConfig().RequestsPerMinute) / 60.0
	}
	if tokensPerSecond <= 0 {
		tokensPerSecond = 1
	}

	limit := rate.Limit(tokensPerSecond)

	// Get key function
	keyFunc := config.KeyFunc
	if keyFunc == nil {
		keyFunc = func(ctx *web.FastRequestContext) string {
			return ctx.RequestCtx.RemoteIP().String()
		}
	}

	skipFunc := config.SkipFunc
	if skipFunc == nil {
		skipFunc = func(ctx *web.FastRequestContext) bool { return false }
	}

	ttl := config.EntryTTL
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	cleanupInterval := config.CleanupInterval
	if cleanupInterval <= 0 {
		cleanupInterval = time.Minute
	}

	burst := config.Burst
	if burst <= 0 {
		// Default burst to ~1 second worth of traffic, clamped to [1, 1000]
		burst = int(math.Ceil(tokensPerSecond))
		if burst < 1 {
			burst = 1
		}
		if burst > 1000 {
			burst = 1000
		}
	}

	includeRetryAfter := !config.DisableRetryAfter

	store := newLimiterStore(limit, burst, ttl, cleanupInterval)

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			if skipFunc(ctx) {
				return next(ctx)
			}

			key := keyFunc(ctx)
			cl := store.get(key)

			if !cl.limiter.Allow() {
				// Rate limit exceeded
				if config.OnLimitReached != nil {
					return config.OnLimitReached(ctx)
				}

				if includeRetryAfter {
					// Compute a best-effort retry delay without consuming a token.
					res := cl.limiter.Reserve()
					delay := res.Delay()
					res.Cancel()
					if delay > 0 && delay != rate.InfDuration {
						seconds := int(math.Ceil(delay.Seconds()))
						if seconds < 1 {
							seconds = 1
						}
						ctx.RequestCtx.Response.Header.Set("Retry-After", fmt.Sprintf("%d", seconds))
					}
				}

				// Default: return 429 Too Many Requests
				ctx.RequestCtx.SetStatusCode(429)
				ctx.RequestCtx.SetContentType("application/json")
				ctx.RequestCtx.WriteString(`{"error":"rate_limit_exceeded","message":"Too many requests"}`)
				return nil
			}

			return next(ctx)
		}
	}
}
