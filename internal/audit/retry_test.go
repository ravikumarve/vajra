package audit

import (
	"errors"
	"testing"
	"time"
)

func TestWithRetrySucceedsFirstTry(t *testing.T) {
	calls := 0
	err := withRetry(RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}, func() error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("expected 1 call nil error, got %d calls err=%v", calls, err)
	}
}

func TestWithRetryTransientThenSuccess(t *testing.T) {
	calls := 0
	err := withRetry(RetryPolicy{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}, func() error {
		calls++
		if calls < 3 {
			return errors.New("database is locked")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("expected 3 calls nil error, got %d calls err=%v", calls, err)
	}
}

func TestWithRetryNonTransientImmediate(t *testing.T) {
	calls := 0
	sentinel := errors.New("CHECK constraint failed")
	err := withRetry(RetryPolicy{MaxAttempts: 5, BaseDelay: time.Millisecond, MaxDelay: 5 * time.Millisecond}, func() error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) || calls != 1 {
		t.Fatalf("expected immediate sentinel, got %d calls err=%v", calls, err)
	}
}

func TestWithRetryExhausts(t *testing.T) {
	calls := 0
	err := withRetry(RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 2 * time.Millisecond}, func() error {
		calls++
		return errors.New("database table is locked")
	})
	if err == nil || calls != 3 {
		t.Fatalf("expected 3 calls + last error, got %d calls err=%v", calls, err)
	}
}
