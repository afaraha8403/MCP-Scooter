package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/registry"
	"github.com/mcp-scooter/scooter/internal/logger"
)

// =============================================================================
// StdioWorker - MCP Server Process Manager
// =============================================================================
//
// OVERVIEW:
// StdioWorker manages a persistent external MCP (Model Context Protocol) server
// that communicates over standard input/output using JSON-RPC 2.0.
//
// DATA FLOW:
//
//   ┌─────────────────┐         stdin (JSON-RPC)        ┌─────────────────┐
//   │                 │ ──────────────────────────────► │                 │
//   │  MCP Scooter    │                                 │  External MCP   │
//   │  (this code)    │                                 │  Server (npx)   │
//   │                 │ ◄────────────────────────────── │                 │
//   └─────────────────┘         stdout (JSON-RPC)       └─────────────────┘
//                                    │
//                               stderr (logs/errors)
//                                    ▼
//                              [logged + monitored]
//
// LIFECYCLE:
//   1. NewStdioWorker() - Create worker with command/args
//   2. Start()          - Spawn process, perform MCP handshake
//   3. CallTool()       - Send tool invocation requests
//   4. Close()          - Gracefully shutdown the process
//
// MUTEX HANDLING (CRITICAL - READ CAREFULLY):
// The mutex (w.mu) protects access to shared state (cmd, stdin, stdout, initialized).
// However, the Start() method has COMPLEX locking behavior because:
//   - We need the lock to set up the process
//   - We must RELEASE the lock before calling initializeHandshake()
//   - initializeHandshake() calls sendRequest() which does NOT acquire the lock
//   - After handshake, we re-acquire the lock to set initialized=true
//
// DO NOT use "defer w.mu.Unlock()" in Start() - it will cause double-unlock panics!
//
// =============================================================================

// StdioWorker handles execution of a persistent MCP server over stdio.
// The server process stays running and communicates via JSON-RPC over stdin/stdout.
type StdioWorker struct {
	// Command configuration (immutable after creation)
	command string   // The executable to run (e.g., "npx")
	args    []string // Command arguments (e.g., ["-y", "@modelcontextprotocol/server-brave-search"])

	// Runtime state (protected by mu)
	env map[string]string // Environment variables passed to the process
	cmd *exec.Cmd         // The running process handle
	ctx context.Context   // Context for cancellation

	// I/O pipes to the child process (protected by mu)
	stdin  io.WriteCloser // We write JSON-RPC requests here → child's stdin
	stdout *bufio.Reader  // We read JSON-RPC responses here ← child's stdout

	// Synchronization
	mu          sync.Mutex // Protects all mutable state below
	initialized bool       // True after successful MCP handshake
	requestID   int64      // Auto-incrementing JSON-RPC request ID

	// Cached data from the MCP server
	tools []registry.Tool // Tool definitions fetched from the server
}

// NewStdioWorker creates a new StdioWorker but does NOT start the process.
// Call Start() to actually spawn the MCP server.
func NewStdioWorker(ctx context.Context, command string, args []string) *StdioWorker {
	return &StdioWorker{
		command:   command,
		args:      args,
		ctx:       ctx,
		requestID: 1, // JSON-RPC IDs start at 1
	}
}

// =============================================================================
// Start - Spawn Process and Perform MCP Handshake
// =============================================================================
//
// MUTEX BEHAVIOR (CRITICAL):
// This function has MANUAL mutex management. DO NOT add "defer w.mu.Unlock()".
//
// Lock timeline:
//   1. Lock acquired at entry
//   2. Lock released BEFORE initializeHandshake() (line ~120)
//   3. Lock re-acquired in error handlers and at the end
//   4. Lock released before each return
//
// Why? Because initializeHandshake() → sendRequest() needs to write to stdin,
// and if we held the lock, we'd deadlock or need recursive mutexes.
//
// =============================================================================

