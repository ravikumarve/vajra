package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// SSEServer implements an MCP server over HTTP SSE (Server-Sent Events).
// Used for remote agents or multi-process setups.
type SSEServer struct {
	addr    string
	handler MessageHandler
	server  *http.Server
	clients map[string]chan []byte
	mu      sync.RWMutex
	done    chan struct{}
}

// NewSSEServer creates a new SSE-based MCP server on the given address.
func NewSSEServer(addr string, handler MessageHandler) *SSEServer {
	s := &SSEServer{
		addr:    addr,
		handler: handler,
		clients: make(map[string]chan []byte),
		done:    make(chan struct{}),
	}
	return s
}

// Serve starts the HTTP SSE server.
func (s *SSEServer) Serve() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/events", s.handleEvents)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	log.Printf("[vajra] SSE transport: listening on %s/mcp", s.addr)

	// Start listening
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("SSE listen error: %w", err)
	}

	err = s.server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("SSE serve error: %w", err)
	}
	return nil
}

// Close shuts down the SSE server gracefully.
func (s *SSEServer) Close() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleMCP processes incoming MCP JSON-RPC messages via POST.
func (s *SSEServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the JSON-RPC message
	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Process the request
	respBytes, err := s.processRequest(&request)
	if err != nil {
		http.Error(w, fmt.Sprintf("Processing error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

// handleHealth returns a simple health check response.
func (s *SSEServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "vajra"})
}

// handleEvents implements Server-Sent Events for real-time audit streaming.
func (s *SSEServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientID := fmt.Sprintf("client_%x", time.Now().UnixNano())
	eventCh := make(chan []byte, 64)

	s.mu.Lock()
	s.clients[clientID] = eventCh
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		s.mu.Unlock()
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		}
	}
}

// BroadcastEvent sends an event to all connected SSE clients.
func (s *SSEServer) BroadcastEvent(data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.clients {
		select {
		case ch <- data:
		default:
			// Drop if client is slow
		}
	}
}

// processRequest handles a decoded JSON-RPC request and returns response bytes.
func (s *SSEServer) processRequest(req *Request) ([]byte, error) {
	if req == nil {
		msg, _ := json.Marshal(NewErrorResponse("0", ErrInvalidRequest))
		// Use a dummy ID since req is nil
		var resp Response
		json.Unmarshal(msg, &resp)
		return json.Marshal(resp)
	}

	// Basic validation
	if req.JSONRPC != JSONRPCVersion {
		resp := NewErrorResponse(req.ID, ErrInvalidRequest)
		return json.Marshal(resp)
	}

	// Route based on method
	var result interface{}
	switch req.Method {
	case MethodPing:
		result = map[string]string{"status": "ok"}
	case MethodInitialize:
		result = InitializeResult{
			ProtocolVersion: "2026-07-28",
			ServerInfo: struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}{
				Name:    "vajra",
				Version: "0.1.0",
			},
		}
	case MethodToolsList:
		result = ToolsListResult{
			Tools: []Tool{
				{
					Name:        "vajra_mint",
					Description: "Mint an ephemeral credential for AI agent access",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"target":{"type":"string"},"scope":{"type":"string"},"ttl":{"type":"string"}}}`),
				},
			},
		}
	default:
		resp := NewErrorResponse(req.ID, ErrMethodNotFound)
		return json.Marshal(resp)
	}

	resp := NewResponse(req.ID, result)
	return json.Marshal(resp)
}


