package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// MCPTestClient is a test client for MCP Scooter's HTTP/SSE gateway.
type MCPTestClient struct {
	BaseURL    string // e.g., "http://127.0.0.1:3001"
	ProfileID  string // e.g., "work"
	APIKey     string
	HTTPClient *http.Client

	mu           sync.Mutex
	lastID       int
	sseEventChan chan SSEEvent
	done         chan struct{}
}

// SSEEvent represents an event received over SSE.
type SSEEvent struct {
	Event string
	Data  string
}

// JSONRPCRequest represents a standard JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a standard JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError represents a standard JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// NewClient creates a new MCP test client.
func NewClient(baseURL, profileID, apiKey string) *MCPTestClient {
	return &MCPTestClient{
		BaseURL:    baseURL,
		ProfileID:  profileID,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		sseEventChan: make(chan SSEEvent, 100),
		done:         make(chan struct{}),
	}
}

// nextID generates the next JSON-RPC request ID.
func (c *MCPTestClient) nextID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastID++
	return c.lastID
}

// Initialize performs the MCP handshake.
func (c *MCPTestClient) Initialize() (*JSONRPCResponse, error) {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "mcp-scooter-test-client",
			"version": "0.1.0",
		},
	}
	return c.Call("initialize", params)
}

// ListTools lists available tools.
func (c *MCPTestClient) ListTools() (*JSONRPCResponse, error) {
	return c.Call("tools/list", nil)
}

// CallTool calls a specific tool.
func (c *MCPTestClient) CallTool(name string, args map[string]interface{}) (*JSONRPCResponse, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}
	return c.Call("tools/call", params)
}

// Call sends a JSON-RPC request to the gateway.
func (c *MCPTestClient) Call(method string, params interface{}) (*JSONRPCResponse, error) {
	id := c.nextID()
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
	}

	if params != nil {
		p, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		reqBody.Params = p
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/profiles/%s/message", c.BaseURL, c.ProfileID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var jsonResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return nil, err
	}

	return &jsonResp, nil
}

// StartSSE starts listening for SSE events.
func (c *MCPTestClient) StartSSE() error {
	url := fmt.Sprintf("%s/profiles/%s/sse", c.BaseURL, c.ProfileID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("unexpected status code for SSE: %d", resp.StatusCode)
	}

	go func() {
		defer resp.Body.Close()
		reader := io.Reader(resp.Body)
		buf := make([]byte, 4096)
		for {
			select {
			case <-c.done:
				return
			default:
				n, err := reader.Read(buf)
				if err != nil {
					return
				}
				// Basic SSE parsing (very simplified)
				fmt.Printf("SSE data: %s\n", string(buf[:n]))
			}
		}
	}()

	return nil
}

// Close closes the client.
func (c *MCPTestClient) Close() {
	close(c.done)
}
