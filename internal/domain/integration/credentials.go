package integration

import (
	"fmt"

	"github.com/mcp-scooter/scooter/internal/domain/registry"
)

// CredentialManager handles secure credential storage and retrieval for MCP tools.
type CredentialManager struct {
	keychain *Keychain
}

// NewCredentialManager creates a new credential manager.
func NewCredentialManager() *CredentialManager {
	return &CredentialManager{
		keychain: NewKeychain("mcp-scooter"),
	}
}

// GetCredentialsForTool retrieves credentials for a tool based on its authorization config.
// Returns a map of environment variable names to values.
func (c *CredentialManager) GetCredentialsForTool(toolName string, auth *registry.Authorization) (map[string]string, error) {
	creds := make(map[string]string)

	if auth == nil {
		return creds, nil
	}

	// Handle single env_var (most common case)
	if auth.EnvVar != "" {
		secret, err := c.keychain.GetSecret(fmt.Sprintf("%s:%s", toolName, auth.EnvVar))
		if err == nil && secret != "" {
			creds[auth.EnvVar] = secret
		}
	}

	// Handle multiple env_vars (for tools with complex auth)
	for _, envDef := range auth.EnvVars {
		secret, err := c.keychain.GetSecret(fmt.Sprintf("%s:%s", toolName, envDef.Name))
		if err == nil && secret != "" {
			creds[envDef.Name] = secret
		}
	}

	// Handle OAuth tokens
	if auth.OAuth != nil && auth.OAuth.TokenEnv != "" {
		token, err := c.keychain.GetSecret(fmt.Sprintf("%s:%s", toolName, auth.OAuth.TokenEnv))
		if err == nil && token != "" {
			creds[auth.OAuth.TokenEnv] = token
		}
	}

	return creds, nil
}

// SetCredential stores a credential securely in the keychain.
func (c *CredentialManager) SetCredential(toolName, envVar, value string) error {
	return c.keychain.SetSecret(fmt.Sprintf("%s:%s", toolName, envVar), value)
}

// GetCredential retrieves a single credential from the keychain.
func (c *CredentialManager) GetCredential(toolName, envVar string) (string, error) {
	return c.keychain.GetSecret(fmt.Sprintf("%s:%s", toolName, envVar))
}

// DeleteCredential removes a credential from the keychain.
func (c *CredentialManager) DeleteCredential(toolName, envVar string) error {
	return c.keychain.RemoveSecret(fmt.Sprintf("%s:%s", toolName, envVar))
}

// HasRequiredCredentials checks if all required credentials are present.
func (c *CredentialManager) HasRequiredCredentials(toolName string, auth *registry.Authorization) (bool, []string) {
	if auth == nil || !auth.Required {
		return true, nil
	}

	missing := []string{}

	// Check single env_var
	if auth.EnvVar != "" {
		secret, _ := c.keychain.GetSecret(fmt.Sprintf("%s:%s", toolName, auth.EnvVar))
		if secret == "" {
			missing = append(missing, auth.EnvVar)
		}
	}

	// Check multiple env_vars
	for _, envDef := range auth.EnvVars {
		if envDef.Required {
			secret, _ := c.keychain.GetSecret(fmt.Sprintf("%s:%s", toolName, envDef.Name))
			if secret == "" {
				missing = append(missing, envDef.Name)
			}
		}
	}

	return len(missing) == 0, missing
}
