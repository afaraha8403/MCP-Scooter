package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult holds the result of validating an MCP entry.
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// Regular expressions for validation
var (
	namePattern    = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	versionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?$`)
	toolNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	colorPattern   = regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)
	envVarPattern  = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	sha256Pattern  = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// ValidCategories contains all valid category values.
var ValidCategories = map[Category]bool{
	CategoryDevelopment:   true,
	CategoryDatabase:      true,
	CategoryProductivity:  true,
	CategoryCommunication: true,
	CategorySearch:        true,
	CategoryCloud:         true,
	CategoryAnalytics:     true,
	CategoryAI:            true,
	CategoryUtility:       true,
	CategoryCustom:        true,
}

// ValidSources contains all valid source values.
var ValidSources = map[Source]bool{
	SourceOfficial:   true,
	SourceCommunity:  true,
	SourceEnterprise: true,
	SourceLocal:      true,
}

// ValidAuthTypes contains all valid auth type values.
var ValidAuthTypes = map[AuthType]bool{
	AuthNone:        true,
	AuthAPIKey:      true,
	AuthOAuth2:      true,
	AuthBearerToken: true,
	AuthCustom:      true,
}

// ValidPackageTypes contains all valid package type values.
var ValidPackageTypes = map[PackageType]bool{
	PackageNPM:    true,
	PackagePyPI:   true,
	PackageCargo:  true,
	PackageWASM:   true,
	PackageDocker: true,
	PackageBinary: true,
}

// ValidTransportTypes contains all valid transport type values.
var ValidTransportTypes = map[TransportType]bool{
	TransportStdio:          true,
	TransportHTTP:           true,
	TransportSSE:            true,
	TransportStreamableHTTP: true,
}

// Validate checks an MCPEntry against the schema rules.
func Validate(entry *MCPEntry) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Required fields
	validateRequired(entry, result)
	if len(result.Errors) > 0 {
		result.Valid = false
		return result
	}

	// Field format validation
	validateFormats(entry, result)

	// Authorization validation
	validateAuth(entry.Auth, result)

	// Tools validation
	validateTools(entry.Tools, result)

	// Package validation
	validatePackage(entry.Package, result)

	// Runtime validation (optional but validate if present)
	if entry.Runtime != nil {
		validateRuntime(entry.Runtime, result)
	}

	// Optional field warnings
	addWarnings(entry, result)

	result.Valid = len(result.Errors) == 0
	return result
}

func validateRequired(entry *MCPEntry, result *ValidationResult) {
	if entry.Name == "" {
		result.Errors = append(result.Errors, ValidationError{"name", "required field is missing"})
	}
	if entry.Version == "" {
		result.Errors = append(result.Errors, ValidationError{"version", "required field is missing"})
	}
	if entry.Title == "" {
		result.Errors = append(result.Errors, ValidationError{"title", "required field is missing"})
	}
	if entry.Description == "" {
		result.Errors = append(result.Errors, ValidationError{"description", "required field is missing"})
	}
	if entry.Category == "" {
		result.Errors = append(result.Errors, ValidationError{"category", "required field is missing"})
	}
	if entry.Source == "" {
		result.Errors = append(result.Errors, ValidationError{"source", "required field is missing"})
	}
	if entry.Auth == nil {
		result.Errors = append(result.Errors, ValidationError{"authorization", "required field is missing"})
	}
	if len(entry.Tools) == 0 {
		result.Errors = append(result.Errors, ValidationError{"tools", "at least one tool is required"})
	}
	if entry.Package == nil {
		result.Errors = append(result.Errors, ValidationError{"package", "required field is missing"})
	}
}

func validateFormats(entry *MCPEntry, result *ValidationResult) {
	// Name format
	if entry.Name != "" {
		if len(entry.Name) < 2 || len(entry.Name) > 64 {
			result.Errors = append(result.Errors, ValidationError{"name", "must be between 2 and 64 characters"})
		} else if !namePattern.MatchString(entry.Name) {
			result.Errors = append(result.Errors, ValidationError{"name", "must be lowercase letters, numbers, and hyphens only, starting with a letter"})
		}
	}

	// Version format (semver)
	if entry.Version != "" && !versionPattern.MatchString(entry.Version) {
		result.Errors = append(result.Errors, ValidationError{"version", "must be a valid semantic version (e.g., 1.0.0)"})
	}

	// Title length
	if len(entry.Title) > 64 {
		result.Errors = append(result.Errors, ValidationError{"title", "must be 64 characters or less"})
	}

	// Description length
	if entry.Description != "" {
		if len(entry.Description) < 10 {
			result.Errors = append(result.Errors, ValidationError{"description", "must be at least 10 characters"})
		} else if len(entry.Description) > 200 {
			result.Errors = append(result.Errors, ValidationError{"description", "must be 200 characters or less"})
		}
	}

	// Category enum
	if entry.Category != "" && !ValidCategories[entry.Category] {
		result.Errors = append(result.Errors, ValidationError{"category", fmt.Sprintf("invalid category: %s", entry.Category)})
	}

	// Source enum
	if entry.Source != "" && !ValidSources[entry.Source] {
		result.Errors = append(result.Errors, ValidationError{"source", fmt.Sprintf("invalid source: %s", entry.Source)})
	}

	// Color format
	if entry.Color != "" && !colorPattern.MatchString(entry.Color) {
		result.Errors = append(result.Errors, ValidationError{"color", "must be a valid hex color (e.g., #FF0000)"})
	}

	// Tags validation
	if len(entry.Tags) > 10 {
		result.Errors = append(result.Errors, ValidationError{"tags", "maximum 10 tags allowed"})
	}
	for i, tag := range entry.Tags {
		if !namePattern.MatchString(tag) {
			result.Errors = append(result.Errors, ValidationError{fmt.Sprintf("tags[%d]", i), "must be lowercase letters, numbers, and hyphens only"})
		}
	}
}

func validateAuth(auth *Authorization, result *ValidationResult) {
	if auth == nil {
		return
	}

	if !ValidAuthTypes[auth.Type] {
		result.Errors = append(result.Errors, ValidationError{"authorization.type", fmt.Sprintf("invalid auth type: %s", auth.Type)})
		return
	}

	switch auth.Type {
	case AuthAPIKey:
		if auth.EnvVar == "" {
			result.Errors = append(result.Errors, ValidationError{"authorization.env_var", "required for api_key auth type"})
		} else if !envVarPattern.MatchString(auth.EnvVar) {
			result.Errors = append(result.Errors, ValidationError{"authorization.env_var", "must be uppercase letters, numbers, and underscores"})
		}
		if auth.DisplayName == "" {
			result.Errors = append(result.Errors, ValidationError{"authorization.display_name", "required for api_key auth type"})
		}

	case AuthOAuth2:
		if auth.Provider == "" {
			result.Errors = append(result.Errors, ValidationError{"authorization.provider", "required for oauth2 auth type"})
		}
		if auth.OAuth == nil {
			result.Errors = append(result.Errors, ValidationError{"authorization.oauth", "required for oauth2 auth type"})
		} else {
			validateOAuth(auth.OAuth, result)
		}

	case AuthBearerToken:
		if auth.EnvVar == "" {
			result.Errors = append(result.Errors, ValidationError{"authorization.env_var", "required for bearer_token auth type"})
		} else if !envVarPattern.MatchString(auth.EnvVar) {
			result.Errors = append(result.Errors, ValidationError{"authorization.env_var", "must be uppercase letters, numbers, and underscores"})
		}
		if auth.DisplayName == "" {
			result.Errors = append(result.Errors, ValidationError{"authorization.display_name", "required for bearer_token auth type"})
		}

	case AuthCustom:
		if len(auth.EnvVars) == 0 {
			result.Errors = append(result.Errors, ValidationError{"authorization.env_vars", "required for custom auth type"})
		}
		for i, ev := range auth.EnvVars {
			if ev.Name == "" || !envVarPattern.MatchString(ev.Name) {
				result.Errors = append(result.Errors, ValidationError{fmt.Sprintf("authorization.env_vars[%d].name", i), "must be uppercase letters, numbers, and underscores"})
			}
			if ev.DisplayName == "" {
				result.Errors = append(result.Errors, ValidationError{fmt.Sprintf("authorization.env_vars[%d].display_name", i), "required"})
			}
		}
	}
}

func validateOAuth(oauth *OAuthConfig, result *ValidationResult) {
	if oauth.AuthorizationURL == "" {
		result.Errors = append(result.Errors, ValidationError{"authorization.oauth.authorization_url", "required"})
	}
	if oauth.TokenURL == "" {
		result.Errors = append(result.Errors, ValidationError{"authorization.oauth.token_url", "required"})
	}
	if len(oauth.Scopes) == 0 {
		result.Errors = append(result.Errors, ValidationError{"authorization.oauth.scopes", "at least one scope is required"})
	}
	if oauth.TokenEnv == "" {
		result.Errors = append(result.Errors, ValidationError{"authorization.oauth.token_env", "required"})
	} else if !envVarPattern.MatchString(oauth.TokenEnv) {
		result.Errors = append(result.Errors, ValidationError{"authorization.oauth.token_env", "must be uppercase letters, numbers, and underscores"})
	}
}

func validateTools(tools []Tool, result *ValidationResult) {
	seenNames := make(map[string]bool)

	for i, tool := range tools {
		prefix := fmt.Sprintf("tools[%d]", i)

		// Required fields
		if tool.Name == "" {
			result.Errors = append(result.Errors, ValidationError{prefix + ".name", "required"})
		} else {
			if !toolNamePattern.MatchString(tool.Name) {
				result.Errors = append(result.Errors, ValidationError{prefix + ".name", "must be snake_case (lowercase letters, numbers, underscores)"})
			}
			if seenNames[tool.Name] {
				result.Errors = append(result.Errors, ValidationError{prefix + ".name", fmt.Sprintf("duplicate tool name: %s", tool.Name)})
			}
			seenNames[tool.Name] = true
		}

		if tool.Description == "" {
			result.Errors = append(result.Errors, ValidationError{prefix + ".description", "required"})
		} else if len(tool.Description) < 10 {
			result.Errors = append(result.Errors, ValidationError{prefix + ".description", "must be at least 10 characters"})
		}

		if tool.InputSchema == nil {
			result.Errors = append(result.Errors, ValidationError{prefix + ".inputSchema", "required"})
		} else if tool.InputSchema.Type != "object" {
			result.Errors = append(result.Errors, ValidationError{prefix + ".inputSchema.type", "must be 'object'"})
		}
	}
}

func validatePackage(pkg *Package, result *ValidationResult) {
	if pkg == nil {
		return
	}

	if !ValidPackageTypes[pkg.Type] {
		result.Errors = append(result.Errors, ValidationError{"package.type", fmt.Sprintf("invalid package type: %s", pkg.Type)})
		return
	}

	switch pkg.Type {
	case PackageNPM:
		if pkg.Name == "" {
			result.Errors = append(result.Errors, ValidationError{"package.name", "required for npm package"})
		}

	case PackagePyPI:
		if pkg.Name == "" {
			result.Errors = append(result.Errors, ValidationError{"package.name", "required for pypi package"})
		}

	case PackageDocker:
		if pkg.Image == "" {
			result.Errors = append(result.Errors, ValidationError{"package.image", "required for docker package"})
		}

	case PackageBinary:
		if len(pkg.Platforms) == 0 {
			result.Errors = append(result.Errors, ValidationError{"package.platforms", "required for binary package"})
		}
		for platform, bin := range pkg.Platforms {
			if bin.URL == "" {
				result.Errors = append(result.Errors, ValidationError{fmt.Sprintf("package.platforms.%s.url", platform), "required"})
			}
			if bin.SHA256 != "" && !sha256Pattern.MatchString(bin.SHA256) {
				result.Errors = append(result.Errors, ValidationError{fmt.Sprintf("package.platforms.%s.sha256", platform), "invalid SHA256 hash"})
			}
		}

	case PackageWASM:
		if pkg.URL == "" && pkg.LocalPath == "" {
			result.Errors = append(result.Errors, ValidationError{"package", "either url or local_path is required for wasm package"})
		}
		if pkg.SHA256 != "" && !sha256Pattern.MatchString(pkg.SHA256) {
			result.Errors = append(result.Errors, ValidationError{"package.sha256", "invalid SHA256 hash"})
		}
	}
}

func validateRuntime(runtime *Runtime, result *ValidationResult) {
	if runtime.Transport != "" && !ValidTransportTypes[runtime.Transport] {
		result.Errors = append(result.Errors, ValidationError{"runtime.transport", fmt.Sprintf("invalid transport type: %s", runtime.Transport)})
	}

	if runtime.Timeout != 0 && runtime.Timeout < 1000 {
		result.Errors = append(result.Errors, ValidationError{"runtime.timeout", "must be at least 1000ms"})
	}
}

func addWarnings(entry *MCPEntry, result *ValidationResult) {
	if entry.About == "" {
		result.Warnings = append(result.Warnings, ValidationError{"about", "recommended: add markdown documentation"})
	}
	if entry.Icon == "" {
		result.Warnings = append(result.Warnings, ValidationError{"icon", "recommended: add an icon"})
	}
	if entry.Homepage == "" && entry.Repository == "" {
		result.Warnings = append(result.Warnings, ValidationError{"homepage/repository", "recommended: add a homepage or repository URL"})
	}
}

// ValidateFile reads and validates a JSON file.
func ValidateFile(path string) (*ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var entry MCPEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "json",
				Message: fmt.Sprintf("invalid JSON: %v", err),
			}},
		}, nil
	}

	return Validate(&entry), nil
}

// ValidateDirectory validates all JSON files in a directory.
func ValidateDirectory(dir string) (map[string]*ValidationResult, error) {
	results := make(map[string]*ValidationResult)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, file.Name())
		result, err := ValidateFile(path)
		if err != nil {
			results[file.Name()] = &ValidationResult{
				Valid: false,
				Errors: []ValidationError{{
					Field:   "file",
					Message: err.Error(),
				}},
			}
		} else {
			results[file.Name()] = result
		}
	}

	return results, nil
}
