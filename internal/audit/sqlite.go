// Package audit implements audit logging for VAJRA.
//
// Every credential mint and revocation is logged to an append-only
// SQLite database for compliance and debugging.
package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Entry represents a single audit log entry.
type Entry struct {
	ID           int64     `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	EventType    string    `json:"event_type"`    // "mint", "revoke", "expire", "deny"
	AgentID      string    `json:"agent_id"`      // Agent that requested the credential
	CredentialID string    `json:"credential_id"` // The credential involved
	Target       string    `json:"target"`        // Target system
	Scope        string    `json:"scope"`         // Scope of access
	TTL          string    `json:"ttl"`           // Time-to-live
	Decision     string    `json:"decision"`      // "allow", "deny"
	Reason       string    `json:"reason,omitempty"`
	Metadata     string    `json:"metadata,omitempty"` // JSON blob for extra context
}

// Logger manages the SQLite audit database.
type Logger struct {
	db         *sql.DB
	mu         sync.Mutex
	dbPath     string
	maxSize    int64 // max bytes before rotation
	insertStmt *sql.Stmt
}

// NewLogger creates or opens the audit database.
func NewLogger(dbPath string, maxSizeMB int) (*Logger, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating audit dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_sync=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("opening audit db: %w", err)
	}

	l := &Logger{
		db:      db,
		dbPath:  dbPath,
		maxSize: int64(maxSizeMB) * 1024 * 1024,
	}

	if err := l.initSchema(); err != nil {
		return nil, fmt.Errorf("initializing audit schema: %w", err)
	}

	// Record schema version (see migrations/). Fresh DBs are created at
	// the current version; future schema changes ship as migrations that
	// bump PRAGMA user_version instead of editing initSchema in place.
	if err := l.ensureSchemaVersion(); err != nil {
		return nil, fmt.Errorf("recording audit schema version: %w", err)
	}

	if err := l.prepareInsert(); err != nil {
		return nil, fmt.Errorf("preparing insert statement: %w", err)
	}

	return l, nil
}

// AuditSchemaVersion is the current audit DB schema version.
// Bump it only together with a new file in migrations/.
const AuditSchemaVersion = 1

// ensureSchemaVersion stamps fresh databases with the current version.
// Existing databases keep whatever version they were created with, so a
// future migrator can detect drift instead of silently assuming the schema.
func (l *Logger) ensureSchemaVersion() error {
	var v int
	if err := l.db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v == 0 {
		if _, err := l.db.Exec(fmt.Sprintf("PRAGMA user_version = %d", AuditSchemaVersion)); err != nil {
			return err
		}
		return nil
	}
	if v > AuditSchemaVersion {
		return fmt.Errorf("audit db schema v%d newer than binary v%d — upgrade vajra", v, AuditSchemaVersion)
	}
	return nil
}

// initSchema creates the audit table if it doesn't exist.
func (l *Logger) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL DEFAULT (datetime('now')),
		event_type TEXT NOT NULL,
		agent_id TEXT NOT NULL,
		credential_id TEXT NOT NULL,
		target TEXT NOT NULL,
		scope TEXT NOT NULL DEFAULT '',
		ttl TEXT NOT NULL DEFAULT '',
		decision TEXT NOT NULL DEFAULT 'allow',
		reason TEXT NOT NULL DEFAULT '',
		metadata TEXT NOT NULL DEFAULT '{}'
	);
	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_agent ON audit(agent_id);
	CREATE INDEX IF NOT EXISTS idx_audit_credential ON audit(credential_id);
	`
	_, err := l.db.Exec(schema)
	return err
}

// prepareInsert pre-compiles the insert statement for performance.
func (l *Logger) prepareInsert() error {
	stmt, err := l.db.Prepare(`
		INSERT INTO audit (event_type, agent_id, credential_id, target, scope, ttl, decision, reason, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	l.insertStmt = stmt
	return nil
}

// Log records an audit entry.
func (l *Logger) Log(eventType, agentID, credentialID, target, scope, ttl, decision, reason string, metadata map[string]interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	metaJSON := "{}"
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err == nil {
			metaJSON = string(data)
		}
	}

	// Concurrent dashboard/proxy writers can hit SQLITE_BUSY — retry with
	// backoff rather than dropping the audit row. Non-transient errors
	// (e.g. constraint violations) return immediately, no retry.
	err := withRetry(defaultRetryPolicy(), func() error {
		_, rerr := l.insertStmt.Exec(
			eventType, agentID, credentialID, target, scope, ttl, decision, reason, metaJSON,
		)
		return rerr
	})
	if err != nil {
		return fmt.Errorf("writing audit entry: %w", err)
	}

	return nil
}

// Query returns the most recent audit entries, limited by n.
func (l *Logger) Query(n int) ([]Entry, error) {
	rows, err := l.db.Query(`
		SELECT id, timestamp, event_type, agent_id, credential_id, target, scope, ttl, decision, reason, metadata
		FROM audit
		ORDER BY id DESC
		LIMIT ?
	`, n)
	if err != nil {
		return nil, fmt.Errorf("querying audit: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var ts string
		if err := rows.Scan(&e.ID, &ts, &e.EventType, &e.AgentID, &e.CredentialID,
			&e.Target, &e.Scope, &e.TTL, &e.Decision, &e.Reason, &e.Metadata); err != nil {
			return nil, fmt.Errorf("scanning audit row: %w", err)
		}
		e.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// Close shuts down the audit logger.
func (l *Logger) Close() error {
	if l.insertStmt != nil {
		l.insertStmt.Close()
	}
	return l.db.Close()
}
