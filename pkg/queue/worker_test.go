package queue

import (
	"testing"
	"time"
)

func TestRateLimiterNoBurstBeyondCapacity(t *testing.T) {
	// 60/min -> 1 token/sec, capacity 1.
	l := newRateLimiter(60)
	if !l.allow() {
		t.Fatal("expected first token to be available immediately")
	}
	if l.allow() {
		t.Fatal("expected second token to be unavailable until refill")
	}
	time.Sleep(1100 * time.Millisecond)
	if !l.allow() {
		t.Fatal("expected a token after the refill interval")
	}
}

func TestRateLimiterAllowsBatchSizedBurst(t *testing.T) {
	// 600/min -> 10 tokens/sec, capacity 10: a full batch can fire at once,
	// but the sustained rate stays capped.
	l := newRateLimiter(600)
	granted := 0
	for i := 0; i < 100; i++ {
		if !l.allow() {
			break
		}
		granted++
	}
	if granted != 10 {
		t.Errorf("expected a burst of exactly capacity (10), got %d", granted)
	}
}

func TestRateLimiterSustainedRate(t *testing.T) {
	perMinute := 120 // 2/sec
	l := newRateLimiter(perMinute)

	start := time.Now()
	allowed := 0
	for time.Since(start) < 1600*time.Millisecond {
		if l.allow() {
			allowed++
		} else {
			time.Sleep(20 * time.Millisecond)
		}
	}

	// Between capacity (2) and elapsed*rate+slack (~3.5).
	if allowed < 2 || allowed > 5 {
		t.Errorf("expected ~3 tokens in 1.6s, got %d", allowed)
	}
}
