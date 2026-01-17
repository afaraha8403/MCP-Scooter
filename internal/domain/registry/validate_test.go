package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate_ValidEntry(t *testing.T) {
	entry := &MCPEntry{
		Name:        "test-mcp",
		Version:     "1.0.0",
		Title:       "Test MCP",
		Description: "A test MCP server for validation",
		Category:    CategoryUtility,
		Source:      SourceCommunity,
		Auth: &Authorization{
			Type: AuthNone,
		},
		Tools: []Tool{
			{
				Name:        "test_tool",
				Description: "A test tool for validation testing",
				InputSchema: &JSONSchema{
					Type:       "object",
					Properties: map[string]PropertySchema{},
				},
			},
		},
		Package: &Package{
			Type: PackageNPM,
			Name: "@test/test-mcp",
		},
	}

	result := Validate(entry)
	assert.True(t, result.Valid, "Expected valid entry, got errors: %v", result.Errors)
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	entry := &MCPEntry{}

	result := Validate(entry)
	assert.False(t, result.Valid)
	assert.True(t, len(result.Errors) > 0)
}

func TestValidate_InvalidName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid lowercase", "my-tool", true},
		{"valid with numbers", "tool123", true},
		{"invalid uppercase", "MyTool", false},
		{"invalid starts with number", "123tool", false},
		{"invalid special chars", "my_tool", false},
		{"too short", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := createMinimalEntry()
			entry.Name = tt.input

			result := Validate(entry)
			hasNameError := false
			for _, e := range result.Errors {
				if e.Field == "name" {
					hasNameError = true
					break
				}
			}
			if tt.expected {
				assert.False(t, hasNameError, "Expected no name error for %q", tt.input)
			} else {
				assert.True(t, hasNameError, "Expected name error for %q", tt.input)
			}
		})
	}
}

func TestValidate_InvalidVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid semver", "1.0.0", true},
		{"valid with prerelease", "1.0.0-beta.1", true},
		{"invalid no patch", "1.0", false},
		{"invalid text", "latest", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := createMinimalEntry()
			entry.Version = tt.input

			result := Validate(entry)
			hasVersionError := false
			for _, e := range result.Errors {
				if e.Field == "version" {
					hasVersionError = true
					break
				}
			}
			if tt.expected {
				assert.False(t, hasVersionError, "Expected no version error for %q", tt.input)
			} else {
				assert.True(t, hasVersionError, "Expected version error for %q", tt.input)
			}
		})
	}
}

func TestValidate_AuthAPIKey(t *testing.T) {
	entry := createMinimalEntry()
	entry.Auth = &Authorization{
		Type:        AuthAPIKey,
		EnvVar:      "MY_API_KEY",
		DisplayName: "My API Key",
	}

	result := Validate(entry)
	assert.True(t, result.Valid, "Expected valid API key auth, got errors: %v", result.Errors)
}

func TestValidate_AuthAPIKey_MissingEnvVar(t *testing.T) {
	entry := createMinimalEntry()
	entry.Auth = &Authorization{
		Type:        AuthAPIKey,
		DisplayName: "My API Key",
	}

	result := Validate(entry)
	assert.False(t, result.Valid)

	hasEnvVarError := false
	for _, e := range result.Errors {
		if e.Field == "authorization.env_var" {
			hasEnvVarError = true
			break
		}
	}
	assert.True(t, hasEnvVarError)
}

func TestValidate_AuthOAuth2(t *testing.T) {
	entry := createMinimalEntry()
	entry.Auth = &Authorization{
		Type:     AuthOAuth2,
		Provider: "google",
		OAuth: &OAuthConfig{
			AuthorizationURL: "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:         "https://oauth2.googleapis.com/token",
			Scopes:           []string{"read"},
			TokenEnv:         "GOOGLE_ACCESS_TOKEN",
		},
	}

	result := Validate(entry)
	assert.True(t, result.Valid, "Expected valid OAuth2 auth, got errors: %v", result.Errors)
}

func TestValidate_Tools_DuplicateNames(t *testing.T) {
	entry := createMinimalEntry()
	entry.Tools = []Tool{
		{
			Name:        "my_tool",
			Description: "First tool with this name",
			InputSchema: &JSONSchema{Type: "object", Properties: map[string]PropertySchema{}},
		},
		{
			Name:        "my_tool",
			Description: "Second tool with same name",
			InputSchema: &JSONSchema{Type: "object", Properties: map[string]PropertySchema{}},
		},
	}

	result := Validate(entry)
	assert.False(t, result.Valid)

	hasDuplicateError := false
	for _, e := range result.Errors {
		if e.Field == "tools[1].name" && e.Message == "duplicate tool name: my_tool" {
			hasDuplicateError = true
			break
		}
	}
	assert.True(t, hasDuplicateError)
}

func TestValidate_Package_NPM(t *testing.T) {
	entry := createMinimalEntry()
	entry.Package = &Package{
		Type: PackageNPM,
		Name: "@scope/package-name",
	}

	result := Validate(entry)
	assert.True(t, result.Valid, "Expected valid npm package, got errors: %v", result.Errors)
}

func TestValidate_Package_Binary_MissingPlatforms(t *testing.T) {
	entry := createMinimalEntry()
	entry.Package = &Package{
		Type: PackageBinary,
	}

	result := Validate(entry)
	assert.False(t, result.Valid)

	hasPlatformsError := false
	for _, e := range result.Errors {
		if e.Field == "package.platforms" {
			hasPlatformsError = true
			break
		}
	}
	assert.True(t, hasPlatformsError)
}

func TestValidate_Warnings(t *testing.T) {
	entry := createMinimalEntry()
	// No icon, no about, no homepage

	result := Validate(entry)
	assert.True(t, result.Valid)
	assert.True(t, len(result.Warnings) > 0, "Expected warnings for missing optional fields")
}

// Helper function to create a minimal valid entry
func createMinimalEntry() *MCPEntry {
	return &MCPEntry{
		Name:        "test-mcp",
		Version:     "1.0.0",
		Title:       "Test MCP",
		Description: "A test MCP server for validation",
		Category:    CategoryUtility,
		Source:      SourceCommunity,
		Auth: &Authorization{
			Type: AuthNone,
		},
		Tools: []Tool{
			{
				Name:        "test_tool",
				Description: "A test tool for validation testing",
				InputSchema: &JSONSchema{
					Type:       "object",
					Properties: map[string]PropertySchema{},
				},
			},
		},
		Package: &Package{
			Type: PackageNPM,
			Name: "@test/test-mcp",
		},
	}
}
