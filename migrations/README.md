# Migrations — audit SQLite schema versioning

The audit logger owns its schema (`internal/audit/sqlite.go`, `initSchema`
with `IF NOT EXISTS`) and stamps fresh databases with
`PRAGMA user_version = AuditSchemaVersion` (currently 1). A binary that
opens a DB stamped *newer* than itself refuses to start instead of
corrupting the trail.

- `001_audit_schema.sql` — baseline reference, mirrors `initSchema`.
- **Rule:** schema changes ship as `002_<name>.sql` + a migrator keyed off
  `PRAGMA user_version`, and bump `AuditSchemaVersion`. Never edit `001`
  after release — buyers' audit trails depend on it.
