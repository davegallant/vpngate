package util

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetrySucceedsImmediately(t *testing.T) {
	calls := 0
	err := Retry(3, time.Millisecond, func() error {
		calls++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetrySucceedsAfterFailures(t *testing.T) {
	calls := 0
	err := Retry(5, time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryExhaustsAttempts(t *testing.T) {
	calls := 0
	wantErr := errors.New("permanent")
	err := Retry(4, time.Millisecond, func() error {
		calls++
		return wantErr
	})
	assert.Equal(t, wantErr, err)
	assert.Equal(t, 4, calls)
}

// TestRetryWaitsBetweenAttempts guards against the delay parameter being
// passed as an untyped int that silently truncates to nanoseconds.
func TestRetryWaitsBetweenAttempts(t *testing.T) {
	const attempts = 3
	delay := 20 * time.Millisecond

	start := time.Now()
	_ = Retry(attempts, delay, func() error {
		return errors.New("fail")
	})
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed, delay*time.Duration(attempts))
}
