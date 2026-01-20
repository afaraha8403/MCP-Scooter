package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeIntegration handles configuring Claude Desktop to use MCP Scooter.
type ClaudeIntegration struct{}

// Configure adds the MCP Scooter server to Claude Desktop's config file.
func (c *ClaudeIntegration) Configure(port int, profileID string, apiKey string) error {
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

	// Add or update MCP Scooter entry for Claude
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

// ConfigureCode adds the MCP Scooter server to Claude Code's settings file.
func (c *ClaudeIntegration) ConfigureCode(port int, profileID string, apiKey string) error {
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
