package credential

import (
	"fmt"
	"log"
	"time"
)

// MintRequest describes what credential to create.
type MintRequest struct {
	AgentID  string            `json:"agent_id"`
	Target   string            `json:"target"`   // e.g. "postgres://host/db", "api.example.com"
	Scope    string            `json:"scope"`    // e.g. "SELECT users(email,name)", "GET /v2/users"
	TTL      time.Duration     `json:"ttl"`      // time-to-live
	Metadata map[string]string `json:"metadata,omitempty"`
}

// MintResult describes the minted credential and its properties.
type MintResult struct {
	Credential *Credential `json:"credential"`
	TargetDSN  string      `json:"target_dsn"` // The actual scoped connection string/token
	ExpiresIn  string      `json:"expires_in"`
}

// Minter handles the provisioning of ephemeral credentials.
type Minter struct {
	ring      *RingBuffer
	providers map[string]CredentialProvider
}

// CredentialProvider defines how to provision a credential for a target type.
type CredentialProvider interface {
	// Provision creates a temporary credential on the target system.
	// Returns the actual scoped credential string (DSN, token, etc.)
	Provision(agentID, target, scope string, ttl time.Duration) (string, error)

	// Deprovision removes a temporary credential from the target system.
	Deprovision(credID, target string) error
}

// NewMinter creates a credential minter backed by a ring buffer.
func NewMinter(ring *RingBuffer) *Minter {
	return &Minter{
		ring:      ring,
		providers: make(map[string]CredentialProvider),
	}
}

// RegisterProvider registers a credential provider for a target type.
func (m *Minter) RegisterProvider(targetType string, provider CredentialProvider) {
	m.providers[targetType] = provider
}

// Mint creates an ephemeral credential and stores it in the ring buffer.
func (m *Minter) Mint(req *MintRequest) (*MintResult, error) {
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if req.Target == "" {
		return nil, fmt.Errorf("target is required")
	}
	if req.TTL <= 0 {
		req.TTL = 5 * time.Second
	}
	if req.TTL > 60*time.Second {
		return nil, fmt.Errorf("TTL cannot exceed 60 seconds")
	}

	cred := NewCredential(req.AgentID, req.Target, req.Scope, req.TTL)
	if req.Metadata != nil {
		cred.Metadata = req.Metadata
	}

	// Try to provision via a registered provider
	var targetDSN string
	targetType := detectTargetType(req.Target)
	if provider, ok := m.providers[targetType]; ok {
		dsn, err := provider.Provision(req.AgentID, req.Target, req.Scope, req.TTL)
		if err != nil {
			log.Printf("[vajra] mint: provider %s failed: %v", targetType, err)
			// Fall through — still mint the credential for the ring buffer
		} else {
			targetDSN = dsn
		}
	}

	// Store in ring buffer
	m.ring.Insert(cred)

	// If no provider provisioned, use a placeholder DSN with the credential ID
	if targetDSN == "" {
		targetDSN = fmt.Sprintf("vajra://%s/%s?cred_id=%s", req.Target, req.Scope, cred.ID)
	}

	log.Printf("[vajra] mint: credential %s for agent %s (TTL: %v)",
		cred.ID, req.AgentID, req.TTL)

	return &MintResult{
		Credential: cred,
		TargetDSN:  targetDSN,
		ExpiresIn:  req.TTL.String(),
	}, nil
}

// Resolve retrieves the credential and target DSN for a credential ID.
func (m *Minter) Resolve(credID string) (*MintResult, error) {
	cred := m.ring.Get(credID)
	if cred == nil {
		return nil, fmt.Errorf("credential %s not found or expired", credID)
	}

	return &MintResult{
		Credential: cred,
		TargetDSN:  fmt.Sprintf("vajra://%s/%s?cred_id=%s", cred.Target, cred.Scope, cred.ID),
		ExpiresIn:  cred.RemainingTTL().String(),
	}, nil
}

func detectTargetType(target string) string {
	// Simple heuristic — can be extended with a proper registry
	if len(target) > 6 && target[:7] == "postgres" {
		return "postgres"
	}
	if len(target) > 4 && target[:5] == "mysql" {
		return "mysql"
	}
	if len(target) > 3 && target[:4] == "http" {
		return "http"
	}
	return "generic"
}
