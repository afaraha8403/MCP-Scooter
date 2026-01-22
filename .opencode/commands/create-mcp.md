# Create New MCP

Create a new MCP registry entry from a GitHub repository, documentation, or description.

## Input

Provide ONE of the following:

1. **GitHub Repository URL** - e.g., `https://github.com/anthropics/mcp-server-brave-search`
2. **Documentation URL** - Link to MCP documentation or README
3. **Text Description** - Plain text describing the MCP and its tools

## Workflow Steps

### Step 1: Gather Information

Based on the input provided, I need to gather the following information:

**If GitHub URL provided:**
- Fetch the README.md to understand the MCP
- Look for package.json or pyproject.toml for package info
- Identify the tools/functions exposed
- Find authentication requirements

**If Documentation/Text provided:**
- Extract the MCP name and description
- Identify all tools and their parameters
- Determine authentication method (API key, OAuth, none)
- Find the package installation method

**Questions to answer:**
1. What is the unique name for this MCP? (lowercase, hyphens)
2. What does it do? (one-line description, 10-200 chars)
3. What category does it belong to?
4. What authentication does it require?
5. What tools does it expose?
6. How is it installed? (npm, pypi, cargo, docker, wasm, binary)
7. What transport does it use? (stdio, http, sse, streamable-http)

### Step 2: Determine Authentication

Based on the gathered information, determine the auth type:

- **No auth** → `"type": "none"`
- **API Key** → `"type": "api_key"` with `env_var`, `display_name`, `help_url`
- **OAuth 2.0** → `"type": "oauth2"` with provider config
- **Personal Access Token** → `"type": "bearer_token"`
- **Database/Custom** → `"type": "custom"` with `env_vars` array

For API key auth, find:
- Where to get the key (help_url)
- What to name the environment variable (SCREAMING_SNAKE_CASE)
- Any validation pattern for the key format

### Step 3: Define Tools

For each tool/function the MCP exposes, create a tool definition:

```json
{
  "name": "tool_name_snake_case",
  "title": "Human Readable Title",
  "description": "Clear description of what this tool does (min 10 chars, max 500)",
  "inputSchema": {
    "type": "object",
    "properties": {
      // Define each parameter
    },
    "required": ["required_params"]
  },
  "annotations": {
    "readOnlyHint": true/false,
    "destructiveHint": true/false,
    "idempotentHint": true/false,
    "openWorldHint": true/false,
    "requiresApproval": true/false,
    "rateLimit": "100 requests/minute",
    "costPerCall": "$0.001"
  }
}
```

**Annotation Guidelines:**

| Annotation | When to use |
|------------|-------------|
| `readOnlyHint: true` | Tool only reads data |
| `destructiveHint: true` | Tool can delete/modify data |
| `idempotentHint: true` | Repeated calls produce same result |
| `openWorldHint: true` | Tool accesses external services |
| `requiresApproval: true` | Should prompt user before executing |
| `rateLimit` | Document API rate limits (optional) |
| `costPerCall` | Document cost per invocation (optional) |

### Step 4: Create Registry JSON

Create the complete registry JSON file at `appdata/registry/official/{name}.json`:

```json
{
  "$schema": "../../schemas/mcp-registry.schema.json",
  "name": "{name}",
  "version": "1.0.0",
  "title": "{Title}",
  "description": "{One-line description, 10-200 chars}",
  "category": "{category}",
  "source": "official",
  "tags": ["{tag1}", "{tag2}"],
  "icon": "/registry-logos/{name}.svg",
  "color": "#{hexcolor}",
  "about": "{Markdown documentation}",
  "homepage": "{url}",
  "repository": "{github_url}",
  "documentation": "{api_docs_url}",
  "authorization": {
    // Auth config based on Step 2
  },
  "tools": [
    // Tool definitions from Step 3
  ],
  "package": {
    // Package config (see below)
  },
  "runtime": {
    // Runtime config (see below)
  },
  "metadata": {
    "author": "{author}",
    "license": "{license}"
  }
}
```

**About Field Guidelines:**
- Do NOT include manual setup instructions for other IDEs (Cursor, Claude, Zed, etc.).
- MCP Scooter handles the connection; focusing on the MCP's features and purpose is enough.
- Include features, getting started (scooter_add), and any specific prerequisites.

**Package Configuration by Type:**

```json
// npm
{ "type": "npm", "name": "@scope/package", "version": "^1.0.0" }

// pypi
{ "type": "pypi", "name": "package-name", "version": ">=1.0.0" }

// cargo (Rust)
{ "type": "cargo", "name": "crate-name" }

// wasm
{ "type": "wasm", "url": "https://...", "sha256": "..." }

// docker
{ "type": "docker", "image": "image:tag" }

// binary
{ "type": "binary", "platforms": { "darwin-arm64": { "url": "...", "sha256": "..." } } }
```

**Runtime Configuration by Transport:**

```json
// stdio (most common)
{ "transport": "stdio", "command": "npx", "args": ["-y", "@scope/package"] }

// http
{ "transport": "http", "command": "python", "args": ["-m", "mcp_server"] }

// sse (Server-Sent Events)
{ "transport": "sse", "command": "node", "args": ["server.js"] }

// streamable-http
{ "transport": "streamable-http", "command": "...", "args": [...] }
```

### Step 5: Create Icon Placeholder

Note that an icon is needed at `desktop/public/registry-logos/{name}.svg`.

Options:
1. Download from the project's repository if available
2. Create a simple placeholder SVG
3. Note it as TODO for manual addition

If creating a placeholder, use this template:

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
  <rect x="3" y="3" width="18" height="18" rx="2"/>
  <text x="12" y="16" text-anchor="middle" font-size="10" fill="currentColor">{initial}</text>
</svg>
```

### Step 6: Validate

Run validation to ensure the new MCP entry is correct:

```bash
# Validate the new file
go run cmd/validate-registry/main.go appdata/registry/official/{name}.json

# Or use make to validate all
make validate
```

Fix any validation errors before completing.

**Common Validation Errors:**
- `description` too short (min 10 chars) or too long (max 200 chars)
- `name` contains uppercase or invalid characters
- Missing required fields in `authorization` for the chosen type
- Tool `description` too short (min 10 chars)

## Output

After completing the workflow, provide:

1. ✅ The created JSON file path
2. ✅ Summary of tools added
3. ⚠️ Any missing information that needs manual completion
4. 📋 Icon status (created or TODO)

## Example Usage

**Input:** `https://github.com/anthropics/mcp-server-brave-search`

**Output:**
- Created: `appdata/registry/official/brave-search.json`
- Tools: `brave_web_search`, `brave_local_search`
- Auth: API key (`BRAVE_API_KEY`)
- Icon: TODO - need to add `desktop/public/registry-logos/brave-search.svg`
