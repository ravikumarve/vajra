package auth

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// AgentInfo represents a registered agent in the VAJRA agent registry.
type AgentInfo struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	SecretHash   string            `json:"-"` // bcrypt hash, never serialized
	Policies     []string          `json:"policies"`
	Tags         map[string]string `json:"tags,omitempty"`
	IdentityType string            `json:"identity_type"` // "oauth" or "mtls"
	OwnerID      string            `json:"owner_id,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	LastSeen     time.Time         `json:"last_seen,omitempty"`
	Active       bool              `json:"active"`
}

// Registry manages agent identities and their associated policies.
type Registry struct {
	mu     sync.RWMutex
	agents map[string]*AgentInfo
}

// NewRegistry creates a new agent registry.
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]*AgentInfo),
	}
}

// Register adds a new agent to the registry.
func (r *Registry) Register(id, name, secretHash, identityType string, policies []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[id]; exists {
		return fmt.Errorf("agent %s already registered", id)
	}

	r.agents[id] = &AgentInfo{
		ID:           id,
		Name:         name,
		SecretHash:   secretHash,
		Policies:     policies,
		Tags:         make(map[string]string),
		IdentityType: identityType,
		CreatedAt:    time.Now(),
		Active:       true,
	}

	log.Printf("[vajra] registry: registered agent %q (%s)", id, name)
	return nil
}

// Get retrieves an agent by ID.
func (r *Registry) Get(id string) (*AgentInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent %s not found", id)
	}

	return agent, nil
}

// List returns all registered agents.
func (r *Registry) List() []*AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*AgentInfo, 0, len(r.agents))
	for _, agent := range r.agents {
		result = append(result, agent)
	}
	return result
}

// UpdateLastSeen records that an agent was seen.
func (r *Registry) UpdateLastSeen(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if agent, ok := r.agents[id]; ok {
		agent.LastSeen = time.Now()
	}
}

// Deactivate marks an agent as inactive.
func (r *Registry) Deactivate(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[id]
	if !ok {
		return fmt.Errorf("agent %s not found", id)
	}

	agent.Active = false
	log.Printf("[vajra] registry: deactivated agent %s", id)
	return nil
}

// Count returns the number of registered agents.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}
