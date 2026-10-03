package util

import (
	"math/rand/v2"
	"time"
)

// Backoff produces exponentially increasing delays with full jitter,
// for retry loops that must not hammer a failing server. The zero value
// is not usable; construct with NewBackoff.
type Backoff struct {
	initial time.Duration
	max     time.Duration
	current time.Duration
}

// NewBackoff returns a Backoff that starts at initial and doubles on
// every Next call, capped at max.
func NewBackoff(initial, max time.Duration) *Backoff {
	return &Backoff{initial: initial, max: max}
}

// Next returns the next delay, with full jitter applied (a uniform
// random value between 0 and the current step), then advances the
// sequence.
func (b *Backoff) Next() time.Duration {
	if b.current == 0 {
		b.current = b.initial
	}
	delay := b.current
	if delay > 0 {
		delay = time.Duration(rand.Int64N(int64(delay) + 1))
	}
	b.current *= 2
	if b.current > b.max || b.current <= 0 {
		b.current = b.max
	}
	return delay
}

// Reset restarts the sequence from the initial delay. Call it when a
// retry loop finally succeeds, so the next failure starts fast again
// instead of inheriting a long-grown delay.
func (b *Backoff) Reset() {
	b.current = 0
}
