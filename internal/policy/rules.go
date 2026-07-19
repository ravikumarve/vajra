package policy

import (
	"encoding/json"
)

// Rule represents a single policy rule.
type Rule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Resource    string `json:"resource"`
	Effect      string `json:"effect"` // "allow" or "deny"
	Priority    int    `json:"priority"`
}

// RuleSet is a collection of rules.
type RuleSet struct {
	Rules []Rule `json:"rules"`
}

// DefaultRules returns the built-in default rule set.
func DefaultRules() *RuleSet {
	return &RuleSet{
		Rules: []Rule{
			{
				Name:        "default-db-read",
				Description: "Allow read-only database queries",
				Action:      "db_query",
				Resource:    "*",
				Effect:      "allow",
				Priority:    0,
			},
			{
				Name:        "default-api-read",
				Description: "Allow read-only API calls",
				Action:      "api_call",
				Resource:    "*",
				Effect:      "allow",
				Priority:    0,
			},
			{
				Name:        "deny-write-default",
				Description: "Deny write operations by default",
				Action:      "db_write",
				Resource:    "*",
				Effect:      "deny",
				Priority:    100,
			},
		},
	}
}

// Ensure json is used
var _ = json.Marshal
