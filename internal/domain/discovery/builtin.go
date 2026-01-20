package discovery

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/registry"
)

// PrimordialTools returns the definitions for built-in MCP tools.
// These are the "meta-layer" tools that are always available to AI clients.
// External tools (like brave-search) are NOT exposed until explicitly activated via scooter_add.
func PrimordialTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "scooter_find",
			Title:       "Search Registry",
			Description: "Searches the Local Registry and Community Catalog for MCP tools. Use this to discover available tools before adding them.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_find",
					Description: "Searches the Local Registry and Community Catalog for MCP tools. Returns tool names, descriptions, and available sub-tools. Use this to discover what tools can be added to your session.",
					InputSchema: &registry.JSONSchema{
						Type: "object",
						Properties: map[string]registry.PropertySchema{
							"query": {
								Type:        "string",
								Description: "Search query to find tools (e.g., 'search', 'database', 'github'). Leave empty to list all available tools.",
							},
						},
					},
				},
			},
		},
		{
			Name:        "scooter_add",
			Title:       "Enable Tool",
			Description: "Installs and enables an MCP tool for the current session. After adding, the tool's functions become available.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_add",
					Description: "Activates an MCP tool server for the current session. Once added, the tool's functions become available for use. Use scooter_find first to discover available tools.",
					InputSchema: &registry.JSONSchema{
						Type: "object",
						Properties: map[string]registry.PropertySchema{
							"tool_name": {
								Type:        "string",
								Description: "The name of the tool/server to add (e.g., 'brave-search', 'github'). Use the server name, not individual function names.",
							},
						},
						Required: []string{"tool_name"},
					},
				},
			},
		},
		{
			Name:        "scooter_remove",
			Title:       "Disable Tool",
			Description: "Unloads an MCP tool to free up context window space and system resources.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_remove",
					Description: "Deactivates an MCP tool server, removing its functions from the session. Use this to free up resources when a tool is no longer needed.",
					InputSchema: &registry.JSONSchema{
						Type: "object",
						Properties: map[string]registry.PropertySchema{
							"tool_name": {
								Type:        "string",
								Description: "The name of the tool/server to remove (e.g., 'brave-search'). Use scooter_list_active to see currently active tools.",
							},
						},
						Required: []string{"tool_name"},
					},
				},
			},
		},
		{
			Name:        "scooter_list_active",
			Title:       "List Active",
			Description: "Returns a list of currently active tool servers and their available functions.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_list_active",
					Description: "Lists all currently active MCP tool servers and the functions they provide. Use this to see what tools are available in your current session.",
					InputSchema: &registry.JSONSchema{
						Type:       "object",
						Properties: map[string]registry.PropertySchema{},
					},
				},
			},
		},
		{
			Name:        "scooter_code_interpreter",
			Title:       "Code Interpreter",
			Description: "Executes sandboxed JavaScript code. Can chain other tools using callTool(name, args).",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_code_interpreter",
					Description: "Executes JavaScript code in a secure sandbox. Use 'callTool(name, args)' within the script to chain other active tools. Results are returned without flooding the context window.",
					InputSchema: &registry.JSONSchema{
						Type: "object",
						Properties: map[string]registry.PropertySchema{
							"script": {
								Type:        "string",
								Description: "JavaScript code to execute. Use callTool('tool_name', {args}) to invoke other tools.",
							},
							"arguments": {
								Type:        "object",
								Description: "Optional arguments to pass to the script, accessible as 'args' variable.",
							},
						},
						Required: []string{"script"},
					},
				},
			},
		},
		{
			Name:        "scooter_filesystem",
			Title:       "Filesystem",
			Description: "Safe file operations with strict path scoping. Supports read, write, list, delete, and exists.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_filesystem",
					Description: "Performs safe file system operations with path validation. Supports read, write, list, delete, and exists operations.",
					InputSchema: &registry.JSONSchema{
						Type: "object",
						Properties: map[string]registry.PropertySchema{
							"operation": {
								Type:        "string",
								Description: "The operation to perform: 'read', 'write', 'list', 'delete', or 'exists'.",
								Enum:        []string{"read", "write", "list", "delete", "exists"},
							},
							"path": {
								Type:        "string",
								Description: "The file or directory path to operate on.",
							},
							"content": {
								Type:        "string",
								Description: "Content to write (required for 'write' operation).",
							},
						},
						Required: []string{"operation", "path"},
					},
				},
			},
		},
		{
			Name:        "scooter_fetch",
			Title:       "HTTP Fetch",
			Description: "Makes HTTP requests to retrieve web content. Supports GET, POST, and custom headers.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_fetch",
					Description: "Makes HTTP requests to retrieve web content. Supports various methods and custom headers.",
					InputSchema: &registry.JSONSchema{
						Type: "object",
						Properties: map[string]registry.PropertySchema{
							"url": {
								Type:        "string",
								Description: "The URL to fetch.",
							},
							"method": {
								Type:        "string",
								Description: "HTTP method (GET, POST, PUT, DELETE, etc.). Defaults to GET.",
								Enum:        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
							},
							"headers": {
								Type:        "object",
								Description: "Optional HTTP headers as key-value pairs.",
							},
							"body": {
								Type:        "string",
								Description: "Request body for POST/PUT requests.",
							},
						},
						Required: []string{"url"},
					},
				},
			},
		},
	}
}

