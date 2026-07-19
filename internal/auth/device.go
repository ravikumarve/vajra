// Package auth implements agent identity and authentication for VAJRA.
//
// Supports OAuth 2.1 Device Flow (for human-linked agents) and
// mTLS certificate-based identity (for headless agents).
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// DeviceFlowState represents an active OAuth 2.1 Device Flow session.
type DeviceFlowState struct {
	DeviceCode      string    `json:"device_code"`
	UserCode        string    `json:"user_code"`
	VerificationURI string    `json:"verification_uri"`
	Interval        int       `json:"interval"`
	ExpiresAt       time.Time `json:"expires_at"`
	AgentID         string    `json:"agent_id,omitempty"`
	Status          string    `json:"status"` // "pending", "approved", "expired"
	CreatedAt       time.Time `json:"created_at"`
}

// DeviceFlowManager handles OAuth 2.1 Device Flow sessions.
type DeviceFlowManager struct {
	mu       sync.RWMutex
	sessions map[string]*DeviceFlowState
	baseURL  string
}

// NewDeviceFlowManager creates a new device flow manager.
func NewDeviceFlowManager(baseURL string) *DeviceFlowManager {
	return &DeviceFlowManager{
		sessions: make(map[string]*DeviceFlowState),
		baseURL:  baseURL,
	}
}

// StartFlow initiates a new device authorization flow.
func (m *DeviceFlowManager) StartFlow() *DeviceFlowState {
	m.mu.Lock()
	defer m.mu.Unlock()

	deviceCode := generateCode(32)
	userCode := fmt.Sprintf("%s-%s", generateCode(4), generateCode(4))

	state := &DeviceFlowState{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURI: fmt.Sprintf("%s/activate", m.baseURL),
		Interval:        5,
		ExpiresAt:       time.Now().Add(15 * time.Minute),
		Status:          "pending",
		CreatedAt:       time.Now(),
	}

	m.sessions[deviceCode] = state

	// Auto-expire old sessions
	go func() {
		time.Sleep(16 * time.Minute)
		m.mu.Lock()
		delete(m.sessions, deviceCode)
		m.mu.Unlock()
	}()

	return state
}

// CheckFlow checks the status of a device flow session.
func (m *DeviceFlowManager) CheckFlow(deviceCode string) (*DeviceFlowState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.sessions[deviceCode]
	if !ok {
		return nil, fmt.Errorf("invalid device code")
	}

	if time.Now().After(state.ExpiresAt) {
		state.Status = "expired"
		return nil, fmt.Errorf("device code expired")
	}

	return state, nil
}

// ApproveFlow approves a pending device flow session for an agent.
func (m *DeviceFlowManager) ApproveFlow(userCode, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, state := range m.sessions {
		if state.UserCode == userCode && state.Status == "pending" {
			state.Status = "approved"
			state.AgentID = agentID
			return nil
		}
	}

	return fmt.Errorf("invalid or expired user code")
}

// RemoveFlow removes a completed or expired flow.
func (m *DeviceFlowManager) RemoveFlow(deviceCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, deviceCode)
}

func generateCode(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("failed to generate random code: %v", err))
	}
	return hex.EncodeToString(bytes)[:length]
}
