package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ZedIntegration handles configuring Zed to use MCP Scooter.
type ZedIntegration struct{}

// Configure adds the MCP Scooter server to Zed's settings.json.
func (z *ZedIntegration) Configure(port int, profileID string) error {
	path, err := z.findConfig()
	if err != nil {
		return err
	}

	var config map[string]interface{}

	data, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(data, &config)
	}

	if config == nil {
		config = make(map[string]interface{})
	}

	// Zed uses "context_servers" key for MCP
	contextServers, ok := config["context_servers"].(map[string]interface{})
	if !ok {
		contextServers = make(map[string]interface{})
		config["context_servers"] = contextServers
	}

	// Add or update MCP Scooter entry
	url := fmt.Sprintf("http://localhost:%d/profiles/%s/sse", port, profileID)
	if profileID == "work" {
		url = fmt.Sprintf("http://localhost:%d/sse", port)
	}

	contextServers["mcp-scooter"] = map[string]interface{}{
		"url": url,
	}

	newData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, newData, 0644)
}

func (z *ZedIntegration) findConfig() (string, error) {
	// Try Windows path first if on Windows
	appData := os.Getenv("APPDATA")
	if appData != "" {
		path := filepath.Join(appData, "Zed", "settings.json")
		if _, err := os.Stat(filepath.Dir(path)); err == nil {
			return path, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Linux/macOS fallback
	paths := []string{
		filepath.Join(home, ".config", "zed", "settings.json"),
		filepath.Join(home, "Library", "Application Support", "Zed", "settings.json"),
	}

	for _, p := range paths {
		if _, err := os.Stat(filepath.Dir(p)); err == nil {
			return p, nil
		}
	}

	// Default to Linux style if nothing else found
	zedDir := filepath.Join(home, ".config", "zed")
	os.MkdirAll(zedDir, 0755)
	return filepath.Join(zedDir, "settings.json"), nil
}
