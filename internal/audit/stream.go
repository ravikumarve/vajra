package audit

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// StreamWriter writes audit entries to stdout as newline-delimited JSON.
// This is designed for log aggregators (Vector, Loki, Datadog, etc.)
type StreamWriter struct {
	writer *json.Encoder
	mu     sync.Mutex
}

// NewStreamWriter creates a JSON stream writer that writes to stdout.
func NewStreamWriter() *StreamWriter {
	return &StreamWriter{
		writer: json.NewEncoder(os.Stdout),
	}
}

// NewStreamWriterTo creates a JSON stream writer targeting a specific file.
func NewStreamWriterTo(path string) (*StreamWriter, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening stream file: %w", err)
	}
	return &StreamWriter{
		writer: json.NewEncoder(f),
	}, nil
}

// Write outputs a structured log entry as JSON.
func (s *StreamWriter) Write(entry map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry["@timestamp"] = time.Now().UTC().Format(time.RFC3339)
	entry["@version"] = 1
	entry["@service"] = "vajra"

	if err := s.writer.Encode(entry); err != nil {
		log.Printf("[vajra] audit stream: write error: %v", err)
	}
}

// LogMint writes a credential mint event to the stream.
func (s *StreamWriter) LogMint(agentID, credentialID, target, scope, ttl string) {
	s.Write(map[string]interface{}{
		"event":         "credential.mint",
		"agent_id":      agentID,
		"credential_id": credentialID,
		"target":        target,
		"scope":         scope,
		"ttl":           ttl,
	})
}

// LogRevoke writes a credential revocation event to the stream.
func (s *StreamWriter) LogRevoke(agentID, credentialID, reason string) {
	s.Write(map[string]interface{}{
		"event":         "credential.revoke",
		"agent_id":      agentID,
		"credential_id": credentialID,
		"reason":        reason,
	})
}

// LogDeny writes a policy denial event to the stream.
func (s *StreamWriter) LogDeny(agentID, action, resource, reason string) {
	s.Write(map[string]interface{}{
		"event":    "policy.deny",
		"agent_id": agentID,
		"action":   action,
		"resource": resource,
		"reason":   reason,
	})
}

// Ensure fmt and time are used
var _, _ = fmt.Sprintf, time.Now
