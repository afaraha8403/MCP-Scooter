package fixtures

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// MockMCPServer is a simple mock MCP server for testing.
type MockMCPServer struct {
	mu        sync.RWMutex
	tools     []map[string]interface{}
	responses map[string]interface{}
	server    *http.Server
}

// NewMockMCPServer creates a new mock MCP server.
func NewMockMCPServer() *MockMCPServer {
	return &MockMCPServer{
		tools: []map[string]interface{}{
			{
				"name":        "echo",
				"description": "Echoes back the input",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"message": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
		responses: make(map[string]interface{}),
	}
}

// Start starts the mock server on the given port.
func (s *MockMCPServer) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Mock server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the mock server.
func (s *MockMCPServer) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

func (s *MockMCPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send endpoint event
	fmt.Fprintf(w, "event: endpoint\ndata: /message\n\n")
	w.(http.Flusher).Flush()

	// Keep connection open
	<-r.Context().Done()
}

func (s *MockMCPServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      interface{}     `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var result interface{}
	switch req.Method {
	case "initialize":
		result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"serverInfo":      map[string]string{"name": "mock-server", "version": "0.1.0"},
		}
	case "tools/list":
		s.mu.RLock()
		result = map[string]interface{}{
			"tools": s.tools,
		}
		s.mu.RUnlock()
	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		json.Unmarshal(req.Params, &params)

		s.mu.RLock()
		if resp, ok := s.responses[params.Name]; ok {
			result = resp
		} else {
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("Mock response for %s", params.Name)},
				},
			}
		}
		s.mu.RUnlock()
	default:
		result = map[string]interface{}{"error": "method not found"}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"result":  result,
	})
}

// SetToolResponse sets a custom response for a tool.
func (s *MockMCPServer) SetToolResponse(toolName string, response interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[toolName] = response
}
