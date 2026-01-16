package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CursorIntegration handles configuring Cursor to use MCP Scout.
type CursorIntegration struct{}

// Configure adds the MCP Scout server to Cursor's mcp.json.
func (c *CursorIntegration) Configure(port int) error {
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

func (c *CursorIntegration) findConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Common Windows paths for Cursor MCP config
	paths := []string{
		filepath.Join(home, ".cursor", "mcp.json"),
		filepath.Join(os.Getenv("APPDATA"), "Cursor", "User", "globalStorage", "mcp.json"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// If none exist, create in ~/.cursor/mcp.json
	cursorDir := filepath.Join(home, ".cursor")
	os.MkdirAll(cursorDir, 0755)
	return filepath.Join(cursorDir, "mcp.json"), nil
}
