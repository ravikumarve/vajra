-- Vajra migration 001 — audit schema baseline (fresh-install reference).
-- Mirrors internal/audit/sqlite.go initSchema + AuditSchemaVersion = 1.
-- The logger applies this automatically (CREATE TABLE IF NOT EXISTS) and
-- stamps PRAGMA user_version. Future changes MUST ship as 002_<name>.sql
-- with a migrator that checks user_version — never edit this file.

PRAGMA journal_mode=WAL;

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
