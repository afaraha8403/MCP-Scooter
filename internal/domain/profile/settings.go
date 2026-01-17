package profile

import (
	"crypto/rand"
	"encoding/base64"
)

// Settings represents global application configuration.
type Settings struct {
	ControlPort   int    `yaml:"control_port" json:"control_port"`
	McpPort       int    `yaml:"mcp_port" json:"mcp_port"`
	EnableBeta    bool   `yaml:"enable_beta" json:"enable_beta"`
	GatewayAPIKey string `yaml:"gateway_api_key" json:"gateway_api_key"`
}

// DefaultSettings returns the standard port configuration.
func DefaultSettings() Settings {
	return Settings{
		ControlPort: 6200,
		McpPort:     6277,
		EnableBeta:  false,
	}
}

// GenerateAPIKey creates a secure random key for the MCP gateway.
func GenerateAPIKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "sk-scooter-" + base64.RawURLEncoding.EncodeToString(b)
}