// Start spawns the MCP server process and performs the initialize handshake.
// This is safe to call multiple times - it's a no-op if already running.
func (w *StdioWorker) Start(env map[string]string) error {
	// =========================================================================
	// PHASE 1: Setup (mutex held)
	// =========================================================================
	w.mu.Lock()
	// NOTE: We do NOT use "defer w.mu.Unlock()" here because we need to
	// release the lock before the handshake and re-acquire it afterward.
	// Using defer would cause a double-unlock panic when we manually unlock.

	// Idempotency check: if already running, return immediately
	if w.cmd != nil && w.cmd.Process != nil {
		w.mu.Unlock()
		return nil
	}

	// Store environment variables for later use
	w.env = env

	// Create the command with context (allows cancellation)
	w.cmd = exec.CommandContext(w.ctx, w.command, w.args...)

	// -------------------------------------------------------------------------
	// Set up stdin pipe: We write JSON-RPC requests here
	// -------------------------------------------------------------------------
	stdin, err := w.cmd.StdinPipe()
	if err != nil {
		w.mu.Unlock()
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	w.stdin = stdin

	// -------------------------------------------------------------------------
	// Set up stdout pipe: We read JSON-RPC responses here
	// -------------------------------------------------------------------------
	stdout, err := w.cmd.StdoutPipe()
	if err != nil {
		w.mu.Unlock()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	w.stdout = bufio.NewReader(stdout)

	// -------------------------------------------------------------------------
	// Set up stderr pipe: For logging and error detection
	// -------------------------------------------------------------------------
	stderr, err := w.cmd.StderrPipe()
	if err != nil {
		w.mu.Unlock()
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Channel to signal critical errors detected in stderr
	// Buffered with size 1 so the goroutine doesn't block
	criticalErrChan := make(chan string, 1)

	// -------------------------------------------------------------------------
	// Stderr monitoring goroutine
	// -------------------------------------------------------------------------
	// This runs in the background, logging all stderr output and detecting
	// critical errors that would cause the MCP handshake to fail.
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			// Log all stderr output for debugging
			logger.AddLog("INFO", fmt.Sprintf("[%s] %s", w.command, line))

			// Detect critical error patterns that indicate the server failed to start.
			// These errors typically cause EOF on stdout later, so we catch them early.
			// IMPORTANT: We ignore "npm WARN" messages because they often contain
			// the word "Error" but are not actually fatal errors.
			isCriticalError := (strings.Contains(line, "environment variable is required") ||
				strings.Contains(line, "Error:") ||
				strings.Contains(line, "Exception"))
			isNpmWarning := strings.Contains(line, "npm WARN")

			if isCriticalError && !isNpmWarning {
				logger.AddLog("ERROR", fmt.Sprintf("[%s] Critical error detected in stderr: %s", w.command, line))
				// Non-blocking send - only the first error is captured
				select {
				case criticalErrChan <- line:
				default:
					// Channel already has an error, ignore subsequent ones
				}
			}
		}
	}()

	// -------------------------------------------------------------------------
	// Merge environment variables
	// -------------------------------------------------------------------------
	// Start with the current process's environment, then add/override with
	// the provided env map (e.g., API keys like BRAVE_API_KEY)
	w.cmd.Env = os.Environ()
	for k, v := range env {
		w.cmd.Env = append(w.cmd.Env, k+"="+v)
	}

	// -------------------------------------------------------------------------
	// Start the child process
	// -------------------------------------------------------------------------
	if err := w.cmd.Start(); err != nil {
		w.mu.Unlock()
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// =========================================================================
	// PHASE 2: MCP Handshake (mutex RELEASED)
	// =========================================================================
	// CRITICAL: We MUST release the lock here because initializeHandshake()
	// calls sendRequest(), which writes to stdin. If we held the lock,
	// other methods couldn't access the worker during the handshake.
	w.mu.Unlock()

	// Channel to receive the handshake result
	type handshakeResult struct {
		err error
	}
	resChan := make(chan handshakeResult, 1)

	// Run the handshake in a goroutine so we can race against:
	// 1. Critical errors from stderr
	// 2. A timeout (60 seconds for slow npx downloads on Windows)
	go func() {
		resChan <- handshakeResult{err: w.initializeHandshake()}
	}()

	// -------------------------------------------------------------------------
	// Wait for handshake completion, error, or timeout
	// -------------------------------------------------------------------------
	select {
	case res := <-resChan:
		// Handshake completed (successfully or with error)
		if res.err != nil {
			// Handshake failed - kill the process and return error
			w.mu.Lock()
			if w.cmd != nil && w.cmd.Process != nil {
				w.cmd.Process.Kill()
			}
			w.mu.Unlock()
			return fmt.Errorf("MCP initialize handshake failed: %w", res.err)
		}
		// Handshake succeeded - fall through to set initialized=true

	case critLine := <-criticalErrChan:
		// Critical error detected in stderr before handshake completed
		w.mu.Lock()
		if w.cmd != nil && w.cmd.Process != nil {
			w.cmd.Process.Kill()
		}
		w.mu.Unlock()
		return fmt.Errorf("MCP server failed with critical error: %s", critLine)

	case <-time.After(60 * time.Second):
		// Timeout - npx can be slow on Windows, especially first run
		w.mu.Lock()
		if w.cmd != nil && w.cmd.Process != nil {
			w.cmd.Process.Kill()
		}
		w.mu.Unlock()
		return fmt.Errorf("MCP server timed out during initialization")
	}

	// =========================================================================
	// PHASE 3: Mark as initialized (mutex held briefly)
	// =========================================================================
	w.mu.Lock()
	w.initialized = true
	w.mu.Unlock()
	return nil
}

// =============================================================================
// MCP Protocol Implementation
// =============================================================================
//
// The MCP (Model Context Protocol) handshake follows this sequence:
//
//   Client (us)                          Server (external process)
//       │                                        │
//       │─────── initialize request ───────────►│
//       │                                        │
//       │◄────── initialize response ───────────│
//       │                                        │
//       │─────── initialized notification ─────►│
//       │                                        │
//       │─────── tools/list request ───────────►│
//       │                                        │
//       │◄────── tools/list response ───────────│
//       │                                        │
//   [Ready to call tools]
//
// =============================================================================

// initializeHandshake performs the MCP protocol initialization sequence.
// This is called WITHOUT the mutex held (from a goroutine in Start()).
func (w *StdioWorker) initializeHandshake() error {
	// -------------------------------------------------------------------------
	// Step 1: Send "initialize" request
	// -------------------------------------------------------------------------
	// This tells the server who we are and what protocol version we support.
	initReq := registry.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      w.nextID(),
		Method:  "initialize",
	}
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05", // MCP protocol version
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

	// -------------------------------------------------------------------------
	// Step 2: Send "initialized" notification
	// -------------------------------------------------------------------------
	// This is a notification (no response expected) that tells the server
	// we've processed its initialize response and are ready to proceed.
	initializedNotif := registry.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		// Note: No ID field for notifications
	}
	if err := w.sendNotification(initializedNotif); err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	// -------------------------------------------------------------------------
	// Step 3: Fetch available tools
	// -------------------------------------------------------------------------
	// This is optional - some servers might not support tools/list.
	// We don't fail initialization if this fails.
	if err := w.fetchTools(); err != nil {
		fmt.Printf("[StdioWorker] Warning: failed to fetch tools: %v\n", err)
	}

	return nil
}

