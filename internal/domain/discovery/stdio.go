package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/registry"
	"github.com/mcp-scooter/scooter/internal/logger"
)

// StdioWorker handles execution of a persistent MCP server over stdio.
// The server process stays running and communicates via JSON-RPC over stdin/stdout.
type StdioWorker struct {
	command string
	args    []string
	env     map[string]string
	cmd     *exec.Cmd
	ctx     context.Context

	stdin  io.WriteCloser
	stdout *bufio.Reader

	mu          sync.Mutex
	initialized bool
	requestID   int64

	// Cached tool definitions from the server
	tools []registry.Tool
}

func NewStdioWorker(ctx context.Context, command string, args []string) *StdioWorker {
	return &StdioWorker{
		command:   command,
		args:      args,
		ctx:       ctx,
		requestID: 1,
	}
}

// Start spawns the MCP server process and performs the initialize handshake.
func (w *StdioWorker) Start(env map[string]string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cmd != nil && w.cmd.Process != nil {
		// Already running
		return nil
	}

	w.env = env
	w.cmd = exec.CommandContext(w.ctx, w.command, w.args...)

	// Set up pipes for communication
	stdin, err := w.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	w.stdin = stdin

	stdout, err := w.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	w.stdout = bufio.NewReader(stdout)

	// Capture stderr and log it
	stderr, err := w.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			logger.AddLog("INFO", fmt.Sprintf("[%s] %s", w.command, scanner.Text()))
		}
	}()

	// Merge provided env with current process env
	w.cmd.Env = os.Environ()
	for k, v := range env {
		w.cmd.Env = append(w.cmd.Env, k+"="+v)
	}

	// Start the process
	if err := w.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Perform MCP initialize handshake
	if err := w.initializeHandshake(); err != nil {
		w.cmd.Process.Kill()
		return fmt.Errorf("MCP initialize handshake failed: %w", err)
	}

	w.initialized = true
	return nil
}

// initializeHandshake performs the MCP protocol initialization.
func (w *StdioWorker) initializeHandshake() error {
	// Send initialize request
	initReq := registry.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      w.nextID(),
		Method:  "initialize",
	}
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "mcp-scooter",
			"version": "0.1.0",
		},
	}
	initReq.Params, _ = json.Marshal(initParams)

	resp, err := w.sendRequest(initReq)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
	}

	// Send initialized notification
	initializedNotif := registry.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	if err := w.sendNotification(initializedNotif); err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	// Fetch tools list
	if err := w.fetchTools(); err != nil {
		fmt.Printf("[StdioWorker] Warning: failed to fetch tools: %v\n", err)
		// Don't fail initialization if tools/list fails - some servers may not support it
	}

	return nil
}

// fetchTools retrieves the list of tools from the MCP server.
func (w *StdioWorker) fetchTools() error {
	req := registry.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      w.nextID(),
		Method:  "tools/list",
	}

	resp, err := w.sendRequest(req)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	// Parse tools from response
	if resp.Result != nil {
		var result struct {
			Tools []registry.Tool `json:"tools"`
		}
		resultBytes, _ := json.Marshal(resp.Result)
		if err := json.Unmarshal(resultBytes, &result); err == nil {
			w.tools = result.Tools
			fmt.Printf("[StdioWorker] Discovered %d tools from server\n", len(w.tools))
		}
	}

	return nil
}

// GetTools returns the cached tool definitions from the server.
func (w *StdioWorker) GetTools() []registry.Tool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.tools
}

// Execute sends a tools/call request to the running MCP server.
func (w *StdioWorker) Execute(stdin io.Reader, stdout io.Writer, env map[string]string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Start server if not running
	if !w.initialized {
		w.mu.Unlock()
		if err := w.Start(env); err != nil {
			return err
		}
		w.mu.Lock()
	}

	// Read the incoming request from stdin
	var req registry.JSONRPCRequest
	decoder := json.NewDecoder(stdin)
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request: %w", err)
	}

	// Forward the request to the MCP server
	req.ID = w.nextID()
	resp, err := w.sendRequest(req)
	if err != nil {
		return fmt.Errorf("failed to send request to MCP server: %w", err)
	}

	// Write response to stdout
	encoder := json.NewEncoder(stdout)
	return encoder.Encode(resp)
}

// CallTool directly calls a tool on the MCP server.
func (w *StdioWorker) CallTool(name string, arguments map[string]interface{}) (*registry.JSONRPCResponse, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.initialized {
		return nil, fmt.Errorf("server not initialized")
	}

	req := registry.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      w.nextID(),
		Method:  "tools/call",
	}
	callParams := struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}{
		Name:      name,
		Arguments: arguments,
	}
	req.Params, _ = json.Marshal(callParams)

	return w.sendRequest(req)
}

// sendRequest sends a JSON-RPC request and waits for a response.
func (w *StdioWorker) sendRequest(req registry.JSONRPCRequest) (*registry.JSONRPCResponse, error) {
	// Write request
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	reqBytes = append(reqBytes, '\n')

	startTime := time.Now()
	if _, err := w.stdin.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	logger.AddLog("INFO", fmt.Sprintf("[%s] Sent request %v (%s), waiting for response...", w.command, req.ID, req.Method))

	// Read response with timeout
	responseChan := make(chan *registry.JSONRPCResponse, 1)
	errorChan := make(chan error, 1)

	go func() {
		line, err := w.stdout.ReadBytes('\n')
		if err != nil {
			errorChan <- err
			return
		}

		var resp registry.JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			errorChan <- fmt.Errorf("failed to parse response: %w", err)
			return
		}
		responseChan <- &resp
	}()

	select {
	case resp := <-responseChan:
		duration := time.Since(startTime)
		logger.AddLog("INFO", fmt.Sprintf("[%s] Received response for %v in %v", w.command, req.ID, duration))
		return resp, nil
	case err := <-errorChan:
		duration := time.Since(startTime)
		logger.AddLog("ERROR", fmt.Sprintf("[%s] Error reading response for %v after %v: %v", w.command, req.ID, duration, err))
		return nil, err
	case <-time.After(30 * time.Second):
		duration := time.Since(startTime)
		logger.AddLog("ERROR", fmt.Sprintf("[%s] Timeout waiting for response for %v (%s) after %v", w.command, req.ID, req.Method, duration))
		return nil, fmt.Errorf("timeout waiting for response")
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	}
}

// sendNotification sends a JSON-RPC notification (no response expected).
func (w *StdioWorker) sendNotification(req registry.JSONRPCRequest) error {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}
	reqBytes = append(reqBytes, '\n')

	_, err = w.stdin.Write(reqBytes)
	return err
}

func (w *StdioWorker) nextID() int64 {
	w.requestID++
	return w.requestID
}

// IsRunning returns whether the server process is running.
func (w *StdioWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.initialized && w.cmd != nil && w.cmd.Process != nil
}

func (w *StdioWorker) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.initialized = false

	if w.stdin != nil {
		w.stdin.Close()
	}

	if w.cmd != nil && w.cmd.Process != nil {
		// Try graceful shutdown first
		w.cmd.Process.Signal(os.Interrupt)
		
		// Give it a moment to shut down
		done := make(chan error, 1)
		go func() {
			done <- w.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(2 * time.Second):
			// Force kill
			w.cmd.Process.Kill()
		}
	}

	return nil
}
