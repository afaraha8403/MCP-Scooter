package discovery

import (
	"fmt"
)

// PrimordialTools returns the definitions for built-in MCP tools.
func PrimordialTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "scooter_find",
			Title:       "Search Registry",
			Description: "Searches the Local Registry and Community Catalog for MCP tools.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
		},
		{
			Name:        "scooter_add",
			Title:       "Enable Tool",
			Description: "Installs and enables an MCP tool for the current session.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
		},
		{
			Name:        "scooter_remove",
			Title:       "Disable Tool",
			Description: "Unloads an MCP tool to free up context window space.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
		},
		{
			Name:        "scooter_list_active",
			Title:       "List Active",
			Description: "Returns a list of currently loaded tools in this session.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
		},
		{
			Name:        "scooter_code_interpreter",
			Title:       "Code Interpreter",
			Description: "Executes sandboxed JavaScript. Use 'callTool(name, args)' to chain other tools.",
			Category:    "system",
			Source:      "builtin",
			Installed:   true,
		},
	}
}

// HandleBuiltinTool handles calls to the primordial tools.
func (e *DiscoveryEngine) HandleBuiltinTool(name string, params map[string]interface{}) (interface{}, error) {
	switch name {
	case "scooter_find":
		query, _ := params["query"].(string)
		return e.Find(query), nil
	case "scooter_add":
		tool, ok := params["tool_name"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_name is required")
		}
		err := e.Add(tool)
		if err != nil {
			return nil, err
		}
		return map[string]string{"status": "installed", "tool": tool}, nil
	case "scooter_remove":
		tool, ok := params["tool_name"].(string)
		if !ok {
			return nil, fmt.Errorf("tool_name is required")
		}
		err := e.Remove(tool)
		if err != nil {
			return nil, err
		}
		return map[string]string{"status": "removed", "tool": tool}, nil
	case "scooter_list_active":
		return e.ListActive(), nil
	case "scooter_code_interpreter":
		script, _ := params["script"].(string)
		args, _ := params["arguments"].(map[string]interface{})
		interpreter := NewCodeInterpreter(e.CallTool)
		return interpreter.Execute(script, args)
	default:
		return nil, fmt.Errorf("unknown builtin tool: %s", name)
	}
}
