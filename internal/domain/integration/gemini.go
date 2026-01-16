package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GeminiIntegration handles configuring Google Antigravity and Gemini CLI.
type GeminiIntegration struct{}

// Configure adds the MCP Scout server to Gemini's settings.json.
func (g *GeminiIntegration) Configure(port int) error {
	path, err := g.findConfig()
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

func (g *GeminiIntegration) findConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	geminiDir := filepath.Join(home, ".gemini")
	os.MkdirAll(geminiDir, 0755)

	return filepath.Join(geminiDir, "settings.json"), nil
}
