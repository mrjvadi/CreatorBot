package worker

import (
	"sync"
	"time"
)

// rateLimiter is a simple token-bucket limiter, one per BotWorker.
// Non-blocking: Allow() returns false immediately if no token is available
// so the worker can skip the job and let another bot handle it.
type rateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	refillPS float64 // tokens per second
	lastTick time.Time
}

func newRateLimiter(perSecond int) *rateLimiter {
	cap := float64(perSecond)
	return &rateLimiter{
		tokens:   cap,
		capacity: cap,
		refillPS: cap,
		lastTick: time.Now(),
	}
}

// Allow consumes one token and returns true, or returns false if empty.
func (r *rateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.tokens += now.Sub(r.lastTick).Seconds() * r.refillPS
	r.lastTick = now
	if r.tokens > r.capacity {
		r.tokens = r.capacity
	}
	if r.tokens < 1 {
		return false
	}
	r.tokens--
	return true
}
