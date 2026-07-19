package transport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// StdioServer implements an MCP server over stdin/stdout.
// Used for local agent processes (pipe-based MCP communication).
type StdioServer struct {
	reader    *bufio.Reader
	writer    *bufio.Writer
	handler   MessageHandler
	closeOnce sync.Once
	done      chan struct{}
}

// MessageHandler processes incoming MCP requests and returns responses.
type MessageHandler interface {
	HandleMessage(msg []byte) ([]byte, error)
}

// NewStdioServer creates a new stdio-based MCP server.
func NewStdioServer(handler MessageHandler) *StdioServer {
	return &StdioServer{
		reader:  bufio.NewReader(os.Stdin),
		writer:  bufio.NewWriter(os.Stdout),
		handler: handler,
		done:    make(chan struct{}),
	}
}

// NewStdioServerWithIO creates a stdio server with custom reader/writer.
func NewStdioServerWithIO(r io.Reader, w io.Writer, handler MessageHandler) *StdioServer {
	return &StdioServer{
		reader:  bufio.NewReader(r),
		writer:  bufio.NewWriter(w),
		handler: handler,
		done:    make(chan struct{}),
	}
}

// Serve starts the stdio MCP server loop. It reads JSON-RPC messages
// from stdin, processes them, and writes responses to stdout.
func (s *StdioServer) Serve() error {
	log.Println("[vajra] stdio transport: starting MCP server on stdin/stdout")
	defer close(s.done)

	for {
		// Read a newline-delimited JSON message
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("[vajra] stdio transport: stdin closed, shutting down")
				return nil
			}
			return fmt.Errorf("stdio read error: %w", err)
		}

		// Skip empty lines
		if len(line) <= 1 {
			continue
		}

		// Handle the message
		response, err := s.handler.HandleMessage(line)
		if err != nil {
			log.Printf("[vajra] stdio transport: handler error: %v", err)
			continue
		}

		// Write response
		if response != nil {
			_, err = s.writer.Write(response)
			if err != nil {
				return fmt.Errorf("stdio write error: %w", err)
			}
			s.writer.Write([]byte("\n"))
			s.writer.Flush()
		}
	}
}

// Close shuts down the stdio server.
func (s *StdioServer) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
	})
	return nil
}

// Done returns a channel that is closed when the server shuts down.
func (s *StdioServer) Done() <-chan struct{} {
	return s.done
}

// HandleMessage implements a basic JSON-RPC dispatcher.
// This is a simple router that can be embedded or wrapped.
func HandleMessage(msg []byte) ([]byte, error) {
	// Parse the request
	var request Request
	if err := json.Unmarshal(msg, &request); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Basic validation
	if request.JSONRPC != JSONRPCVersion {
		resp := NewErrorResponse(request.ID, ErrInvalidRequest)
		return json.Marshal(resp)
	}

	// Route based on method
	var result interface{}
	switch request.Method {
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
		resp := NewErrorResponse(request.ID, ErrMethodNotFound)
		return json.Marshal(resp)
	}

	resp := NewResponse(request.ID, result)
	return json.Marshal(resp)
}
