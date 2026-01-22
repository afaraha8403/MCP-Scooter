package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/integration"
	"github.com/mcp-scooter/scooter/internal/domain/profile"
	"github.com/mcp-scooter/scooter/internal/domain/registry"
	"github.com/mcp-scooter/scooter/internal/logger"
)

// ToolWorker defines the interface for executing MCP tools.
type ToolWorker interface {
	Execute(stdin io.Reader, stdout io.Writer, env map[string]string) error
	Close() error
}

// PersistentWorker extends ToolWorker for servers that stay running.
type PersistentWorker interface {
	ToolWorker
	Start(env map[string]string) error
	CallTool(name string, arguments map[string]interface{}) (*registry.JSONRPCResponse, error)
	IsRunning() bool
	GetTools() []registry.Tool
	RefreshTools() error
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
	IconBackground *registry.IconBackground `json:"icon_background,omitempty"`
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

// CleanupCallback is called when a tool is auto-unloaded due to inactivity.
type CleanupCallback func(serverName string)

// DiscoveryEngine manages tools for an MCP session.
type DiscoveryEngine struct {
	mu              sync.RWMutex
	activeServers   map[string]ToolWorker // name -> worker
	toolToServer    map[string]string     // toolName -> serverName
	lastUsed        map[string]time.Time
	registry        []ToolDefinition
	wasmDir         string
	registryDir     string
	env             map[string]string
	disabledTools   map[string]bool
	ctx             context.Context
	credentials     *integration.CredentialManager
	cleanupCallback CleanupCallback
	settings        profile.Settings // AI routing configuration
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
		disabledTools: make(map[string]bool),
		ctx:           ctx,
			credentials:   integration.NewCredentialManager(),
	}
	e.loadRegistry()
	go e.monitor()
	return e
}

// SetCleanupCallback sets the callback function called when tools are auto-unloaded.
func (e *DiscoveryEngine) SetCleanupCallback(cb CleanupCallback) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cleanupCallback = cb
}

// GetWorker returns the worker for a given server name.
func (e *DiscoveryEngine) GetWorker(serverName string) (ToolWorker, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	worker, ok := e.activeServers[serverName]
	return worker, ok
}

// GetActiveToolsForServer returns the tools provided by an active server.
func (e *DiscoveryEngine) GetActiveToolsForServer(serverName string) []registry.Tool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	worker, ok := e.activeServers[serverName]
	if !ok {
		return nil
	}

	if pw, ok := worker.(PersistentWorker); ok {
		return pw.GetTools()
	}

	// Fall back to registry-defined tools for non-persistent workers
	for _, td := range e.registry {
		if td.Name == serverName {
			return td.Tools
		}
	}
	return nil
}

// GetCredentialManager returns the credential manager for external access.
func (e *DiscoveryEngine) GetCredentialManager() *integration.CredentialManager {
	return e.credentials
}

// SetEnv updates the environment variables for tools.
func (e *DiscoveryEngine) SetEnv(env map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.env = env
}

// SetDisabledTools updates the list of disabled system tools.
func (e *DiscoveryEngine) SetDisabledTools(disabled []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.disabledTools = make(map[string]bool)
	for _, tool := range disabled {
		e.disabledTools[tool] = true
	}
}

// IsToolDisabled checks if a tool is in the disabled list.
func (e *DiscoveryEngine) IsToolDisabled(name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.disabledTools[name]
}

// SetSettings updates the AI routing configuration.
func (e *DiscoveryEngine) SetSettings(settings profile.Settings) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.settings = settings
}

