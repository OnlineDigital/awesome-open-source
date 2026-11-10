package application

import (
	"sync"
	"time"
)

// rateLimiter is a tiny fixed-window limiter keyed by client address (login throttling).
type rateLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	entries map[string]*window
}

type window struct {
	count int
	start time.Time
}

func newRateLimiter(max int, seconds int) *rateLimiter {
	return &rateLimiter{max: max, window: time.Duration(seconds) * time.Second, entries: map[string]*window{}}
}

// Allow records an attempt and reports whether it is within the limit.
func (r *rateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	w := r.entries[key]
	if w == nil || now.Sub(w.start) > r.window {
		if len(r.entries) > 10000 {
			r.entries = map[string]*window{}
		}
		w = &window{start: now}
		r.entries[key] = w
	}
	w.count++
	return w.count <= r.max
}

func (r *rateLimiter) Reset(key string) {
	r.mu.Lock()
	delete(r.entries, key)
	r.mu.Unlock()
}
