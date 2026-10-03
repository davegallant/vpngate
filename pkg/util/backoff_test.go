package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoffStaysWithinBounds(t *testing.T) {
	b := NewBackoff(time.Second, 4*time.Second)
	for i := 0; i < 50; i++ {
		d := b.Next()
		assert.GreaterOrEqual(t, d, time.Duration(0))
		assert.LessOrEqual(t, d, 4*time.Second)
	}
}

func TestBackoffReset(t *testing.T) {
	b := NewBackoff(time.Second, time.Minute)
	for i := 0; i < 10; i++ {
		b.Next()
	}
	b.Reset()
	// After a reset the next delay is drawn from the initial step again.
	assert.LessOrEqual(t, b.Next(), time.Second)
}
