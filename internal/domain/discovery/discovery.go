package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/registry"
)

// ToolWorker defines the interface for executing MCP tools.
type ToolWorker interface {
	Execute(stdin io.Reader, stdout io.Writer, env map[string]string) error
	Close() error
}

// ToolDefinition represents a metadata for an MCP tool.
type ToolDefinition struct {
	Name          string                 `json:"name"`
	Title         string                 `json:"title,omitempty"`
	Version       string                 `json:"version,omitempty"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category"`
	Source        string                 `json:"source"` // "local", "community"
	Installed     bool                   `json:"installed"`
	Icon          string                 `json:"icon,omitempty"`
	About         string                 `json:"about,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Homepage      string                 `json:"homepage,omitempty"`
	Repository    string                 `json:"repository,omitempty"`
	Documentation string                 `json:"documentation,omitempty"`
	Authorization *registry.Authorization `json:"authorization,omitempty"`
	Runtime       *registry.Runtime      `json:"runtime,omitempty"`
	Tools         []registry.Tool        `json:"tools,omitempty"`
	Package       *registry.Package      `json:"package,omitempty"`
	Metadata      *registry.Metadata     `json:"metadata,omitempty"`
}

// DiscoveryEngine manages tools for an MCP session.
type DiscoveryEngine struct {
	mu            sync.RWMutex
	activeServers map[string]ToolWorker // name -> worker
	toolToServer  map[string]string     // toolName -> serverName
	lastUsed      map[string]time.Time
	registry      []ToolDefinition
	wasmDir       string
	registryDir   string
	env           map[string]string
	ctx           context.Context
}

func NewDiscoveryEngine(ctx context.Context, wasmDir string, registryDir string) *DiscoveryEngine {
	e := &DiscoveryEngine{
		activeServers: make(map[string]ToolWorker),
		toolToServer:  make(map[string]string),
		lastUsed:      make(map[string]time.Time),
		registry:      PrimordialTools(),
		wasmDir:       wasmDir,
		registryDir:   registryDir,
		env:           make(map[string]string),
		ctx:           ctx,
	}
	e.loadRegistry()
	go e.monitor()
	return e
}

// SetEnv updates the environment variables for tools.
func (e *DiscoveryEngine) SetEnv(env map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.env = env
}

func (e *DiscoveryEngine) loadRegistry() {
	if e.registryDir == "" {
		return
	}

	files, err := os.ReadDir(e.registryDir)
	if err != nil {
		fmt.Printf("Warning: failed to read registry directory %s: %v\n", e.registryDir, err)
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(e.registryDir, file.Name()))
			if err != nil {
				fmt.Printf("Warning: failed to read tool definition %s: %v\n", file.Name(), err)
				continue
			}

			// Use the full MCPEntry from registry package for thoroughness
			var entry registry.MCPEntry
			if err := json.Unmarshal(data, &entry); err != nil {
				fmt.Printf("Warning: failed to parse tool definition %s: %v\n", file.Name(), err)
				continue
			}

			td := ToolDefinition{
				Name:          entry.Name,
				Title:         entry.Title,
				Version:       entry.Version,
				Description:   entry.Description,
				Category:      string(entry.Category),
				Source:        string(entry.Source),
				Icon:          entry.Icon,
				About:         entry.About,
				Tags:          entry.Tags,
				Homepage:      entry.Homepage,
				Repository:    entry.Repository,
				Documentation: entry.Docs,
				Authorization: entry.Auth,
				Runtime:       entry.Runtime,
				Tools:         entry.Tools,
				Package:       entry.Package,
				Metadata:      entry.Metadata,
			}
			e.Register(td)
		}
	}
}

// Find searches for tools in the registry.
func (e *DiscoveryEngine) Find(query string) []ToolDefinition {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Simple mock filter for now
	return e.registry
}

// Register adds a new tool definition to the registry.
func (e *DiscoveryEngine) Register(td ToolDefinition) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check for duplicates
	for i, existing := range e.registry {
		if existing.Name == td.Name {
			e.registry[i] = td
			return
		}
	}
	e.registry = append(e.registry, td)
}

// GetServerForTool finds which server provides the given tool name.
func (e *DiscoveryEngine) GetServerForTool(toolName string) (string, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 1. Check if it's already active
	if serverName, ok := e.toolToServer[toolName]; ok {
		return serverName, true
	}

	// 2. Check registry for the tool name
	for _, td := range e.registry {
		for _, t := range td.Tools {
			if t.Name == toolName {
				return td.Name, true
			}
		}
		// Also check if the server name itself is what's being called
		if td.Name == toolName {
			return td.Name, true
		}
	}

	return "", false
}

// Add installs and activates a tool.
func (e *DiscoveryEngine) Add(serverName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.lastUsed[serverName] = time.Now()
	if _, ok := e.activeServers[serverName]; ok {
		return nil // Already active
	}

	// Check if server exists in registry
	var targetDef *ToolDefinition
	for _, t := range e.registry {
		if t.Name == serverName {
			targetDef = &t
			break
		}
	}
	if targetDef == nil {
		return fmt.Errorf("server not found in registry: %s", serverName)
	}

	var worker ToolWorker
	// Handle Stdio transport (e.g., npx, python, etc.)
	if targetDef.Runtime != nil && targetDef.Runtime.Transport == registry.TransportStdio {
		worker = NewStdioWorker(e.ctx, targetDef.Runtime.Command, targetDef.Runtime.Args)
	} else {
		// Default to WASM
		wasmWorker := NewWASMWorker(e.ctx)
		wasmPath := filepath.Join(e.wasmDir, fmt.Sprintf("%s.wasm", serverName))
		if err := wasmWorker.Load(wasmPath); err != nil {
			return fmt.Errorf("failed to load wasm tool %s: %w", serverName, err)
		}
		worker = wasmWorker
	}

	e.activeServers[serverName] = worker
	for _, tool := range targetDef.Tools {
		e.toolToServer[tool.Name] = serverName
	}
	return nil
}

// Remove unloads a tool.
func (e *DiscoveryEngine) Remove(serverName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if worker, ok := e.activeServers[serverName]; ok {
		worker.Close()
		delete(e.activeServers, serverName)
		delete(e.lastUsed, serverName)

		// Remove tool mappings
		for toolName, sName := range e.toolToServer {
			if sName == serverName {
				delete(e.toolToServer, toolName)
			}
		}
		return nil
	}
	return fmt.Errorf("server not found: %s", serverName)
}

// ListActive returns names of currently loaded servers.
func (e *DiscoveryEngine) ListActive() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	active := make([]string, 0, len(e.activeServers))
	for name := range e.activeServers {
		active = append(active, name)
	}
	return active
}

// CallTool executes a tool (builtin or WASM/Stdio) and returns the result.
func (e *DiscoveryEngine) CallTool(name string, params map[string]interface{}) (interface{}, error) {
	// 1. Try built-in tools
	result, err := e.HandleBuiltinTool(name, params)
	if err == nil {
		return result, nil
	}

	// 2. Try active servers
	e.mu.RLock()
	serverName := e.toolToServer[name]
	worker, active := e.activeServers[serverName]
	e.mu.RUnlock()

	if active {
		e.MarkUsed(serverName)

		// Prepare JSON-RPC request for MCP 'tools/call'
		req := registry.JSONRPCRequest{
			JSONRPC: "2.0",
			ID:      time.Now().UnixNano(),
			Method:  "tools/call",
		}
		callParams := struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{
			Name:      name,
			Arguments: params,
		}
		req.Params, _ = json.Marshal(callParams)

		input, _ := json.Marshal(req)
		stdin := io.MultiReader(
			io.LimitReader(os.Stdin, 0), // empty
			io.NopCloser(bytes.NewReader(input)),
			io.NopCloser(bytes.NewReader([]byte("\n"))),
		)

		var stdout bytes.Buffer
		e.mu.RLock()
		currentEnv := e.env
		e.mu.RUnlock()

		if err := worker.Execute(stdin, &stdout, currentEnv); err != nil {
			return nil, fmt.Errorf("tool execution failed: %w", err)
		}

		var resp registry.JSONRPCResponse
		if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
			// Some servers might output extra logs before the JSON, 
			// but for this simple implementation we expect clean JSON.
			return stdout.String(), nil
		}

		if resp.Error != nil {
			return nil, fmt.Errorf("tool error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
		}

		return resp.Result, nil
	}

	return nil, fmt.Errorf("tool not found: %s", name)
}

// MarkUsed updates the last used timestamp for a tool.
func (e *DiscoveryEngine) MarkUsed(serverName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastUsed[serverName] = time.Now()
}

// monitor background cleanup for inactive tools.
func (e *DiscoveryEngine) monitor() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.cleanup()
		case <-e.ctx.Done():
			return
		}
	}
}

func (e *DiscoveryEngine) cleanup() {
	e.mu.Lock()
	defer e.mu.Unlock()

	threshold := 10 * time.Minute // Auto-unload after 10 mins
	now := time.Now()

	for name, lastUsed := range e.lastUsed {
		if now.Sub(lastUsed) > threshold {
			if worker, ok := e.activeServers[name]; ok {
				fmt.Printf("Auto-unloading inactive tool: %s\n", name)
				worker.Close()
				delete(e.activeServers, name)
				delete(e.lastUsed, name)

				// Remove tool mappings
				for toolName, sName := range e.toolToServer {
					if sName == name {
						delete(e.toolToServer, toolName)
					}
				}
			}
		}
	}
}