// HandleBuiltinTool handles calls to the primordial tools.
func (e *DiscoveryEngine) HandleBuiltinTool(name string, params map[string]interface{}) (interface{}, error) {
	e.mu.RLock()
	isDisabled := e.disabledTools[name]
	e.mu.RUnlock()

	if isDisabled {
		return nil, fmt.Errorf("tool is disabled: %s", name)
	}

	switch name {
	case "scooter_find":
		query, _ := params["query"].(string)
		results := e.Find(query)
		
		// Format results to show available tools for each server
		formatted := make([]map[string]interface{}, 0, len(results))
		for _, td := range results {
			if td.Source == "builtin" {
				continue // Skip builtins in find results - they're always available
			}
			
			toolNames := make([]string, 0, len(td.Tools))
			for _, t := range td.Tools {
				toolNames = append(toolNames, t.Name)
			}
			
			formatted = append(formatted, map[string]interface{}{
				"name":        td.Name,
				"title":       td.Title,
				"description": td.Description,
				"category":    td.Category,
				"tools":       toolNames,
				"source":      td.Source,
			})
		}
		return formatted, nil
		
	case "scooter_add":
		tool, ok := params["tool_name"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_name is required")
		}
		
		// Check if tool is already active
		activeServers := e.ListActive()
		alreadyActive := false
		for _, s := range activeServers {
			if s == tool {
				alreadyActive = true
				break
			}
		}

		if alreadyActive {
			// Get the tools that are already available
			availableTools := e.GetActiveToolsForServer(tool)
			toolNames := make([]string, 0, len(availableTools))
			for _, t := range availableTools {
				toolNames = append(toolNames, t.Name)
			}
			return map[string]interface{}{
				"status":          "already_active",
				"server":          tool,
				"available_tools": toolNames,
				"message":         fmt.Sprintf("Server '%s' is already active. Available tools: %v", tool, toolNames),
			}, nil
		}

		err := e.Add(tool)
		if err != nil {
			return nil, err
		}
		
		// Get the tools that are now available from this server
		availableTools := e.GetActiveToolsForServer(tool)
		toolNames := make([]string, 0, len(availableTools))
		for _, t := range availableTools {
			toolNames = append(toolNames, t.Name)
		}
		
		return map[string]interface{}{
			"status":          "activated",
			"server":          tool,
			"available_tools": toolNames,
			"message":         fmt.Sprintf("Server '%s' is now active. You can now use: %v", tool, toolNames),
		}, nil
		
	case "scooter_remove":
		tool, ok := params["tool_name"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_name is required")
		}
		
		// Get tools before removal for the response
		removedTools := e.GetActiveToolsForServer(tool)
		toolNames := make([]string, 0, len(removedTools))
		for _, t := range removedTools {
			toolNames = append(toolNames, t.Name)
		}
		
		err := e.Remove(tool)
		if err != nil {
			return nil, err
		}
		
		return map[string]interface{}{
			"status":        "deactivated",
			"server":        tool,
			"removed_tools": toolNames,
			"message":       fmt.Sprintf("Server '%s' has been deactivated. Tools no longer available: %v", tool, toolNames),
		}, nil
		
	case "scooter_list_active":
		activeServers := e.ListActive()
		
		// Build detailed response with tools per server
		servers := make([]map[string]interface{}, 0, len(activeServers))
		for _, serverName := range activeServers {
			serverTools := e.GetActiveToolsForServer(serverName)
			toolNames := make([]string, 0, len(serverTools))
			for _, t := range serverTools {
				toolNames = append(toolNames, t.Name)
			}
			
			servers = append(servers, map[string]interface{}{
				"name":  serverName,
				"tools": toolNames,
			})
		}
		
		return map[string]interface{}{
			"active_servers": servers,
			"count":          len(activeServers),
		}, nil
	case "scooter_code_interpreter":
		script, _ := params["script"].(string)
		args, _ := params["arguments"].(map[string]interface{})
		interpreter := NewCodeInterpreter(e.CallTool)
		return interpreter.Execute(script, args)
	case "scooter_filesystem":
		return handleFilesystem(params)
	case "scooter_fetch":
		return handleFetch(params)
	default:
		return nil, fmt.Errorf("unknown builtin tool: %s", name)
	}
}

