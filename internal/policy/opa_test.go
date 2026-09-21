package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsMintAllowed(t *testing.T) {
	e := NewEngine("")
	if err := e.Load(); err != nil {
		t.Fatalf("Load defaults: %v", err)
	}
	d, err := e.Evaluate("agent-1", "vajra/mint", "postgres://db", nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !d.Allowed {
		t.Fatalf("expected mint allowed, got %+v", d)
	}
}

func TestDefaultsEmptyActionDenied(t *testing.T) {
	e := NewEngine("")
	if err := e.Load(); err != nil {
		t.Fatalf("Load defaults: %v", err)
	}
	d, err := e.Evaluate("", "", "", nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if d.Allowed {
		t.Fatalf("expected empty request denied, got %+v", d)
	}
}

func TestCustomPolicyAllowDeny(t *testing.T) {
	dir := t.TempDir()
	mod := `package vajra
default allow = false
allow = true { input.action == "db_query" }
`
	if err := os.WriteFile(filepath.Join(dir, "test.rego"), []byte(mod), 0600); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(dir)
	if err := e.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	allow, err := e.Evaluate("a", "db_query", "tbl", nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !allow.Allowed {
		t.Fatalf("expected db_query allowed, got %+v", allow)
	}
	deny, err := e.Evaluate("a", "api_call", "x", nil)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if deny.Allowed {
		t.Fatalf("expected api_call denied, got %+v", deny)
	}
}

func TestInvalidPolicyRejected(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.rego"), []byte("package vajra\nallow = = ="), 0600); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(dir)
	if err := e.Load(); err == nil {
		t.Fatal("expected Load to reject invalid rego, got nil")
	}
}

func TestDefaultRulesNonEmpty(t *testing.T) {
	rs := DefaultRules()
	if rs == nil || len(rs.Rules) == 0 {
		t.Fatal("expected built-in default rules")
	}
}
