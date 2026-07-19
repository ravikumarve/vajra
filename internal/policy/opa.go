// Package policy implements OPA/Rego-based access control for VAJRA.
//
// Policies are evaluated at request time for every tool call.
// They define what agents can access, under what conditions,
// and with what scope.
package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/rego"
)

// Decision represents the result of a policy evaluation.
type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	TTL     string `json:"ttl,omitempty"`
}

// Engine is the OPA/Rego policy evaluation engine.
type Engine struct {
	mu          sync.RWMutex
	policyDir   string
	regoModules map[string]string
	compiled    bool
	lastLoad    time.Time
}

// NewEngine creates a policy engine that loads Rego files from the given directory.
func NewEngine(policyDir string) *Engine {
	return &Engine{
		policyDir:   policyDir,
		regoModules: make(map[string]string),
	}
}

// Load reads all .rego files from the configured directory.
func (e *Engine) Load() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.regoModules = make(map[string]string)

	// Expand ~ in path
	dir := e.policyDir
	if len(dir) > 0 && dir[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			dir = filepath.Join(home, dir[1:])
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[vajra] policy: no policy dir at %s, using defaults", dir)
			return e.loadDefaults()
		}
		return fmt.Errorf("reading policy dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".rego" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("reading policy %s: %w", entry.Name(), err)
		}
		e.regoModules[entry.Name()] = string(data)
	}

	if len(e.regoModules) == 0 {
		return e.loadDefaults()
	}

	// Verify policies compile
	if err := e.compile(); err != nil {
		return fmt.Errorf("policy compilation error: %w", err)
	}

	e.lastLoad = time.Now()
	log.Printf("[vajra] policy: loaded %d policy files from %s", len(e.regoModules), dir)
	return nil
}

// Evaluate evaluates a request against loaded policies.
func (e *Engine) Evaluate(agentID, action, resource string, contextData map[string]interface{}) (*Decision, error) {
	e.mu.RLock()
	modules := make(map[string]string, len(e.regoModules))
	for k, v := range e.regoModules {
		modules[k] = v
	}
	e.mu.RUnlock()

	if len(modules) == 0 {
		return &Decision{Allowed: true, Reason: "default allow (no policies)", TTL: "5s"}, nil
	}

	// Build input
	input := map[string]interface{}{
		"agent_id":  agentID,
		"action":    action,
		"resource":  resource,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	for k, v := range contextData {
		input[k] = v
	}

	// Prepare Rego query with all modules + input
	opts := []func(r *rego.Rego){
		rego.Query("data.vajra.allow = true"),
		rego.Input(input),
	}

	// Add all modules
	for filename, module := range modules {
		opts = append(opts, rego.Module(filename, module))
	}

	// Create and evaluate the Rego query
	r := rego.New(opts...)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	rs, err := r.Eval(ctx)
	if err != nil {
		return nil, fmt.Errorf("OPA evaluation error: %w", err)
	}

	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return &Decision{Allowed: false, Reason: "denied by policy (no matching rules)"}, nil
	}

	allowed, ok := rs[0].Expressions[0].Value.(bool)
	if !ok || !allowed {
		return &Decision{Allowed: false, Reason: "denied by policy"}, nil
	}

	// Try to get suggested TTL from policy
	ttl := e.evaluateTTL(modules, input)

	return &Decision{
		Allowed: true,
		Reason:  "allowed by policy",
		TTL:     ttl,
	}, nil
}

// evaluateTTL tries to extract a suggested TTL from the policy.
func (e *Engine) evaluateTTL(modules map[string]string, input map[string]interface{}) string {
	// Build a query for the TTL
	ttlOpts := []func(r *rego.Rego){
		rego.Query("data.vajra.suggested_ttl = x"),
		rego.Input(input),
	}
	for filename, module := range modules {
		ttlOpts = append(ttlOpts, rego.Module(filename, module))
	}

	r := rego.New(ttlOpts...)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	rs, err := r.Eval(ctx)
	if err != nil || len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return "5s"
	}

	ttl, ok := rs[0].Expressions[0].Value.(string)
	if !ok || ttl == "" {
		return "5s"
	}

	return ttl
}

// List returns all loaded policy names.
func (e *Engine) List() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.regoModules))
	for name := range e.regoModules {
		names = append(names, name)
	}
	return names
}

// Health returns diagnostics about the engine.
func (e *Engine) Health() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"policies_loaded": len(e.regoModules),
		"compiled":        e.compiled,
		"last_load":       e.lastLoad.Format(time.RFC3339),
	}
}

// compile verifies that all loaded policies are valid Rego.
func (e *Engine) compile() error {
	// Build a full query that includes all modules to verify they compile
	opts := []func(r *rego.Rego){
		rego.Query("1 = 1"), // Dummy query to trigger compilation
		rego.Input(map[string]interface{}{"test": true}),
	}
	for filename, module := range e.regoModules {
		opts = append(opts, rego.Module(filename, module))
	}

	r := rego.New(opts...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := r.Eval(ctx)
	if err != nil {
		e.compiled = false
		return fmt.Errorf("rego compilation failed: %w", err)
	}

	e.compiled = true
	return nil
}

func (e *Engine) loadDefaults() error {
	e.regoModules["default_allow.rego"] = `package vajra

default allow = false

allow = true {
    input.action == "db_query"
    input.resource != ""
}

allow = true {
    input.action == "api_call"
    input.resource != ""
}

allow = true {
    input.action == "vajra/mint"
    input.agent_id != ""
}

suggest_ttl = "5s" {
    true
}
`
	return e.compile()
}

// Ensure encoding/json is used
var _ = json.Marshal