// handleFilesystem implements safe file operations with path scoping.
func handleFilesystem(params map[string]interface{}) (interface{}, error) {
	operation, _ := params["operation"].(string)
	path, _ := params["path"].(string)

	if operation == "" {
		return nil, fmt.Errorf("operation is required (read, write, list, delete, exists)")
	}
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}

	// Security: Normalize and validate path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Security: Scope to User Home Directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}
	
	// Ensure homeDir is absolute for comparison
	absHome, err := filepath.Abs(homeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve home directory: %w", err)
	}

	// Calculate relative path to check if it's inside home
	rel, err := filepath.Rel(absHome, absPath)
	if err != nil {
		// This usually happens on Windows if paths are on different drives
		return nil, fmt.Errorf("access denied: path must be on the same drive as home directory")
	}

	// If relative path starts with ".." or is absolute (on different drive on unix), reject
	if strings.HasPrefix(rel, "..") || rel == ".." || filepath.IsAbs(rel) {
		return nil, fmt.Errorf("access denied: path must be within user home directory (%s)", absHome)
	}

	switch operation {
	case "read":
		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		return map[string]interface{}{
			"content": string(content),
			"path":    absPath,
			"size":    len(content),
		}, nil

	case "write":
		content, ok := params["content"].(string)
		if !ok {
			return nil, fmt.Errorf("content is required for write operation")
		}
		
		// Create parent directories if needed
		dir := filepath.Dir(absPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}
		return map[string]interface{}{
			"status":  "written",
			"path":    absPath,
			"size":    len(content),
		}, nil

	case "list":
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to list directory: %w", err)
		}
		
		files := make([]map[string]interface{}, 0, len(entries))
		for _, entry := range entries {
			info, _ := entry.Info()
			fileInfo := map[string]interface{}{
				"name":  entry.Name(),
				"isDir": entry.IsDir(),
			}
			if info != nil {
				fileInfo["size"] = info.Size()
				fileInfo["modified"] = info.ModTime().Format(time.RFC3339)
			}
			files = append(files, fileInfo)
		}
		return map[string]interface{}{
			"path":  absPath,
			"files": files,
			"count": len(files),
		}, nil

	case "delete":
		if err := os.Remove(absPath); err != nil {
			return nil, fmt.Errorf("failed to delete: %w", err)
		}
		return map[string]interface{}{
			"status": "deleted",
			"path":   absPath,
		}, nil

	case "exists":
		_, err := os.Stat(absPath)
		exists := err == nil
		return map[string]interface{}{
			"exists": exists,
			"path":   absPath,
		}, nil

	default:
		return nil, fmt.Errorf("unknown operation: %s (supported: read, write, list, delete, exists)", operation)
	}
}

// handleFetch implements HTTP request capabilities.
func handleFetch(params map[string]interface{}) (interface{}, error) {
	url, _ := params["url"].(string)
	method, _ := params["method"].(string)
	body, _ := params["body"].(string)
	headersRaw, _ := params["headers"].(map[string]interface{})

	if url == "" {
		return nil, fmt.Errorf("url is required")
	}
	if method == "" {
		method = "GET"
	}

	// Build request
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, val := range headersRaw {
		if strVal, ok := val.(string); ok {
			req.Header.Set(key, strVal)
		}
	}

	// Set default User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "MCP-Scooter/1.0")
	}

	// Execute request with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body (limit to 10MB)
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Build response headers map
	respHeaders := make(map[string]string)
	for key := range resp.Header {
		respHeaders[key] = resp.Header.Get(key)
	}

	return map[string]interface{}{
		"status":      resp.StatusCode,
		"statusText":  resp.Status,
		"headers":     respHeaders,
		"body":        string(respBody),
		"contentType": resp.Header.Get("Content-Type"),
		"size":        len(respBody),
	}, nil
}
