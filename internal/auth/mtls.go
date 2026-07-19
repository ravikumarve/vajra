package auth

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
)

// mTLSHandler manages client certificate authentication for headless agents.
type mTLSHandler struct {
	mu          sync.RWMutex
	trustedCAs  *x509.CertPool
	agentCerts  map[string]string // cert fingerprint -> agent ID
}

// NewMTLSHandler creates a new mTLS handler.
func NewMTLSHandler(caPath string) (*mTLSHandler, error) {
	h := &mTLSHandler{
		agentCerts: make(map[string]string),
	}

	// If a CA path is provided, load it
	if caPath != "" {
		pool := x509.NewCertPool()
		// In production, read CA cert from file
		// caCert, err := os.ReadFile(caPath)
		// if err != nil { return nil, err }
		// pool.AppendCertsFromPEM(caCert)
		h.trustedCAs = pool
	}

	return h, nil
}

// RegisterAgent registers an agent's certificate fingerprint.
func (h *mTLSHandler) RegisterAgent(agentID, certPEM string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	fingerprint := fingerprintCert(certPEM)
	h.agentCerts[fingerprint] = agentID
	log.Printf("[vajra] mTLS: registered agent %s (fingerprint: %s)", agentID, fingerprint[:16])
	return nil
}

// Authenticate verifies a client certificate and returns the agent ID.
func (h *mTLSHandler) Authenticate(certPEM string) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	fingerprint := fingerprintCert(certPEM)
	agentID, ok := h.agentCerts[fingerprint]
	if !ok {
		return "", fmt.Errorf("unknown certificate: %s", fingerprint[:16])
	}

	return agentID, nil
}

// RevokeAgent removes an agent's certificate registration.
func (h *mTLSHandler) RevokeAgent(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for fp, id := range h.agentCerts {
		if id == agentID {
			delete(h.agentCerts, fp)
			log.Printf("[vajra] mTLS: revoked agent %s", agentID)
			return
		}
	}
}

func fingerprintCert(certPEM string) string {
	hash := sha256.Sum256([]byte(certPEM))
	return hex.EncodeToString(hash[:])
}
