package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mcp-scooter/scooter/internal/domain/integration"
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHome(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "mcp-test-home")
	require.NoError(t, err)

	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	oldAppData := os.Getenv("APPDATA")

	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)
	os.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))

	return tmpDir, func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
		os.Setenv("APPDATA", oldAppData)
		os.RemoveAll(tmpDir)
	}
}

func TestCursorIntegration(t *testing.T) {
	home, cleanup := setupTestHome(t)
	defer cleanup()

	c := &integration.CursorIntegration{}
	err := c.Configure(6277)
	assert.NoError(t, err)

	path := filepath.Join(home, ".cursor", "mcp.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var config struct {
		McpServers map[string]interface{} `json:"mcpServers"`
	}
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	scooter := config.McpServers["mcp-scooter"].(map[string]interface{})
	assert.Equal(t, "sse", scooter["type"])
	assert.Equal(t, "http://localhost:6277/sse", scooter["url"])
}

func TestVSCodeIntegration(t *testing.T) {
	home, cleanup := setupTestHome(t)
	defer cleanup()

	v := &integration.VSCodeIntegration{}
	err := v.Configure(6277)
	assert.NoError(t, err)

	path := filepath.Join(home, ".vscode", "mcp.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var config struct {
		McpServers map[string]interface{} `json:"mcpServers"`
	}
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	scooter := config.McpServers["mcp-scooter"].(map[string]interface{})
	assert.Equal(t, "sse", scooter["type"])
	assert.Equal(t, "http://localhost:6277/sse", scooter["url"])
}

func TestGeminiIntegration(t *testing.T) {
	home, cleanup := setupTestHome(t)
	defer cleanup()

	g := &integration.GeminiIntegration{}
	err := g.Configure(6277)
	assert.NoError(t, err)

	path := filepath.Join(home, ".gemini", "settings.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var config struct {
		McpServers map[string]interface{} `json:"mcpServers"`
	}
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	scooter := config.McpServers["mcp-scooter"].(map[string]interface{})
	assert.Equal(t, "sse", scooter["type"])
	assert.Equal(t, "http://localhost:6277/sse", scooter["url"])
}

func TestCodexIntegration(t *testing.T) {
	home, cleanup := setupTestHome(t)
	defer cleanup()

	c := &integration.CodexIntegration{}
	err := c.Configure(6277)
	assert.NoError(t, err)

	path := filepath.Join(home, ".codex", "config.toml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var config map[string]interface{}
	err = toml.Unmarshal(data, &config)
	require.NoError(t, err)

	mcpServers := config["mcpServers"].(map[string]interface{})
	scooter := mcpServers["mcp-scooter"].(map[string]interface{})
	assert.Equal(t, "sse", scooter["type"])
	assert.Equal(t, "http://localhost:6277/sse", scooter["url"])
}

func TestZedIntegration(t *testing.T) {
	home, cleanup := setupTestHome(t)
	defer cleanup()

	z := &integration.ZedIntegration{}
	err := z.Configure(6277)
	assert.NoError(t, err)

	// Zed uses .config/zed/settings.json as default fallback in implementation
	path := filepath.Join(home, ".config", "zed", "settings.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var config map[string]interface{}
	err = json.Unmarshal(data, &config)
	require.NoError(t, err)

	contextServers := config["context_servers"].(map[string]interface{})
	scooter := contextServers["mcp-scooter"].(map[string]interface{})
	assert.Equal(t, "http://localhost:6277/sse", scooter["url"])
}
