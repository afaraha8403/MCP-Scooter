package profile

import "errors"

// Profile represents an isolated environment for MCP tools.
type Profile struct {
	// Unique identifier for the profile (e.g., "work", "personal")
	ID string `yaml:"id"`

	// AuthMode determines how to authenticate with remote servers ("oauth2", "none", etc.)
	AuthMode string `yaml:"auth_mode"`

	// RemoteServerURL is the URL of the remote MCP server to proxy to
	RemoteServerURL string `yaml:"remote_server_url"`

	// Env contains environment variables to inject into tools
	Env map[string]string `yaml:"env"`

	// AllowTools is a list of allowed tool names
	AllowTools []string `yaml:"allow_tools"`
}

// Validate checks if the profile configuration is valid.
func (p Profile) Validate() error {
	if p.ID == "" {
		return errors.New("profile id is required")
	}
	return nil
}
