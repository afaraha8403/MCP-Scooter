package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// VSCodeIntegration handles configuring VS Code to use MCP Scooter.
type VSCodeIntegration struct{}

// Configure adds the MCP Scooter server to VS Code's mcp.json.
// Note: While the PRD mentions ~/.vscode/mcp.json, VS Code usually
// uses settings.json for global settings or project-level mcp.json.
// We will follow the PRD's request for ~/.vscode/mcp.json as a convention
// that extensions might pick up.
func (v *VSCodeIntegration) Configure(port int, profileID string, apiKey string) error {
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

	// Add or update MCP Scooter entry
	url := fmt.Sprintf("http://127.0.0.1:%d/profiles/%s/sse", port, profileID)
	if profileID == "work" {
		url = fmt.Sprintf("http://127.0.0.1:%d/sse", port)
	}

	serverConfig := map[string]interface{}{
		"type": "sse",
		"url":  url,
	}

	if apiKey != "" {
		serverConfig["headers"] = map[string]string{
			"Authorization": "Bearer " + apiKey,
		}
	}

	config.McpServers["mcp-scooter"] = serverConfig

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
