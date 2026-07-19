package credential

import (
	"log"
	"sync"
)

// Revoker handles credential revocation, both immediate and deferred.
type Revoker struct {
	ring      *RingBuffer
	providers map[string]CredentialProvider
	mu        sync.RWMutex
}

// NewRevoker creates a credential revoker backed by a ring buffer.
func NewRevoker(ring *RingBuffer, providers map[string]CredentialProvider) *Revoker {
	return &Revoker{
		ring:      ring,
		providers: providers,
	}
}

// Revoke immediately invalidates a credential. Returns true if found and revoked.
func (r *Revoker) Revoke(credID string) bool {
	// First, try to deprovision via the target provider
	if cred := r.ring.Get(credID); cred != nil {
		targetType := detectTargetType(cred.Target)
		if provider, ok := r.providers[targetType]; ok {
			if err := provider.Deprovision(credID, cred.Target); err != nil {
				log.Printf("[vajra] revoke: provider deprovision failed: %v", err)
			}
		}
	}

	// Remove from ring buffer
	removed := r.ring.Revoke(credID)
	if removed {
		log.Printf("[vajra] revoke: credential %s revoked", credID)
	} else {
		log.Printf("[vajra] revoke: credential %s not found (may have expired)", credID)
	}

	return removed
}

// RevokeByAgent revokes all active credentials for a given agent.
func (r *Revoker) RevokeByAgent(agentID string) int {
	count := 0
	for _, cred := range r.ring.All() {
		if cred.AgentID == agentID {
			if r.Revoke(cred.ID) {
				count++
			}
		}
	}
	return count
}

// PurgeExpired removes all expired credentials from the ring buffer.
func (r *Revoker) PurgeExpired() int {
	count := 0
	for _, cred := range r.ring.All() {
		if cred.IsExpired() {
			r.ring.Revoke(cred.ID)
			count++
		}
	}
	return count
}
