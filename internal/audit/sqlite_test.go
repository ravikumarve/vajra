package audit

import (
	"path/filepath"
	"testing"
)

func TestLoggerRoundTripAndSchemaVersion(t *testing.T) {
	db := filepath.Join(t.TempDir(), "audit.db")
	l, err := NewLogger(db, 10)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer l.Close()

	if err := l.Log("mint", "agent-1", "cred-1", "postgres://db", "SELECT", "5s", "allow", "", nil); err != nil {
		t.Fatalf("Log: %v", err)
	}
	rows, err := l.Query(10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 1 || rows[0].AgentID != "agent-1" || rows[0].Decision != "allow" {
		t.Fatalf("unexpected rows: %+v", rows)
	}

	var v int
	if err := l.db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	if v != AuditSchemaVersion {
		t.Fatalf("expected schema v%d, got %d", AuditSchemaVersion, v)
	}
}
