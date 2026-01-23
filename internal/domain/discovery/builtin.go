package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/registry"
	"github.com/mcp-scooter/scooter/internal/logger"
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
		{
			Name:        "scooter_execute",
			Title:       "AI Semantic Dispatcher",
			Description: "AI-powered dispatcher for automatic tool routing. Use this when you are unsure which tool handles your intent or cannot route a tool call. This uses an internal LLM to match your intent to available tools.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_execute",
					Description: "Automatically routes your intent to appropriate tool using semantic understanding. Use when you are unsure which tool to call for your task.",
					InputSchema: &registry.JSONSchema{
						Type: "object",
						Properties: map[string]registry.PropertySchema{
							"intent": {
								Type:        "string",
								Description: "Your intent or task description in natural language (e.g., 'search the web for latest AI news', 'create a GitHub issue')",
							},
						},
						Required: []string{"intent"},
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
			
			// Ensure we don't return nil for toolNames to avoid serialization issues
			if toolNames == nil {
				toolNames = []string{}
			}
			
			// Extract simple strings for title and description to avoid complex objects
			title := string(td.Title)
			description := string(td.Description)
			
			// Final safety: ensure everything is a simple type
			entry := map[string]interface{}{
				"name":        string(td.Name),
				"title":       title,
				"description": description,
				"category":    string(td.Category),
				"tools":       toolNames,
				"source":      string(td.Source),
			}
			
			// Log for debugging
			logger.AddLog("DEBUG", fmt.Sprintf("scooter_find: adding tool %s", td.Name))
			
			formatted = append(formatted, entry)
		}
		
		// Return as a map with a key to be more standard
		return map[string]interface{}{
			"tools": formatted,
		}, nil
		
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
			"message":         fmt.Sprintf("Server '%s' is already active. Available tools: %v.\n\n--- ROUTING INSTRUCTION ---\nAll tools listed above are managed exclusively by the 'mcp-scooter' gateway. Even if their names (e.g., starting with '%s_') suggest they belong to another server, you MUST execute them through the 'mcp-scooter' connection. If direct execution fails, use the 'scooter_execute' tool.", tool, toolNames, strings.Split(tool, "-")[0]),
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
	case "scooter_execute":
		intent, ok := params["intent"].(string)
		if !ok || intent == "" {
			return nil, fmt.Errorf("intent is required")
		}
		return e.handleSemanticDispatch(intent)
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

// getAIRoutingCredentials retrieves AI routing credentials from keychain.
func (e *DiscoveryEngine) getAIRoutingCredentials() (provider, model, key string, isFallback bool) {
	// Try primary first
	primaryKey, err := e.credentials.GetCredential("mcp-scooter:ai_primary", "MCP_SCOOTER_PRIMARY_AI_KEY")
	if err == nil && e.settings.PrimaryAIProvider != "" {
		return e.settings.PrimaryAIProvider, e.settings.PrimaryAIModel, primaryKey, false
	}

	// Try fallback
	fallbackKey, err := e.credentials.GetCredential("mcp-scooter:ai_fallback", "MCP_SCOOTER_FALLBACK_AI_KEY")
	if err == nil && e.settings.FallbackAIProvider != "" {
		return e.settings.FallbackAIProvider, e.settings.FallbackAIModel, fallbackKey, true
	}

	return "", "", "", false
}

// callInternalAI calls the appropriate AI provider.
func (e *DiscoveryEngine) callInternalAI(provider, model, key, prompt string) (map[string]interface{}, error) {
	var response map[string]interface{}
	var err error

	switch provider {
	case "gemini":
		response, err = e.callGemini(model, key, prompt)
	case "openrouter":
		response, err = e.callOpenRouter(model, key, prompt)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", provider)
	}

	return response, err
}

// callGemini calls the Gemini API.
func (e *DiscoveryEngine) callGemini(model, key, prompt string) (map[string]interface{}, error) {
	// Build request payload for Gemini API
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, key)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Gemini API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Gemini API error (status %d): %s", resp.StatusCode, string(body))
	}

	var geminiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode Gemini response: %w", err)
	}

	return geminiResp, nil
}

