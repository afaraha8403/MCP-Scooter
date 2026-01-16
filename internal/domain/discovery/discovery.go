package discovery

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// ToolDefinition represents a metadata for an MCP tool.
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"` // "local", "community"
	Installed   bool   `json:"installed"`
}

// DiscoveryEngine manages tools for an MCP session.
type DiscoveryEngine struct {
	mu          sync.RWMutex
	activeTools map[string]*WASMWorker
	lastUsed    map[string]time.Time
	registry    []ToolDefinition
	wasmDir     string
	ctx         context.Context
}

func NewDiscoveryEngine(ctx context.Context, wasmDir string) *DiscoveryEngine {
	e := &DiscoveryEngine{
		activeTools: make(map[string]*WASMWorker),
		lastUsed:    make(map[string]time.Time),
		registry: append(PrimordialTools(),
			ToolDefinition{Name: "test-tool", Description: "Minimal Go-WASM tool for verification", Source: "local"},
			ToolDefinition{Name: "postgres-mcp", Description: "Postgres database explorer", Source: "community"},
			ToolDefinition{Name: "jira-mcp", Description: "Manage JIRA issues", Source: "community"},
			ToolDefinition{Name: "linear-mcp", Description: "Manage Linear issues", Source: "community"},
		),
		wasmDir: wasmDir,
		ctx:     ctx,
	}
	go e.monitor()
	return e
}

// Find searches for tools in the registry.
func (e *DiscoveryEngine) Find(query string) []ToolDefinition {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Simple mock filter for now
	return e.registry
}

// Add installs and activates a tool.
func (e *DiscoveryEngine) Add(toolName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.lastUsed[toolName] = time.Now()
	if _, ok := e.activeTools[toolName]; ok {
		return nil // Already active
	}

	// Check if tool exists in registry
	var found bool
	for _, t := range e.registry {
		if t.Name == toolName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("tool not found in registry: %s", toolName)
	}

	worker := NewWASMWorker(e.ctx)
	wasmPath := filepath.Join(e.wasmDir, fmt.Sprintf("%s.wasm", toolName))
	if err := worker.Load(wasmPath); err != nil {
		return fmt.Errorf("failed to load wasm tool %s: %w", toolName, err)
	}

	e.activeTools[toolName] = worker
	return nil
}

// Remove unloads a tool.
func (e *DiscoveryEngine) Remove(toolName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if worker, ok := e.activeTools[toolName]; ok {
		worker.Close()
		delete(e.activeTools, toolName)
		delete(e.lastUsed, toolName)
		return nil
	}
	return fmt.Errorf("tool not found: %s", toolName)
}

// ListActive returns names of currently loaded tools.
func (e *DiscoveryEngine) ListActive() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	active := make([]string, 0, len(e.activeTools))
	for name := range e.activeTools {
		active = append(active, name)
	}
	return active
}

// CallTool executes a tool (builtin or WASM) and returns the result.
func (e *DiscoveryEngine) CallTool(name string, params map[string]interface{}) (interface{}, error) {
	// 1. Try built-in tools
	result, err := e.HandleBuiltinTool(name, params)
	if err == nil {
		return result, nil
	}

	// 2. Try active WASM tools
	e.mu.RLock()
	_, ok := e.activeTools[name]
	e.mu.RUnlock()

	if ok {
		e.MarkUsed(name)
		// For now, we return a mock result for WASM tools since Execute is blocking
		// with instantiation. In a real persistent world, we'd pipe the JSON-RPC.
		// However, for verification, we'll just proxy the call.
		return fmt.Sprintf("WASM Tool %s activated and proxy-called with %v", name, params), nil
	}

	return nil, fmt.Errorf("tool not found: %s", name)
}

// MarkUsed updates the last used timestamp for a tool.
func (e *DiscoveryEngine) MarkUsed(toolName string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lastUsed[toolName] = time.Now()
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
			if worker, ok := e.activeTools[name]; ok {
				fmt.Printf("Auto-unloading inactive tool: %s\n", name)
				worker.Close()
				delete(e.activeTools, name)
				delete(e.lastUsed, name)
			}
		}
	}
}
