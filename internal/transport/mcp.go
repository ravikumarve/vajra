// Package transport implements MCP 2026-07-28 protocol types and handlers.
//
// MCP (Model Context Protocol) defines the communication between AI agents
// and tools/servers. VAJRA intercepts MCP tool calls to inject ephemeral
// credentials at the transport layer.
package transport

import "encoding/json"

// --- JSON-RPC 2.0 Message Types ---

// JSONRPCVersion is the JSON-RPC version used by MCP.
const JSONRPCVersion = "2.0"

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.Number     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.Number     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

// ErrorObject represents a JSON-RPC 2.0 error.
type ErrorObject struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Notification represents a JSON-RPC 2.0 notification (no ID).
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// --- MCP 2026-07-28 Methods ---

const (
	// Lifecycle
	MethodInitialize   = "initialize"
	MethodInitialized  = "notifications/initialized"
	MethodPing         = "ping"

	// Tools
	MethodToolsList   = "tools/list"
	MethodToolsCall   = "tools/call"

	// Resources
	MethodResourcesList       = "resources/list"
	MethodResourcesSubscribe  = "resources/subscribe"

	// Prompts
	MethodPromptsList = "prompts/list"
	MethodPromptsGet  = "prompts/get"

	// Logging
	MethodSetLevel = "logging/setLevel"

	// VAJRA extensions (custom)
	MethodVajraMint   = "vajra/mint"
	MethodVajraStatus = "vajra/status"
	MethodVajraRevoke = "vajra/revoke"
)

// --- Request/Response Types ---

// InitializeParams is the params for the initialize request.
type InitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities,omitempty"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// InitializeResult is the result of a successful initialize.
type InitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities"`
	ServerInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
}

// Tool represents a tool that an MCP server exposes.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToolsListResult is the result of a tools/list request.
type ToolsListResult struct {
	Tools      []Tool `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ToolsCallParams is the params for a tools/call request.
type ToolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolsCallResult is the result of a tools/call request.
type ToolsCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem is a piece of content returned by a tool.
type ContentItem struct {
	Type string          `json:"type"`
	Text string          `json:"text,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
	MIMEType string     `json:"mimeType,omitempty"`
}

// Resource represents a resource exposed by an MCP server.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

// ResourcesListResult is the result of a resources/list request.
type ResourcesListResult struct {
	Resources  []Resource `json:"resources"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// VajraMintParams is the params for the vajra/mint extension.
type VajraMintParams struct {
	Target string `json:"target"`
	Scope  string `json:"scope"`
	TTL    string `json:"ttl,omitempty"` // duration string like "5s"
}

// VajraMintResult is the result of a vajra/mint request.
type VajraMintResult struct {
	CredentialID string `json:"credential_id"`
	Target       string `json:"target"`
	Scope        string `json:"scope"`
	TTL          string `json:"ttl"`
	ExpiresAt    string `json:"expires_at"`
}

// VajraStatusResult is the result of a vajra/status request.
type VajraStatusResult struct {
	ActiveCredentials int    `json:"active_credentials"`
	Uptime            string `json:"uptime"`
	Version           string `json:"version"`
}

// --- Standard MCP Error Codes ---

var (
	ErrParseError     = &ErrorObject{Code: -32700, Message: "Parse error"}
	ErrInvalidRequest = &ErrorObject{Code: -32600, Message: "Invalid request"}
	ErrMethodNotFound = &ErrorObject{Code: -32601, Message: "Method not found"}
	ErrInvalidParams  = &ErrorObject{Code: -32602, Message: "Invalid params"}
	ErrInternalError  = &ErrorObject{Code: -32603, Message: "Internal error"}
)

// --- Helper Functions ---

// NewResponse creates a successful JSON-RPC response.
func NewResponse(id json.Number, result interface{}) *Response {
	data, _ := json.Marshal(result)
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  data,
	}
}

// NewErrorResponse creates an error JSON-RPC response.
func NewErrorResponse(id json.Number, errObj *ErrorObject) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error:   errObj,
	}
}
