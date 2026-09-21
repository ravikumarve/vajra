package credential

import (
	"testing"
	"time"
)

func mkCred(id string, ttl time.Duration) *Credential {
	now := time.Now()
	return &Credential{ID: id, AgentID: "a", Target: "t", TTL: ttl, CreatedAt: now, ExpiresAt: now.Add(ttl)}
}

func TestRingInsertGetRevoke(t *testing.T) {
	rb := NewRingBuffer(8, time.Minute)
	defer rb.Stop()

	rb.Insert(mkCred("c1", time.Minute))
	if got := rb.Get("c1"); got == nil || got.ID != "c1" {
		t.Fatalf("expected c1, got %+v", got)
	}
	if !rb.Revoke("c1") {
		t.Fatal("expected revoke true")
	}
	if got := rb.Get("c1"); got != nil {
		t.Fatalf("expected nil after revoke, got %+v", got)
	}
	if rb.Revoke("missing") {
		t.Fatal("expected revoke false for missing id")
	}
}

func TestRingExpiredInvisible(t *testing.T) {
	rb := NewRingBuffer(8, time.Minute)
	defer rb.Stop()

	rb.Insert(mkCred("old", -time.Second))
	if got := rb.Get("old"); got != nil {
		t.Fatalf("expected expired cred hidden, got %+v", got)
	}
}
