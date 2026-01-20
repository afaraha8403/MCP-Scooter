package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GeminiIntegration handles configuring Google Antigravity and Gemini CLI.
type GeminiIntegration struct{}

// Configure adds the MCP Scooter server to Gemini's settings.json.
func (g *GeminiIntegration) Configure(port int, profileID string, apiKey string) error {
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

func (g *GeminiIntegration) findConfig() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	geminiDir := filepath.Join(home, ".gemini")
	os.MkdirAll(geminiDir, 0755)

	return filepath.Join(geminiDir, "settings.json"), nil
}
