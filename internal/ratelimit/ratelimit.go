package ratelimit

import (
	"sync"
	"time"
)

const cleanupInterval = time.Minute
const entryTTL = 5 * time.Minute

// RateLimiter tracks the last-seen time per identifier and enforces a minimum
// period between requests for that identifier.
type RateLimiter struct {
	mu       sync.Mutex
	lastSeen map[string]time.Time
	done     chan struct{}
}

func New() *RateLimiter {
	r := &RateLimiter{
		lastSeen: make(map[string]time.Time),
		done:     make(chan struct{}),
	}
	go r.cleanup()
	return r
}

// Stop останавливает фоновую горутину очистки.
func (r *RateLimiter) Stop() {
	close(r.done)
}

// Allow returns true if the request should be allowed.
// Returns true unconditionally when minPeriodMs == 0 or identifier == "".
func (r *RateLimiter) Allow(identifier string, minPeriodMs int) bool {
	if minPeriodMs == 0 || identifier == "" {
		return true
	}

	minPeriod := time.Duration(minPeriodMs) * time.Millisecond

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if last, ok := r.lastSeen[identifier]; ok && now.Sub(last) < minPeriod {
		return false
	}
	r.lastSeen[identifier] = now
	return true
}

// cleanup periodically removes stale entries to prevent unbounded memory growth.
func (r *RateLimiter) cleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-entryTTL)
			r.mu.Lock()
			for id, t := range r.lastSeen {
				if t.Before(cutoff) {
					delete(r.lastSeen, id)
				}
			}
			r.mu.Unlock()
		case <-r.done:
			return
		}
	}
}
