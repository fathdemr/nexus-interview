// Package ratelimit provides a Redis-backed failure counter that blocks
// an IP address after too many failed attempts on a sensitive endpoint.
//
// Algorithm (atomic, no Lua required):
//
//	INCR  auth:ratelimit:<action>:<ip>
//	If count == 1 → newly created key → set EXPIRE to window duration
//	If count > maxAttempts → key already has its TTL → return locked + remaining TTL
//
// The window is fixed: the first failure opens a window and the key expires
// naturally. A successful call resets the counter via Reset().
package ratelimit

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fathdemr/nexus-interview/pkg/cache"
	"github.com/fathdemr/nexus-interview/pkg/httputil"
	"github.com/gin-gonic/gin"
)

const keyPrefix = "auth:ratelimit"

// Limiter holds the configuration for one rate-limited action.
type Limiter struct {
	cache       *cache.Service
	maxAttempts int64
	window      time.Duration
}

// New creates a Limiter.
//   - maxAttempts: number of failures allowed before the IP is locked.
//   - window: lock duration (and the sliding window reset period).
func New(cache *cache.Service, maxAttempts int64, window time.Duration) *Limiter {
	return &Limiter{
		cache:       cache,
		maxAttempts: maxAttempts,
		window:      window,
	}
}

// RecordFailure increments the failure counter for the given action+ip pair.
// It returns (locked=true, retryAfter) when the counter exceeds maxAttempts.
// The first call for a fresh key also sets the expiry, so the window is
// anchored to the first failure.
func (l *Limiter) RecordFailure(action, ip string) (locked bool, retryAfter time.Duration) {
	key := buildKey(action, ip)

	count, err := l.cache.Incr(key)
	if err != nil {
		// Redis unavailable — fail open (do not lock the user out).
		return false, 0
	}

	if count == 1 {
		// First failure in this window — set the expiry.
		_ = l.cache.SetExpire(key, l.window)
	}

	if count > l.maxAttempts {
		ttl, _ := l.cache.TTLDuration(key)
		if ttl < 0 {
			ttl = l.window
		}
		return true, ttl
	}

	return false, 0
}

// Reset removes the failure counter after a successful authentication.
// Call this on every successful login / register so a genuine user can
// retry immediately after recovering from a typo.
func (l *Limiter) Reset(action, ip string) {
	_ = l.cache.Delete(buildKey(action, ip))
}

// IsLocked reports whether the given action+ip is currently locked without
// modifying the counter. Useful for a pre-check before expensive operations.
func (l *Limiter) IsLocked(action, ip string) (locked bool, retryAfter time.Duration) {
	key := buildKey(action, ip)

	val, err := l.cache.Get(key)
	if err != nil || val == "" {
		return false, 0
	}

	var count int64
	if _, err := fmt.Sscanf(val, "%d", &count); err != nil || count <= l.maxAttempts {
		return false, 0
	}

	ttl, _ := l.cache.TTLDuration(key)
	if ttl < 0 {
		ttl = l.window
	}
	return true, ttl
}

// Middleware returns a Gin middleware that aborts with 429 when the caller's
// IP has been locked for the given action. Place it before the handler.
//
//	authGroup.POST("/login", ratelimiter.Middleware("login"), handler.login)
func (l *Limiter) Middleware(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		locked, retryAfter := l.IsLocked(action, ip)
		if locked {
			secs := int(retryAfter.Seconds())
			if secs < 1 {
				secs = 1
			}
			c.Header("Retry-After", fmt.Sprintf("%d", secs))
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				httputil.NewErrorResponse(
					fmt.Sprintf("too many failed attempts, try again in %ds", secs),
					"RATE_LIMITED",
				))
			return
		}

		c.Next()
	}
}

func buildKey(action, ip string) string {
	return fmt.Sprintf("%s:%s:%s", keyPrefix, action, ip)
}
