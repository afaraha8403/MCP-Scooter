package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeIntegration handles configuring Claude Desktop to use MCP Scout.
type ClaudeIntegration struct{}

// Configure adds the MCP Scout server to Claude Desktop's config file.
func (c *ClaudeIntegration) Configure(port int) error {
	path, err := c.findConfig()
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

	// Add or update MCP Scout entry for Claude
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

// ConfigureCode adds the MCP Scout server to Claude Code's settings file.
func (c *ClaudeIntegration) ConfigureCode(port int) error {
	path, err := c.findCodeConfig()
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

func (c *ClaudeIntegration) findConfig() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, "AppData", "Roaming")
	}

	path := filepath.Join(appData, "Claude", "claude_desktop_config.json")

	// Create directory if it doesn't exist
	os.MkdirAll(filepath.Dir(path), 0755)

	return path, nil
}

func (c *ClaudeIntegration) findCodeConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(path), 0755)
	return path, nil
}