// callOpenRouter calls the OpenRouter API (OpenAI-compatible).
func (e *DiscoveryEngine) callOpenRouter(model, key, prompt string) (map[string]interface{}, error) {
	// Build request payload for OpenAI-compatible API
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{"role": "system", "content": "You are a JSON-only tool router. Return only valid JSON objects."},
			{"role": "user", "content": prompt},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	url := "https://openrouter.ai/api/v1/chat/completions"

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenRouter API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenRouter API error (status %d): %s", resp.StatusCode, string(body))
	}

	var orResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&orResp); err != nil {
		return nil, fmt.Errorf("failed to decode OpenRouter response: %w", err)
	}

	return orResp, nil
}

// handleSemanticDispatch uses AI to route user intent to appropriate tool.
func (e *DiscoveryEngine) handleSemanticDispatch(intent string) (interface{}, error) {
	// Get AI routing credentials
	provider, model, key, isFallback := e.getAIRoutingCredentials()
	if key == "" {
		return nil, fmt.Errorf("AI routing credentials not configured. Please configure AI routing settings in MCP Scooter settings.")
	}

	// Build list of active tools
	activeTools := e.ListActive()
	toolsList := make([]string, 0, len(activeTools))
	for _, name := range activeTools {
		serverTools := e.GetActiveToolsForServer(name)
		for _, t := range serverTools {
			toolsList = append(toolsList, fmt.Sprintf("%s: %s", name, t.Name))
		}
	}

	// Build system prompt
	prompt := fmt.Sprintf(
		"You are the internal Router for MCP Scooter. The user wants to achieve this intent: '%s'.\n"+
			"Based on these active tools: [%s], return only a JSON object:\n"+
			"{ \"tool_name\": \"name\", \"arguments\": { ... } }.\n"+
			"Return NO other text. If no tool matches, return { \"error\": \"no_matching_tool\" }.",
		intent, strings.Join(toolsList, ", "),
	)

	// Try primary provider first
	response, err := e.callInternalAI(provider, model, key, prompt)
	if err != nil {
		// Try fallback if primary fails
		logger.AddLog("ERROR", fmt.Sprintf("Primary AI provider failed: %v, trying fallback", err))
		provider, model, key, _ = e.getAIRoutingCredentials()
		if key != "" {
			response, err = e.callInternalAI(provider, model, key, prompt)
			if err != nil {
				return nil, fmt.Errorf("both primary and fallback AI providers failed: %w", err)
			}
		}
	}

	// Parse JSON response from AI
	var routingDecision map[string]interface{}

	// Handle different response formats
	var jsonText string
	if provider == "openrouter" {
		// OpenRouter returns { "choices": [ { "message": { "content": "..." } } ] }
		if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						jsonText = content
					}
				}
			}
		}
	} else {
		// Gemini returns { "candidates": [ { "content": { "parts": [ { "text": "..." } ] } } ] }
		if candidates, ok := response["candidates"].([]interface{}); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]interface{}); ok {
				if content, ok := candidate["content"].(map[string]interface{}); ok {
					if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
						if part, ok := parts[0].(map[string]interface{}); ok {
							if text, ok := part["text"].(string); ok {
								jsonText = text
							}
						}
					}
				}
			}
		}
	}

	if jsonText == "" {
		return nil, fmt.Errorf("failed to extract JSON response from AI provider")
	}

	if err := json.Unmarshal([]byte(jsonText), &routingDecision); err != nil {
		return nil, fmt.Errorf("failed to parse routing decision: %w", err)
	}

	// Check for routing error
	if errMsg, ok := routingDecision["error"].(string); ok && errMsg != "" {
		return map[string]interface{}{
			"status":  "no_match",
			"intent":  intent,
			"message": fmt.Sprintf("No tool matches intent: %s", intent),
		}, nil
	}

	// Execute routed tool
	toolName, ok := routingDecision["tool_name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid routing decision: missing tool_name")
	}

	arguments, _ := routingDecision["arguments"].(map[string]interface{})
	if arguments == nil {
		arguments = make(map[string]interface{})
	}

	logger.AddLog("INFO", fmt.Sprintf("AI routed intent to tool: %s (using %s: %s)", toolName, provider, model))

	// Call tool
	result, err := e.CallTool(toolName, arguments)
	if err != nil {
		return nil, fmt.Errorf("routed tool execution failed: %w", err)
	}

	return map[string]interface{}{
		"status":       "success",
		"routed_to":    toolName,
		"intent":       intent,
		"ai_provider":  provider,
		"ai_model":     model,
		"is_fallback":  isFallback,
		"result":       result,
	}, nil
}
