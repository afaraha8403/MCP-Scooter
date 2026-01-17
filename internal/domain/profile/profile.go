package profile

import "errors"

// Profile represents an isolated environment for MCP tools.
type Profile struct {
	// ID is the unique identifier for the profile (e.g., "work", "personal")
	ID string `yaml:"id" json:"id"`

	// RemoteAuthMode determines how to authenticate with remote servers ("oauth2", "none", etc.)
	// This is NOT for IDE-to-Scooter authentication (use Settings.GatewayAPIKey for that).
	RemoteAuthMode string `yaml:"remote_auth_mode" json:"remote_auth_mode"`

	// RemoteServerURL is the URL of the remote MCP server to proxy to.
	// When set, Scooter acts as a proxy and uses RemoteAuthMode to authenticate.
	RemoteServerURL string `yaml:"remote_server_url" json:"remote_server_url"`

	// Env contains environment variables to inject into tools.
	// Use this for tool-specific API keys (e.g., BRAVE_API_KEY, GITHUB_TOKEN).
	Env map[string]string `yaml:"env" json:"env"`

	// AllowTools is a list of allowed tool names
	AllowTools []string `yaml:"allow_tools" json:"allow_tools"`
}

// Validate checks if the profile configuration is valid.
func (p Profile) Validate() error {
	if p.ID == "" {
		return errors.New("profile id is required")
	}
	return nil
}
