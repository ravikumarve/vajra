# VAJRA default policy: Database access scoping
# Path: ~/.vajra/policies/db_scope.rego
package vajra.db

import future.keywords.if
import future.keywords.in

# Default: deny all database access
default allow = false

# Allow SELECT on specific tables for read-only agents
allow if {
    input.action == "db_query"
    input.resource == "users"
    input.method == "SELECT"
    input.columns != ["password", "ssn", "secret"]
}

allow if {
    input.action == "db_query"
    input.resource == "organizations"
    input.method == "SELECT"
}

# Row-level filtering: agents can only see their own org's data
row_filter = filter {
    input.resource == "users"
    filter = sprintf("org_id = '%s'", [input.agent_org])
}

# Column-level redaction
allowed_columns[resource] = cols {
    resource == "users"
    cols = ["id", "email", "name", "plan", "created_at"]
}

allowed_columns[resource] = cols {
    resource == "organizations"
    cols = ["id", "name", "plan"]
}

# TTL based on operation type
suggested_ttl = "5s" {
    input.method == "SELECT"
}

suggested_ttl = "2s" {
    input.method == "INSERT"
}
