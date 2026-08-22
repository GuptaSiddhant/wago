package queue

import (
	"context"
	"math"
	"sync"
	"time"
)

// Config tunes how aggressively the broadcast worker sends messages.
type Config struct {
	MessagesPerMinute     int
	BatchSize             int
	LeaseSeconds          int
	MaxAttempts           int
	BackoffBase           time.Duration
	LeaseRecoveryInterval time.Duration // how often expired leases are swept
}

// DefaultConfig returns a sane single-instance configuration.
func DefaultConfig() Config {
	return Config{
		MessagesPerMinute:     60,
		BatchSize:             10,
		LeaseSeconds:          300,
		MaxAttempts:           3,
		BackoffBase:           30 * time.Second,
		LeaseRecoveryInterval: time.Minute,
	}
}

// rateLimiter is a token bucket that paces sends to `perMinute` while allowing
// bursts of up to one second's worth of tokens (that's the "batching": we grab
// a batch fast, but the sustained rate stays capped).
type rateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	capacity   float64
	ratePerSec float64
	last       time.Time
}

func newRateLimiter(perMinute int) *rateLimiter {
	rate := float64(perMinute) / 60.0
	return &rateLimiter{tokens: rate, capacity: rate, ratePerSec: rate, last: time.Now()}
}

// Wait blocks until a token is available or the context is cancelled.
func (l *rateLimiter) Wait(ctx context.Context) {
	for {
		if l.allow() {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (l *rateLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.last).Seconds()
	l.last = now
	l.tokens = math.Min(l.capacity, l.tokens+elapsed*l.ratePerSec)
	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}