func (e *DiscoveryEngine) loadRegistry() {
	if e.registryDir == "" {
		return
	}

	// Scan official and custom subdirectories
	subdirs := []string{"official", "custom"}
	for _, subdir := range subdirs {
		dirPath := filepath.Join(e.registryDir, subdir)
		files, err := os.ReadDir(dirPath)
		if err != nil {
			// Skip missing directories silently or log if not just "not found"
			continue
		}

		for _, file := range files {
			if filepath.Ext(file.Name()) == ".json" {
				data, err := os.ReadFile(filepath.Join(dirPath, file.Name()))
				if err != nil {
					fmt.Printf("Warning: failed to read tool definition %s/%s: %v\n", subdir, file.Name(), err)
					continue
				}

				// Use the full MCPEntry from registry package for thoroughness
				var entry registry.MCPEntry
				if err := json.Unmarshal(data, &entry); err != nil {
					fmt.Printf("Warning: failed to parse tool definition %s/%s: %v\n", subdir, file.Name(), err)
					continue
				}

				source := string(entry.Source)
				if source == "" {
					if subdir == "official" {
						source = "official"
					} else {
						source = "custom"
					}
				}

				td := ToolDefinition{
					Name:          entry.Name,
					Title:         entry.Title,
					Version:       entry.Version,
					Description:   entry.Description,
					Category:      string(entry.Category),
					Source:        source,
					Icon:          entry.Icon,
					IconBackground: entry.IconBackground,
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
}

// ReloadRegistry reloads the tool definitions from disk and refreshes running servers.
// This is useful when tool definitions are updated at runtime.
func (e *DiscoveryEngine) ReloadRegistry() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	logger.AddLog("INFO", "[Discovery] Reloading tool registry from disk...")
	e.loadRegistry()

	// Count and log loaded tools
	officialCount := 0
	customCount := 0
	for _, td := range e.registry {
		if td.Source == "official" {
			officialCount++
		} else if td.Source == "custom" {
			customCount++
		}
	}
	logger.AddLog("INFO", fmt.Sprintf("[Discovery] Registry reloaded: %d official tools, %d custom tools", officialCount, customCount))

	// Refresh tools from running persistent servers (e.g., stdio MCP servers)
	refreshedServers := 0
	failedServers := 0
	for serverName, worker := range e.activeServers {
		if persistentWorker, ok := worker.(PersistentWorker); ok && persistentWorker.IsRunning() {
			logger.AddLog("INFO", fmt.Sprintf("[Discovery] Refreshing tools from running server '%s'", serverName))

			// Refresh tools from the server
			if err := persistentWorker.RefreshTools(); err != nil {
				logger.AddLog("WARN", fmt.Sprintf("[Discovery] Failed to refresh tools from server '%s': %v (keeping cached tools)", serverName, err))
				failedServers++
				// Don't fail the entire refresh - continue with other servers
				continue
			}

			// Get fresh tools from the running server
			serverTools := persistentWorker.GetTools()
			if len(serverTools) > 0 {
				// Remove old tool mappings for this server
				for toolName, mappedServer := range e.toolToServer {
					if mappedServer == serverName {
						delete(e.toolToServer, toolName)
					}
				}

				// Add fresh tool mappings
				for _, tool := range serverTools {
					// Normalize tool name: convert dashes to underscores to match registry definition format
					normalizedName := strings.ReplaceAll(tool.Name, "-", "_")
					logger.AddLog("INFO", fmt.Sprintf("[Discovery] Remapping tool '%s' (normalized: '%s') -> server '%s'", tool.Name, normalizedName, serverName))
					e.toolToServer[normalizedName] = serverName
					// Also map with original name for compatibility
					e.toolToServer[tool.Name] = serverName
				}
				logger.AddLog("INFO", fmt.Sprintf("[Discovery] Server '%s' now provides %d tools", serverName, len(serverTools)))
				refreshedServers++
			}
		}
	}

	// Log summary of server refresh results
	if failedServers > 0 {
		logger.AddLog("WARN", fmt.Sprintf("[Discovery] Failed to refresh %d server(s) (keeping cached tools)", failedServers))
	}

	if refreshedServers > 0 {
		logger.AddLog("INFO", fmt.Sprintf("[Discovery] Refreshed tools from %d running server(s)", refreshedServers))
	}

	return nil
}

// Find searches for tools in the registry.
func (e *DiscoveryEngine) Find(query string) []ToolDefinition {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return all tools for management purposes
	return e.registry
}

// ListTools returns tools available for the AI agent (filtering out disabled ones).
func (e *DiscoveryEngine) ListTools() []ToolDefinition {
	e.mu.RLock()
	defer e.mu.RUnlock()

	filtered := make([]ToolDefinition, 0, len(e.registry))
	for _, td := range e.registry {
		if !e.disabledTools[td.Name] {
			filtered = append(filtered, td)
		}
	}
	return filtered
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
	for i := range e.registry {
		if e.registry[i].Name == serverName {
			targetDef = &e.registry[i]
			break
		}
	}
	if targetDef == nil {
		return fmt.Errorf("server not found in registry: %s", serverName)
	}

	// Build environment with credentials from keychain
	toolEnv := make(map[string]string)
	
	// Start with profile env
	for k, v := range e.env {
		toolEnv[k] = v
	}
	
	// Layer in secure credentials from keychain
	if e.credentials != nil && targetDef.Authorization != nil {
		creds, err := e.credentials.GetCredentialsForTool(serverName, targetDef.Authorization)
		if err != nil {
			fmt.Printf("[Discovery] Warning: failed to get credentials for %s: %v\n", serverName, err)
		} else {
			for k, v := range creds {
				toolEnv[k] = v
				fmt.Printf("[Discovery] Injected credential %s for %s\n", k, serverName)
			}
		}
	}

	var worker ToolWorker
	// Handle Stdio transport (e.g., npx, python, etc.)
	if targetDef.Runtime != nil && targetDef.Runtime.Transport == registry.TransportStdio {
		stdioWorker := NewStdioWorker(e.ctx, targetDef.Runtime.Command, targetDef.Runtime.Args)
		
		// Start the persistent server process with initialize handshake
		if err := stdioWorker.Start(toolEnv); err != nil {
			return fmt.Errorf("failed to start MCP server %s: %w", serverName, err)
		}
		
		// Update tool mappings from server's actual tools if available
		serverTools := stdioWorker.GetTools()
		if len(serverTools) > 0 {
			fmt.Printf("[Discovery] Server %s reports %d tools\n", serverName, len(serverTools))
			for _, tool := range serverTools {
				// Normalize tool name: convert dashes to underscores to match registry definition format
				normalizedName := strings.ReplaceAll(tool.Name, "-", "_")
				fmt.Printf("[Discovery] Mapping tool '%s' (normalized: '%s') -> server '%s'\n", tool.Name, normalizedName, serverName)
				e.toolToServer[normalizedName] = serverName
				// Also map with original name for compatibility
				e.toolToServer[tool.Name] = serverName
			}
		} else {
			// Fall back to registry-defined tools
			for _, tool := range targetDef.Tools {
				fmt.Printf("[Discovery] Mapping registry tool '%s' -> server '%s'\n", tool.Name, serverName)
				e.toolToServer[tool.Name] = serverName
			}
		}
		
		worker = stdioWorker
	} else {
		// Default to WASM
		wasmWorker := NewWASMWorker(e.ctx)
		wasmPath := filepath.Join(e.wasmDir, fmt.Sprintf("%s.wasm", serverName))
		if err := wasmWorker.Load(wasmPath); err != nil {
			return fmt.Errorf("failed to load wasm tool %s: %w", serverName, err)
		}
		worker = wasmWorker
		
	// Use registry-defined tools for WASM
	for _, tool := range targetDef.Tools {
		fmt.Printf("[Discovery] Mapping WASM tool '%s' -> server '%s'\n", tool.Name, serverName)
		e.toolToServer[tool.Name] = serverName
	}
}

	e.activeServers[serverName] = worker
	fmt.Printf("[Discovery] Activated server: %s\n", serverName)
	fmt.Printf("[Discovery] Current toolToServer mappings: %v\n", e.toolToServer)
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
	// Normalize incoming tool name (convert underscores to dashes) for lookup
	normalizedName := strings.ReplaceAll(name, "_", "-")
	e.mu.RLock()
	serverName, hasMapping := e.toolToServer[normalizedName]
	worker, active := e.activeServers[serverName]
	e.mu.RUnlock()

	// Debug logging
	if !hasMapping {
		fmt.Printf("[Discovery] Tool '%s' (normalized: '%s') not found in toolToServer mapping\n", name, normalizedName)
		fmt.Printf("[Discovery] Current mappings: %v\n", e.toolToServer)
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	if !active {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	if active {
		e.MarkUsed(serverName)
		startTime := time.Now()

		// Check if this is a persistent worker (StdioWorker)
		if persistentWorker, ok := worker.(PersistentWorker); ok {
			// Use the direct CallTool method for persistent workers
			resp, err := persistentWorker.CallTool(name, params)
			duration := time.Since(startTime)

			if err != nil {
				fmt.Printf("[Discovery] Tool execution failed for '%s': %v\n", name, err)
				return nil, fmt.Errorf("tool execution failed: %w", err)
			}

			if resp.Error != nil {
				return nil, fmt.Errorf("tool error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
			}

			fmt.Printf("[Discovery] Tool '%s' executed successfully in %v\n", name, duration)
			return resp.Result, nil
		}

		// Fall back to Execute pattern for WASM workers
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
			fmt.Printf("[Discovery] Tool execution failed for '%s': %v\n", name, err)
			return nil, fmt.Errorf("tool execution failed: %w", err)
		}

		duration := time.Since(startTime)
		fmt.Printf("[Discovery] Tool '%s' response received in %v: %s\n", name, duration, stdout.String())

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
	
	threshold := 10 * time.Minute // Auto-unload after 10 mins
	now := time.Now()
	
	var unloadedServers []string

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
				
				unloadedServers = append(unloadedServers, name)
			}
		}
	}
	
	callback := e.cleanupCallback
	e.mu.Unlock()
	
	// Call cleanup callback outside of lock to avoid deadlocks
	if callback != nil {
		for _, name := range unloadedServers {
			callback(name)
		}
	}
}
