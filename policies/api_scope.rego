# VAJRA default policy: API access scoping
# Path: ~/.vajra/policies/api_scope.rego
package vajra.api

import future.keywords.if

# Default: deny all API access
default allow = false

# Allow GET requests on user-related endpoints
allow if {
    input.action == "api_call"
    input.method == "GET"
    startswith(input.path, "/v2/users")
}

# Allow GET requests on organization endpoints
allow if {
    input.action == "api_call"
    input.method == "GET"
    startswith(input.path, "/v2/organizations")
}

# Allow POST for specific actions
allow if {
    input.action == "api_call"
    input.method == "POST"
    input.path == "/v2/users/query"
}

# Deny write operations by default
deny["write operations require admin approval"] {
    input.action == "api_call"
    input.method in {"POST", "PUT", "PATCH", "DELETE"}
    input.path != "/v2/users/query"
}

# Rate limiting hints
max_requests_per_minute = 100 {
    input.method == "GET"
}

max_requests_per_minute = 10 {
    input.method == "POST"
}
