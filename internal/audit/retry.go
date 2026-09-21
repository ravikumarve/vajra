// Retry with backoff for transient SQLite contention.
//
// The audit logger is written concurrently by the proxy fast path and read
// by the dashboard/TUI. Under WAL + concurrent writers SQLite can return
// SQLITE_BUSY ("database is locked"). Losing audit rows is not an option,
// so writes retry with exponential backoff + jitter instead of failing fast.
package audit

import (
	"math/rand"
	"strings"
	"time"
)

// RetryPolicy bounds transient-error retries.
type RetryPolicy struct {
	MaxAttempts int           // total tries including the first (default 5)
	BaseDelay   time.Duration // first backoff step (default 10ms)
	MaxDelay    time.Duration // backoff cap (default 500ms)
}

func defaultRetryPolicy() RetryPolicy {
	return RetryPolicy{MaxAttempts: 5, BaseDelay: 10 * time.Millisecond, MaxDelay: 500 * time.Millisecond}
}

// transient reports whether err looks like SQLite contention worth retrying.
func transient(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "busy") || strings.Contains(msg, "locked")
}

// withRetry runs fn until it succeeds, fails non-transiently, or exhausts
// the policy. The last error is returned when attempts run out.
func withRetry(p RetryPolicy, fn func() error) error {
	if p.MaxAttempts < 1 {
		p.MaxAttempts = 1
	}
	if p.BaseDelay <= 0 {
		p.BaseDelay = 10 * time.Millisecond
	}
	if p.MaxDelay <= 0 {
		p.MaxDelay = 500 * time.Millisecond
	}
	var err error
	delay := p.BaseDelay
	for attempt := 1; ; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if !transient(err) || attempt >= p.MaxAttempts {
			return err
		}
		jitter := time.Duration(rand.Int63n(int64(delay)))
		time.Sleep(delay/2 + jitter)
		delay *= 2
		if delay > p.MaxDelay {
			delay = p.MaxDelay
		}
	}
}
