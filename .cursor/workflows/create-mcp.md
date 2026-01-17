# Create New MCP

Create a new MCP registry entry from a GitHub repository, documentation, or description.

## Input

Provide ONE of the following:

1. **GitHub Repository URL** - e.g., `https://github.com/anthropics/mcp-server-brave-search`
2. **Documentation URL** - Link to MCP documentation or README
3. **Text Description** - Plain text describing the MCP and its tools

## Workflow Steps

### Step 1: Gather Information

<step>
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
2. What does it do? (one-line description)
3. What category does it belong to?
4. What authentication does it require?
5. What tools does it expose?
6. How is it installed? (npm, pypi, docker, etc.)
</step>

### Step 2: Determine Authentication

<step>
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
</step>

### Step 3: Define Tools

<step>
For each tool/function the MCP exposes, create a tool definition:

```json
{
  "name": "tool_name_snake_case",
  "title": "Human Readable Title",
  "description": "Clear description of what this tool does",
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
    "openWorldHint": true/false,
    "requiresApproval": true/false
  }
}
```

**Annotation Guidelines:**
- `readOnlyHint: true` - Tool only reads data
- `destructiveHint: true` - Tool can delete/modify data
- `openWorldHint: true` - Tool accesses external services
- `requiresApproval: true` - Should prompt user before executing
</step>

### Step 4: Create Registry JSON

<step>
Create the complete registry JSON file at `appdata/registry/{name}.json`:

```json
{
  "$schema": "../schemas/mcp-registry.schema.json",
  "name": "{name}",
  "version": "1.0.0",
  "title": "{Title}",
  "description": "{One-line description, 10-200 chars}",
  "category": "{category}",
  "source": "community",
  "tags": ["{tag1}", "{tag2}"],
  "icon": "/mcp-logos/{name}.svg",
  "color": "#{hexcolor}",
  "about": "{Markdown documentation}",
  "homepage": "{url}",
  "repository": "{github_url}",
  "authorization": {
    // Auth config based on Step 2
  },
  "tools": [
    // Tool definitions from Step 3
  ],
  "package": {
    // Package config
  },
  "runtime": {
    "transport": "stdio",
    "command": "{command}",
    "args": ["{args}"]
  },
  "metadata": {
    "author": "{author}",
    "license": "{license}"
  }
}
```
</step>

### Step 5: Create Icon Placeholder

<step>
Note that an icon is needed at `desktop/public/mcp-logos/{name}.svg`.

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
</step>

### Step 6: Validate

<step>
Run validation to ensure the new MCP entry is correct:

```bash
./validate-registry.exe appdata/registry/{name}.json
```

Fix any validation errors before completing.
</step>

## Output

After completing the workflow, provide:

1. ✅ The created JSON file path
2. ✅ Summary of tools added
3. ⚠️ Any missing information that needs manual completion
4. 📋 Icon status (created or TODO)

## Example Usage

**Input:** `https://github.com/anthropics/mcp-server-brave-search`

**Output:**
- Created: `appdata/registry/brave-search.json`
- Tools: `brave_web_search`, `brave_local_search`
- Auth: API key (`BRAVE_API_KEY`)
- Icon: TODO - need to add `desktop/public/mcp-logos/brave-search.svg`
