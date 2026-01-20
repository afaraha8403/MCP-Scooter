package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// CodexIntegration handles configuring Codex to use MCP Scooter.
type CodexIntegration struct{}

// Configure adds the MCP Scooter server to Codex's config.toml.
func (c *CodexIntegration) Configure(port int, profileID string, apiKey string) error {
	path, err := c.findConfig()
	if err != nil {
		return err
	}

	var config map[string]interface{}

	data, err := os.ReadFile(path)
	if err == nil {
		toml.Unmarshal(data, &config)
	}

	if config == nil {
		config = make(map[string]interface{})
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
		config["mcpServers"] = mcpServers
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

	mcpServers["mcp-scooter"] = serverConfig

	newData, err := toml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(path, newData, 0644)
}

func (c *CodexIntegration) findConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	codexDir := filepath.Join(home, ".codex")
	os.MkdirAll(codexDir, 0755)

	return filepath.Join(codexDir, "config.toml"), nil
}
