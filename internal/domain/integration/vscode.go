package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// VSCodeIntegration handles configuring VS Code to use MCP Scout.
type VSCodeIntegration struct{}

// Configure adds the MCP Scout server to VS Code's mcp.json.
// Note: While the PRD mentions ~/.vscode/mcp.json, VS Code usually
// uses settings.json for global settings or project-level mcp.json.
// We will follow the PRD's request for ~/.vscode/mcp.json as a convention
// that extensions might pick up.
func (v *VSCodeIntegration) Configure(port int) error {
	path, err := v.findConfig()
	if err != nil {
		return err
	}

	var config struct {
		McpServers map[string]interface{} `json:"mcpServers"`
	}

	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, &config)
	}

	if config.McpServers == nil {
		config.McpServers = make(map[string]interface{})
	}

	// Add or update MCP Scout entry
	config.McpServers["mcp-scout"] = map[string]interface{}{
		"type": "sse",
		"url":  fmt.Sprintf("http://localhost:%d/sse", port),
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, newData, 0644)
}

func (v *VSCodeIntegration) findConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	vscodeDir := filepath.Join(home, ".vscode")
	os.MkdirAll(vscodeDir, 0755)

	return filepath.Join(vscodeDir, "mcp.json"), nil
}
