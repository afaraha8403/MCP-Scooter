package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// CodexIntegration handles configuring Codex to use MCP Scout.
type CodexIntegration struct{}

// Configure adds the MCP Scout server to Codex's config.toml.
func (c *CodexIntegration) Configure(port int) error {
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

	// Add or update MCP Scout entry
	mcpServers["mcp-scout"] = map[string]interface{}{
		"type": "sse",
		"url":  fmt.Sprintf("http://localhost:%d/sse", port),
	}

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
