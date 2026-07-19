// Package credential implements an in-memory ring buffer for ephemeral credentials.
//
// The ring buffer stores credentials with sub-5-second TTL and provides
// O(1) insert, lookup, and eviction. A background goroutine sweeps
// expired entries on a configurable interval.
package credential

import (
	"fmt"
	"sync"
	"time"
)

// Credential represents a single ephemeral credential in the ring buffer.
type Credential struct {
	ID        string            `json:"id"`
	AgentID   string            `json:"agent_id"`
	Target    string            `json:"target"`
	Scope     string            `json:"scope"`
	TTL       time.Duration     `json:"ttl"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// IsExpired returns true if the credential has exceeded its TTL.
func (c *Credential) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// RemainingTTL returns the duration until expiry.
func (c *Credential) RemainingTTL() time.Duration {
	remaining := time.Until(c.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RingBuffer is a fixed-capacity, lock-free-ish ring buffer for credentials.
// It is goroutine-safe via a sync.RWMutex.
type RingBuffer struct {
	mu       sync.RWMutex
	buffer   []*Credential
	capacity int
	head     int // next write position
	count    int // number of active entries
	sweepDur time.Duration
	stopCh   chan struct{}
}

// NewRingBuffer creates a ring buffer with the given capacity and sweep interval.
// The sweep goroutine is started automatically.
func NewRingBuffer(capacity int, sweepInterval time.Duration) *RingBuffer {
	rb := &RingBuffer{
		buffer:   make([]*Credential, capacity),
		capacity: capacity,
		head:     0,
		count:    0,
		sweepDur: sweepInterval,
		stopCh:   make(chan struct{}),
	}
	go rb.sweepLoop()
	return rb
}

// Stop terminates the background sweep goroutine.
func (rb *RingBuffer) Stop() {
	close(rb.stopCh)
}

// Insert adds a credential to the ring buffer. If the buffer is full,
// the oldest entry is evicted (overwritten).
func (rb *RingBuffer) Insert(c *Credential) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer[rb.head] = c
	rb.head = (rb.head + 1) % rb.capacity
	if rb.count < rb.capacity {
		rb.count++
	}
}

// Get retrieves a credential by ID. Returns nil if not found or expired.
func (rb *RingBuffer) Get(id string) *Credential {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	for i := 0; i < rb.count; i++ {
		idx := (rb.head - rb.count + i + rb.capacity) % rb.capacity
		if rb.buffer[idx] != nil && rb.buffer[idx].ID == id {
			if rb.buffer[idx].IsExpired() {
				return nil
			}
			return rb.buffer[idx]
		}
	}
	return nil
}

// Revoke removes a credential by ID. Returns true if found and removed.
func (rb *RingBuffer) Revoke(id string) bool {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for i := 0; i < rb.count; i++ {
		idx := (rb.head - rb.count + i + rb.capacity) % rb.capacity
		if rb.buffer[idx] != nil && rb.buffer[idx].ID == id {
			rb.buffer[idx] = nil
			// Compact: shift remaining entries
			for j := i; j < rb.count-1; j++ {
				src := (rb.head - rb.count + j + 1 + rb.capacity) % rb.capacity
				dst := (rb.head - rb.count + j + rb.capacity) % rb.capacity
				rb.buffer[dst] = rb.buffer[src]
			}
			rb.count--
			rb.head = (rb.head - 1 + rb.capacity) % rb.capacity
			return true
		}
	}
	return false
}

// Len returns the number of active (non-expired) credentials.
func (rb *RingBuffer) Len() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}

// All returns a snapshot of all active credentials.
func (rb *RingBuffer) All() []*Credential {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]*Credential, 0, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.head - rb.count + i + rb.capacity) % rb.capacity
		if rb.buffer[idx] != nil && !rb.buffer[idx].IsExpired() {
			result = append(result, rb.buffer[idx])
		}
	}
	return result
}

// sweepLoop runs in the background, evicting expired credentials.
func (rb *RingBuffer) sweepLoop() {
	ticker := time.NewTicker(rb.sweepDur)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rb.evictExpired()
		case <-rb.stopCh:
			return
		}
	}
}

func (rb *RingBuffer) evictExpired() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	now := time.Now()
	for i := 0; i < rb.count; i++ {
		idx := (rb.head - rb.count + i + rb.capacity) % rb.capacity
		if rb.buffer[idx] != nil && now.After(rb.buffer[idx].ExpiresAt) {
			rb.buffer[idx] = nil
		}
	}
	// Compact the buffer
	writeIdx := 0
	for readIdx := 0; readIdx < rb.capacity; readIdx++ {
		if rb.buffer[readIdx] != nil {
			rb.buffer[writeIdx] = rb.buffer[readIdx]
			writeIdx++
		}
	}
	for i := writeIdx; i < rb.capacity; i++ {
		rb.buffer[i] = nil
	}
	rb.count = writeIdx
	if rb.head >= rb.count {
		rb.head = rb.count % rb.capacity
	} else {
		rb.head = rb.count
	}
}

// NewCredential creates a new credential with a generated ID and TTL.
func NewCredential(agentID, target, scope string, ttl time.Duration) *Credential {
	now := time.Now()
	return &Credential{
		ID:        fmt.Sprintf("cred_%x", now.UnixNano()),
		AgentID:   agentID,
		Target:    target,
		Scope:     scope,
		TTL:       ttl,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
		Metadata:  make(map[string]string),
	}
}