// fetchTools retrieves the list of available tools from the MCP server.
// Some servers need a moment after initialization before they're ready,
// so we retry up to 3 times with a 500ms delay.
func (w *StdioWorker) fetchTools() error {
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		req := registry.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      w.nextID(),
			Method:  "tools/list",
		}

		resp, err := w.sendRequest(req)
		if err == nil {
			if resp.Error != nil {
				lastErr = fmt.Errorf("tools/list error: %s", resp.Error.Message)
			} else {
				// Parse the tools from the response
				if resp.Result != nil {
					var result struct {
						Tools []registry.Tool `json:"tools"`
					}
					// Re-marshal and unmarshal to convert interface{} to struct
					resultBytes, _ := json.Marshal(resp.Result)
					if err := json.Unmarshal(resultBytes, &result); err == nil {
						w.tools = result.Tools
						logger.AddLog("INFO", fmt.Sprintf("[StdioWorker] Discovered %d tools from server", len(w.tools)))
						return nil
					}
				}
				return nil
			}
		} else {
			lastErr = err
		}

		// Retry with delay (except on last attempt)
		if attempt < 2 {
			logger.AddLog("INFO", fmt.Sprintf("[StdioWorker] tools/list failed, retrying in 500ms... (%v)", lastErr))
			time.Sleep(500 * time.Millisecond)
		}
	}

	return lastErr
}

// =============================================================================
// Public API Methods
// =============================================================================

// GetTools returns the cached tool definitions from the server.
// Thread-safe.
func (w *StdioWorker) GetTools() []registry.Tool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.tools
}

// RefreshTools re-fetches the list of tools from the running MCP server.
// Useful if the server's tools might have changed.
func (w *StdioWorker) RefreshTools() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.initialized || w.cmd == nil || w.cmd.Process == nil {
		return fmt.Errorf("server not running")
	}

	logger.AddLog("INFO", fmt.Sprintf("[StdioWorker] Refreshing tools from %s...", w.command))
	return w.fetchTools()
}

// Execute sends a tools/call request to the running MCP server.
// This is the legacy interface that reads from an io.Reader and writes to io.Writer.
// Prefer CallTool() for direct invocation.
func (w *StdioWorker) Execute(stdin io.Reader, stdout io.Writer, env map[string]string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Auto-start if not running
	if !w.initialized {
		w.mu.Unlock()
		if err := w.Start(env); err != nil {
			return err
		}
		w.mu.Lock()
	}

	// Read the incoming JSON-RPC request
	var req registry.JSONRPCRequest
	decoder := json.NewDecoder(stdin)
	if err := decoder.Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request: %w", err)
	}

	// Assign a new request ID and forward to the MCP server
	req.ID = w.nextID()
	resp, err := w.sendRequest(req)
	if err != nil {
		return fmt.Errorf("failed to send request to MCP server: %w", err)
	}

	// Write the response back
	encoder := json.NewEncoder(stdout)
	return encoder.Encode(resp)
}

