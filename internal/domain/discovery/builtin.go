package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/registry"
	"github.com/mcp-scooter/scooter/internal/logger"
)

// PrimordialTools returns the definitions for built-in MCP tools.
// These are the "meta-layer" tools that are always available to AI clients.
// External tools (like brave-search) are NOT exposed until explicitly activated via scooter_activate.
//
// Simplified to just 2 core tools:
// - scooter_find: Discover available tools in the registry
// - scooter_activate: Turn on a tool server for the current session
//
// Note: scooter_ai (AI-powered intent routing) is planned for a future release.
func PrimordialTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "scooter_find",
			Title:       "Find Tools",
			Description: "Search the registry for available MCP tools.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_find",
					Description: "Search the Local Registry and Community Catalog for MCP tools. Returns tool names, descriptions, and available sub-tools. Use this to discover what tools can be activated.",
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
			Name:        "scooter_activate",
			Title:       "Activate Tool",
			Description: "Turn on an MCP tool server. Once activated, its functions become available.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
			Tools: []registry.Tool{
				{
					Name:        "scooter_activate",
					Description: "Turn on an MCP tool server for the current session. Once activated, the tool's functions become available for use. Use scooter_find first to discover available tools.",
					InputSchema: &registry.JSONSchema{
						Type: "object",
						Properties: map[string]registry.PropertySchema{
							"tool_name": {
								Type:        "string",
								Description: "The name of the tool/server to activate (e.g., 'brave-search', 'github'). Use the server name, not individual function names.",
							},
						},
						Required: []string{"tool_name"},
					},
				},
			},
		},
	}
}

// HandleBuiltinTool handles calls to the primordial tools.
// Simplified to just 2 core tools: scooter_find and scooter_activate
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
		
	case "scooter_activate":
		tool, ok := params["tool_name"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_name is required")
		}
		
		// Check if tool is already active (already "on")
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
			toolSchemas := make([]map[string]interface{}, 0, len(availableTools))
			for _, t := range availableTools {
				toolNames = append(toolNames, t.Name)
				toolSchema := buildToolSchema(t)
				toolSchemas = append(toolSchemas, toolSchema)
			}
			
		// Build example call for the first tool
		exampleTool := ""
		exampleCall := ""
		if len(toolNames) > 0 {
			exampleTool = toolNames[0]
			exampleCall = fmt.Sprintf(`{"server": "mcp-scooter", "tool": "%s", "arguments": {...}}`, exampleTool)
		}
		
		return map[string]interface{}{
			"status":          "already_on",
			"server":          "mcp-scooter",  // Always mcp-scooter, not the original server
			"activated_from":  tool,
			"available_tools": toolNames,
			"tool_schemas":    toolSchemas,
			"message":         fmt.Sprintf("Tools from '%s' are ready. Server name for ALL calls: mcp-scooter", tool),
			"how_to_call":     fmt.Sprintf("Use server='mcp-scooter' with tool='%s' (or any tool listed above)", exampleTool),
			"example":         exampleCall,
			"warning":         "DO NOT use 'context7', 'brave-search', or any other server name. ONLY use 'mcp-scooter'.",
		}, nil
		}

		err := e.Add(tool)
		if err != nil {
			return nil, err
		}
		
		// Get the tools that are now available from this server
		availableTools := e.GetActiveToolsForServer(tool)
		toolNames := make([]string, 0, len(availableTools))
		toolSchemas := make([]map[string]interface{}, 0, len(availableTools))
		for _, t := range availableTools {
			toolNames = append(toolNames, t.Name)
			toolSchema := buildToolSchema(t)
			toolSchemas = append(toolSchemas, toolSchema)
		}
		
		// Build example call for the first tool
		exampleTool := ""
		exampleCall := ""
		if len(toolNames) > 0 {
			exampleTool = toolNames[0]
			exampleCall = fmt.Sprintf(`{"server": "mcp-scooter", "tool": "%s", "arguments": {...}}`, exampleTool)
		}
		
		return map[string]interface{}{
			"status":          "on",
			"server":          "mcp-scooter",  // Always mcp-scooter, not the original server
			"activated_from":  tool,
			"available_tools": toolNames,
			"tool_schemas":    toolSchemas,
			"message":         fmt.Sprintf("Tools from '%s' are now ready. Server name for ALL calls: mcp-scooter", tool),
			"how_to_call":     fmt.Sprintf("Use server='mcp-scooter' with tool='%s' (or any tool listed above)", exampleTool),
			"example":         exampleCall,
			"warning":         "DO NOT use 'context7', 'brave-search', or any other server name. ONLY use 'mcp-scooter'.",
		}, nil

	default:
		return nil, fmt.Errorf("unknown builtin tool: %s", name)
	}
}


// buildToolSchema creates a comprehensive schema for a tool that agents can use to understand
// how to call it correctly. This includes the full input schema with types, descriptions,
// required fields, and constraints.
func buildToolSchema(t registry.Tool) map[string]interface{} {
	schema := map[string]interface{}{
		"name":        t.Name,
		"description": t.Description,
	}

	if t.Title != "" {
		schema["title"] = t.Title
	}

	if t.InputSchema != nil {
		// Build a detailed parameters object
		parameters := make([]map[string]interface{}, 0)
		requiredSet := make(map[string]bool)
		for _, req := range t.InputSchema.Required {
			requiredSet[req] = true
		}

		for propName, prop := range t.InputSchema.Properties {
			param := map[string]interface{}{
				"name":        propName,
				"type":        prop.Type,
				"required":    requiredSet[propName],
				"description": prop.Description,
			}

			// Include constraints if present
			if prop.Default != nil {
				param["default"] = prop.Default
			}
			if len(prop.Enum) > 0 {
				param["enum"] = prop.Enum
			}
			if prop.Minimum != nil {
				param["minimum"] = *prop.Minimum
			}
			if prop.Maximum != nil {
				param["maximum"] = *prop.Maximum
			}
			if prop.MinLength != nil {
				param["minLength"] = *prop.MinLength
			}
			if prop.MaxLength != nil {
				param["maxLength"] = *prop.MaxLength
			}

			parameters = append(parameters, param)
		}

		schema["parameters"] = parameters
		schema["required"] = t.InputSchema.Required
	}

	// Include sample input if available - this is gold for agents
	if t.SampleInput != nil {
		schema["example"] = t.SampleInput
	}

	// Include annotations if available
	if t.Annotations != nil {
		annotations := map[string]interface{}{}
		if t.Annotations.ReadOnlyHint {
			annotations["readOnly"] = true
		}
		if t.Annotations.DestructiveHint {
			annotations["destructive"] = true
		}
		if len(annotations) > 0 {
			schema["hints"] = annotations
		}
	}

	return schema
}


// =============================================================================
// AI ROUTING (Reserved for future scooter_ai tool)
// These functions implement AI-powered intent routing and will be exposed
// as scooter_ai in a future release.
// =============================================================================

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
