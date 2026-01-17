package profile

// Settings represents global application configuration.
type Settings struct {
	ControlPort int `yaml:"control_port" json:"control_port"`
	McpPort     int `yaml:"mcp_port" json:"mcp_port"`
}

// DefaultSettings returns the standard port configuration.
func DefaultSettings() Settings {
	return Settings{
		ControlPort: 6200,
		McpPort:     6277,
	}
}
