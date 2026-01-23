// Package registry provides types and validation for MCP registry entries.
package registry

// MCPEntry represents a complete MCP server definition in the registry.
type MCPEntry struct {
	Schema      string         `json:"$schema,omitempty"`
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Category    Category       `json:"category"`
	Source      Source         `json:"source"`
	Tags        []string       `json:"tags,omitempty"`
	Icon        string         `json:"icon,omitempty"`
	IconBackground *IconBackground `json:"icon_background,omitempty"`
	Banner      string         `json:"banner,omitempty"`
	Color       string         `json:"color,omitempty"`
	About       string         `json:"about,omitempty"`
	Homepage    string         `json:"homepage,omitempty"`
	Repository  string         `json:"repository,omitempty"`
	Docs        string         `json:"documentation,omitempty"`
	Auth        *Authorization `json:"authorization"`
	Tools       []Tool         `json:"tools"`
	Package     *Package       `json:"package"`
	Runtime     *Runtime       `json:"runtime,omitempty"`
	Metadata    *Metadata      `json:"metadata,omitempty"`
}

// Category defines the primary classification of an MCP.
type Category string

const (
	CategoryDevelopment   Category = "development"
	CategoryDatabase      Category = "database"
	CategoryProductivity  Category = "productivity"
	CategoryCommunication Category = "communication"
	CategorySearch        Category = "search"
	CategoryCloud         Category = "cloud"
	CategoryAnalytics     Category = "analytics"
	CategoryAI            Category = "ai"
	CategoryUtility       Category = "utility"
	CategoryCustom        Category = "custom"
)

// Source indicates where the MCP comes from.
type Source string

const (
	SourceOfficial   Source = "official"
	SourceCommunity  Source = "community"
	SourceEnterprise Source = "enterprise"
	SourceLocal      Source = "local"
	SourceCustom     Source = "custom"
)

// AuthType defines the authentication method.
type AuthType string

const (
	AuthNone        AuthType = "none"
	AuthAPIKey      AuthType = "api_key"
	AuthOAuth2      AuthType = "oauth2"
	AuthBearerToken AuthType = "bearer_token"
	AuthCustom      AuthType = "custom"
)

// Authorization defines how the MCP authenticates.
type Authorization struct {
	Type        AuthType       `json:"type"`
	Required    bool           `json:"required,omitempty"`
	Recommended bool           `json:"recommended,omitempty"`
	EnvVar      string         `json:"env_var,omitempty"`
	DisplayName string         `json:"display_name,omitempty"`
	Description string         `json:"description,omitempty"`
	HelpURL     string         `json:"help_url,omitempty"`
	Validation  *KeyValidation `json:"validation,omitempty"`
	Provider    string         `json:"provider,omitempty"`
	OAuth       *OAuthConfig   `json:"oauth,omitempty"`
	Scopes      []string       `json:"scopes,omitempty"`
	EnvVars     []EnvVarDef    `json:"env_vars,omitempty"`
}

// KeyValidation defines validation rules for API keys.
type KeyValidation struct {
	Pattern      string `json:"pattern,omitempty"`
	TestEndpoint string `json:"test_endpoint,omitempty"`
}

// OAuthConfig defines OAuth 2.0 configuration.
type OAuthConfig struct {
	AuthorizationURL  string            `json:"authorization_url"`
	TokenURL          string            `json:"token_url"`
	Scopes            []string          `json:"scopes"`
	ScopeDescriptions map[string]string `json:"scope_descriptions,omitempty"`
	PKCERequired      bool              `json:"pkce_required,omitempty"`
	ClientIDEnv       string            `json:"client_id_env,omitempty"`
	ClientSecretEnv   string            `json:"client_secret_env,omitempty"`
	TokenEnv          string            `json:"token_env"`
	RefreshTokenEnv   string            `json:"refresh_token_env,omitempty"`
}