// CallTool directly calls a tool on the MCP server.
// This is the preferred method for invoking tools.
//
// Example:
//
//	resp, err := worker.CallTool("brave_web_search", map[string]interface{}{
//	    "query": "hello world",
//	    "count": 10,
//	})
func (w *StdioWorker) CallTool(name string, arguments map[string]interface{}) (*registry.JSONRPCResponse, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.initialized {
		return nil, fmt.Errorf("server not initialized")
	}

	// Build the tools/call request
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

// =============================================================================
// Low-Level I/O Methods
// =============================================================================

// sendRequest sends a JSON-RPC request and waits for a response.
// This method does NOT acquire the mutex - callers must handle locking.
//
// Data flow:
//   1. Marshal request to JSON
//   2. Write to child's stdin (with newline delimiter)
//   3. Read response from child's stdout (newline-delimited)
//   4. Unmarshal response JSON
//
// Timeout: 60 seconds (some tools like web search can be slow)
func (w *StdioWorker) sendRequest(req registry.JSONRPCRequest) (*registry.JSONRPCResponse, error) {
	// -------------------------------------------------------------------------
	// Write the request to the child's stdin
	// -------------------------------------------------------------------------
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	// MCP uses newline-delimited JSON
	reqBytes = append(reqBytes, '\n')

	startTime := time.Now()
	if _, err := w.stdin.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	logger.AddLog("INFO", fmt.Sprintf("[%s] Sent request %v (%s), waiting for response...", w.command, req.ID, req.Method))

	// -------------------------------------------------------------------------
	// Read the response from the child's stdout (with timeout)
	// -------------------------------------------------------------------------
	// We use channels and a goroutine to implement the timeout because
	// bufio.Reader.ReadBytes() is blocking.
	responseChan := make(chan *registry.JSONRPCResponse, 1)
	errorChan := make(chan error, 1)

	go func() {
		// Read until newline (JSON-RPC response delimiter)
		line, err := w.stdout.ReadBytes('\n')
		if err != nil {
			errorChan <- err
			return
		}

		// Parse the JSON response
		var resp registry.JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			errorChan <- fmt.Errorf("failed to parse response: %w", err)
			return
		}
		responseChan <- &resp
	}()

	// Wait for response, error, timeout, or context cancellation
	select {
	case resp := <-responseChan:
		duration := time.Since(startTime)
		logger.AddLog("INFO", fmt.Sprintf("[%s] Received response for %v in %v", w.command, req.ID, duration))
		return resp, nil

	case err := <-errorChan:
		duration := time.Since(startTime)
		logger.AddLog("ERROR", fmt.Sprintf("[%s] Error reading response for %v after %v: %v", w.command, req.ID, duration, err))
		return nil, err

	case <-time.After(60 * time.Second):
		duration := time.Since(startTime)
		logger.AddLog("ERROR", fmt.Sprintf("[%s] Timeout waiting for response for %v (%s) after %v", w.command, req.ID, req.Method, duration))
		return nil, fmt.Errorf("timeout waiting for response after %v", duration)

	case <-w.ctx.Done():
		// Context was cancelled (e.g., application shutdown)
		return nil, w.ctx.Err()
	}
}

// sendNotification sends a JSON-RPC notification (no response expected).
// Notifications are requests without an ID field.
func (w *StdioWorker) sendNotification(req registry.JSONRPCRequest) error {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}
	reqBytes = append(reqBytes, '\n')

	_, err = w.stdin.Write(reqBytes)
	return err
}

// nextID returns the next JSON-RPC request ID.
// IDs are auto-incrementing integers starting at 1.
// NOT thread-safe - caller must hold the mutex or be in a single-threaded context.
func (w *StdioWorker) nextID() int64 {
	w.requestID++
	return w.requestID
}

// =============================================================================
// Lifecycle Methods
// =============================================================================

// IsRunning returns whether the server process is running and initialized.
// Thread-safe.
func (w *StdioWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.initialized && w.cmd != nil && w.cmd.Process != nil
}

// Close gracefully shuts down the MCP server process.
// It first tries SIGINT, then force-kills after 2 seconds.
func (w *StdioWorker) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Mark as not initialized to prevent new requests
	w.initialized = false

	// Close stdin to signal EOF to the child
	if w.stdin != nil {
		w.stdin.Close()
	}

	// Graceful shutdown with timeout
	if w.cmd != nil && w.cmd.Process != nil {
		// Try graceful shutdown first (SIGINT)
		w.cmd.Process.Signal(os.Interrupt)

		// Wait for process to exit with timeout
		done := make(chan error, 1)
		go func() {
			done <- w.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(2 * time.Second):
			// Force kill if it didn't exit in time
			w.cmd.Process.Kill()
		}
	}

	return nil
}
