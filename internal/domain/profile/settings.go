package profile

// Settings represents global application configuration.
type Settings struct {
	ControlPort int  `yaml:"control_port" json:"control_port"`
	McpPort     int  `yaml:"mcp_port" json:"mcp_port"`
	EnableBeta  bool `yaml:"enable_beta" json:"enable_beta"`
}

// DefaultSettings returns the standard port configuration.
func DefaultSettings() Settings {
	return Settings{
		ControlPort: 6200,
		McpPort:     6277,
		EnableBeta:  false,
	}
}