// EnvVarDef defines a custom environment variable for auth.
type EnvVarDef struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description,omitempty"`
	Secret      bool     `json:"secret,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// Tool represents a single tool/function exposed by the MCP.
type Tool struct {
	Name         string                 `json:"name"`
	Title        string                 `json:"title,omitempty"`
	Description  string                 `json:"description"`
	InputSchema  *JSONSchema            `json:"inputSchema"`
	OutputSchema *JSONSchema            `json:"outputSchema,omitempty"`
	SampleInput  map[string]interface{} `json:"sampleInput,omitempty"`
	Annotations  *ToolAnnotations       `json:"annotations,omitempty"`
}

// JSONSchema represents a JSON Schema for tool input/output.
type JSONSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]PropertySchema `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
	Items      *PropertySchema           `json:"items,omitempty"`
}

// PropertySchema defines a single property in a JSON Schema.
type PropertySchema struct {
	Type        string          `json:"type,omitempty"`
	Description string          `json:"description,omitempty"`
	Default     interface{}     `json:"default,omitempty"`
	Enum        []string        `json:"enum,omitempty"`
	Minimum     *int            `json:"minimum,omitempty"`
	Maximum     *int            `json:"maximum,omitempty"`
	MinLength   *int            `json:"minLength,omitempty"`
	MaxLength   *int            `json:"maxLength,omitempty"`
	Items       *PropertySchema `json:"items,omitempty"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"`
}

// ToolAnnotations provides hints about tool behavior.
type ToolAnnotations struct {
	ReadOnlyHint     bool   `json:"readOnlyHint,omitempty"`
	DestructiveHint  bool   `json:"destructiveHint,omitempty"`
	IdempotentHint   bool   `json:"idempotentHint,omitempty"`
	OpenWorldHint    bool   `json:"openWorldHint,omitempty"`
	RequiresApproval bool   `json:"requiresApproval,omitempty"`
	RateLimit        string `json:"rateLimit,omitempty"`
	CostPerCall      string `json:"costPerCall,omitempty"`
}

// IconBackground defines custom background colors for the icon.
type IconBackground struct {
	Light string `json:"light,omitempty"`
	Dark  string `json:"dark,omitempty"`
}

// PackageType defines how the MCP is distributed.
type PackageType string

const (
	PackageNPM    PackageType = "npm"
	PackagePyPI   PackageType = "pypi"
	PackageCargo  PackageType = "cargo"
	PackageWASM   PackageType = "wasm"
	PackageDocker PackageType = "docker"
	PackageBinary PackageType = "binary"
)

// Package defines how to install/obtain the MCP.
type Package struct {
	Type      PackageType           `json:"type"`
	Name      string                `json:"name,omitempty"`
	Version   string                `json:"version,omitempty"`
	Registry  string                `json:"registry,omitempty"`
	Index     string                `json:"index,omitempty"`
	URL       string                `json:"url,omitempty"`
	LocalPath string                `json:"local_path,omitempty"`
	SHA256    string                `json:"sha256,omitempty"`
	Image     string                `json:"image,omitempty"`
	Platforms map[string]PlatformBinary `json:"platforms,omitempty"`
}

// PlatformBinary defines a binary download for a specific platform.
type PlatformBinary struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256,omitempty"`
}

// TransportType defines how Scooter communicates with the MCP.
type TransportType string

const (
	TransportStdio          TransportType = "stdio"
	TransportHTTP           TransportType = "http"
	TransportSSE            TransportType = "sse"
	TransportStreamableHTTP TransportType = "streamable-http"
)

// Runtime defines how to execute the MCP server.
type Runtime struct {
	Transport   TransportType     `json:"transport,omitempty"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Cwd         *string           `json:"cwd,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	HealthCheck *HealthCheck      `json:"healthCheck,omitempty"`
}

// HealthCheck defines health monitoring configuration.
type HealthCheck struct {
	Enabled  bool `json:"enabled,omitempty"`
	Interval int  `json:"interval,omitempty"`
}

// Metadata provides additional attribution information.
type Metadata struct {
	Author             string   `json:"author,omitempty"`
	License            string   `json:"license,omitempty"`
	Maintainers        []string `json:"maintainers,omitempty"`
	Created            string   `json:"created,omitempty"`
	Updated            string   `json:"updated,omitempty"`
	Deprecated         bool     `json:"deprecated,omitempty"`
	DeprecationMessage *string  `json:"deprecation_message,omitempty"`
	MinScooterVersion  string   `json:"minimum_scooter_version,omitempty"`
	VerifiedAt         string   `json:"verified_at,omitempty"`
}
