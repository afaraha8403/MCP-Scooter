# MCP Registry JSON Specification

**Version:** 1.0  
**Status:** Draft  
**Date:** January 16, 2026  
**Related:** PRD v2.9

---

## 1. Overview

This document defines the JSON schema for MCP (Model Context Protocol) server entries stored in the `appdata/registry/` folder. These JSON files serve as the metadata manifests that allow MCP Scout to:

- **Discover** available MCP tools in the registry
- **Display** tool information in the Scout Store UI
- **Configure** authentication requirements
- **Install** and manage MCP servers dynamically
- **Inject** proper environment variables and credentials

---

## 2. Design Philosophy

Based on the PRD's core promise of "zero latency and <50MB RAM usage," the registry schema must be:

1. **Minimal by Default** — Only required fields for discovery
2. **Extensible** — Optional fields for rich functionality
3. **Human-Readable** — Clear markdown descriptions
4. **Machine-Parseable** — Valid JSON Schema for validation
5. **Profile-Aware** — Support Scout's Identity Management system

---

## 3. Complete JSON Schema

```json
{
  "$schema": "https://mcp-scooter.com/schemas/registry/v1.json",
  
  // ═══════════════════════════════════════════════════════════════════
  // SECTION 1: IDENTITY (Required)
  // ═══════════════════════════════════════════════════════════════════
  "name": "string",
  "version": "string",
  "title": "string",
  "description": "string",
  
  // ═══════════════════════════════════════════════════════════════════
  // SECTION 2: CLASSIFICATION (Required)
  // ═══════════════════════════════════════════════════════════════════
  "category": "string",
  "source": "string",
  "tags": ["string"],
  
  // ═══════════════════════════════════════════════════════════════════
  // SECTION 3: PRESENTATION (Optional)
  // ═══════════════════════════════════════════════════════════════════
  "icon": "string",
  "banner": "string",
  "color": "string",
  
  // ═══════════════════════════════════════════════════════════════════
  // SECTION 4: DOCUMENTATION (Recommended)
  // ═══════════════════════════════════════════════════════════════════
  "about": "string (markdown)",
  "homepage": "string (URL)",
  "repository": "string (URL)",
  "documentation": "string (URL)",
  
  // ═══════════════════════════════════════════════════════════════════
  // SECTION 5: AUTHORIZATION (Required if auth needed)
  // ═══════════════════════════════════════════════════════════════════
  "authorization": {
    "type": "none | api_key | oauth2 | bearer_token | custom",
    "required": true,
    // ... type-specific fields
  },
  
  // ═══════════════════════════════════════════════════════════════════
  // SECTION 6: TOOLS (Required)
  // ═══════════════════════════════════════════════════════════════════
  "tools": [
    {
      "name": "string",
      "description": "string",
      "inputSchema": {},
      "outputSchema": {},
      "annotations": {}
    }
  ],
  
  // ═══════════════════════════════════════════════════════════════════
  // SECTION 7: INSTALLATION (Required)
  // ═══════════════════════════════════════════════════════════════════
  "package": {
    "type": "npm | pypi | cargo | wasm | docker | binary",
    // ... type-specific fields
  },
  
  // ═══════════════════════════════════════════════════════════════════
  // SECTION 8: RUNTIME (Optional)
  // ═══════════════════════════════════════════════════════════════════
  "runtime": {
    "transport": "stdio | http | sse | streamable-http",
    "command": "string",
    "args": ["string"],
    "env": {}
  },
  
  // ═══════════════════════════════════════════════════════════════════
  // SECTION 9: METADATA (Optional)
  // ═══════════════════════════════════════════════════════════════════
  "metadata": {
    "author": "string",
    "license": "string",
    "maintainers": ["string"],
    "created": "ISO8601",
    "updated": "ISO8601"
  }
}
```

---

## 4. Field Definitions

### 4.1 Identity Fields (Required)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Unique identifier. Use lowercase with hyphens (e.g., `brave-search`, `postgres-mcp`). This is what users type in `scout_add("name")`. |
| `version` | string | ✅ | Semantic version (e.g., `1.0.0`, `2.3.1`). Used for update detection. |
| `title` | string | ✅ | Human-friendly display name (e.g., "Brave Search", "PostgreSQL Explorer"). |
| `description` | string | ✅ | One-line summary (max 120 chars). Shown in search results and tool cards. |

**Example:**
```json
{
  "name": "brave-search",
  "version": "1.2.0",
  "title": "Brave Search",
  "description": "Search the web privately using Brave Search API"
}
```

---

### 4.2 Classification Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `category` | string | ✅ | Primary category for filtering. See [Category Taxonomy](#category-taxonomy). |
| `source` | string | ✅ | Origin of the tool: `official`, `community`, `enterprise`, `local`. |
| `tags` | string[] | ⬜ | Additional searchable keywords (max 10). |

**Category Taxonomy:**
- `development` — Code, Git, IDE integrations
- `database` — SQL, NoSQL, data warehouses
- `productivity` — Task management, calendars, notes
- `communication` — Email, Slack, Discord
- `search` — Web search, knowledge bases
- `cloud` — AWS, GCP, Azure services
- `analytics` — Data analysis, visualization
- `ai` — Other AI/ML services
- `utility` — General purpose tools
- `custom` — User-created local tools

**Example:**
```json
{
  "category": "search",
  "source": "community",
  "tags": ["web-search", "privacy", "brave", "api"]
}
```

---

### 4.3 Presentation Fields (Optional)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `icon` | string | ⬜ | Path to SVG icon. Relative: `/mcp-logos/name.svg`. Or absolute URL. |
| `banner` | string | ⬜ | Path to banner image for detail view. |
| `color` | string | ⬜ | Brand color in hex (e.g., `#FB542B` for Brave). Used for UI accents. |

**Example:**
```json
{
  "icon": "/mcp-logos/brave-search.svg",
  "color": "#FB542B"
}
```

---

### 4.4 Documentation Fields (Recommended)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `about` | string | ⬜ | **Markdown** explanation of the MCP. Supports full GFM. |
| `homepage` | string | ⬜ | Official product/service homepage. |
| `repository` | string | ⬜ | Source code repository URL. |
| `documentation` | string | ⬜ | Link to detailed API docs. |

**The `about` Field:**

This is a **markdown string** that provides comprehensive information about the MCP. It should include:

- What the tool does
- Key features and capabilities
- Usage examples
- Important notes or limitations
- Links to additional resources

**Example:**
```json
{
  "about": "## Brave Search MCP\n\nSearch the web privately using the Brave Search API.\n\n### Features\n\n- **Web Search**: Full web search with result snippets\n- **News Search**: Filter for recent news articles\n- **Image Search**: Find images across the web\n- **Privacy-First**: No tracking, no profiling\n\n### Requirements\n\nRequires a Brave Search API key. Get one at [brave.com/search/api](https://brave.com/search/api).\n\n### Usage\n\n```\nscout_add(\"brave-search\")\n```\n\nThen use `brave_web_search` to search the web.",
  "homepage": "https://brave.com/search",
  "repository": "https://github.com/anthropics/brave-search-mcp",
  "documentation": "https://api.search.brave.com/docs"
}
```

---

### 4.5 Authorization Object (Critical)

The `authorization` field defines how the MCP authenticates. This is critical for Scout's Profile system to properly inject credentials.

#### Authorization Types:

##### Type: `none`
No authentication required. Tool is publicly accessible.

```json
{
  "authorization": {
    "type": "none"
  }
}
```

##### Type: `api_key`
Requires a static API key provided by the user.

```json
{
  "authorization": {
    "type": "api_key",
    "required": true,
    "env_var": "BRAVE_API_KEY",
    "display_name": "Brave Search API Key",
    "description": "Your Brave Search API key from brave.com/search/api",
    "help_url": "https://brave.com/search/api/register",
    "validation": {
      "pattern": "^BSA[a-zA-Z0-9]{32}$",
      "test_endpoint": "https://api.search.brave.com/res/v1/web/search?q=test"
    }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `env_var` | string | Environment variable name Scout injects at runtime |
| `display_name` | string | Human-friendly name shown in UI |
| `description` | string | Help text explaining what this key is |
| `help_url` | string | Link to get/generate the API key |
| `validation.pattern` | string | Regex to validate key format (optional) |
| `validation.test_endpoint` | string | URL to verify key works (optional) |

##### Type: `oauth2`
Uses OAuth 2.0/2.1 flow. Scout handles the entire flow per PRD Section 4.2.

```json
{
  "authorization": {
    "type": "oauth2",
    "required": true,
    "provider": "google",
    "display_name": "Google Account",
    "description": "Connect your Google account to access Drive files",
    "oauth": {
      "authorization_url": "https://accounts.google.com/o/oauth2/v2/auth",
      "token_url": "https://oauth2.googleapis.com/token",
      "scopes": [
        "https://www.googleapis.com/auth/drive.readonly"
      ],
      "scope_descriptions": {
        "https://www.googleapis.com/auth/drive.readonly": "View files in your Google Drive"
      },
      "pkce_required": true,
      "client_id_env": "GOOGLE_CLIENT_ID",
      "client_secret_env": "GOOGLE_CLIENT_SECRET",
      "token_env": "GOOGLE_ACCESS_TOKEN",
      "refresh_token_env": "GOOGLE_REFRESH_TOKEN"
    }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `provider` | string | Known provider for UI hints: `google`, `github`, `slack`, `microsoft`, `custom` |
| `oauth.authorization_url` | string | OAuth authorization endpoint |
| `oauth.token_url` | string | Token exchange endpoint |
| `oauth.scopes` | string[] | Required OAuth scopes |
| `oauth.scope_descriptions` | object | Human-readable scope explanations |
| `oauth.pkce_required` | boolean | If true, use PKCE flow (RFC 7636) |
| `oauth.client_id_env` | string | Env var for OAuth client ID |
| `oauth.client_secret_env` | string | Env var for OAuth client secret |
| `oauth.token_env` | string | Env var Scout injects access token into |
| `oauth.refresh_token_env` | string | Env var for refresh token |

##### Type: `bearer_token`
Static bearer token (similar to API key but used in Authorization header).

```json
{
  "authorization": {
    "type": "bearer_token",
    "required": true,
    "env_var": "GITHUB_TOKEN",
    "display_name": "GitHub Personal Access Token",
    "description": "A GitHub PAT with repo scope",
    "help_url": "https://github.com/settings/tokens",
    "scopes": ["repo", "read:user"]
  }
}
```

##### Type: `custom`
For non-standard authentication methods.

```json
{
  "authorization": {
    "type": "custom",
    "required": true,
    "display_name": "Database Connection",
    "description": "PostgreSQL connection string",
    "env_vars": [
      {
        "name": "DATABASE_URL",
        "display_name": "Connection String",
        "description": "postgres://user:pass@host:port/db",
        "secret": true
      },
      {
        "name": "SSL_MODE",
        "display_name": "SSL Mode",
        "description": "require, prefer, or disable",
        "secret": false,
        "default": "require"
      }
    ]
  }
}
```

---

### 4.6 Tools Array (Required)

The `tools` array lists all tools/functions this MCP server exposes. This enables:
- Scout Store to display capabilities before installation
- `scout_find` to search across tool descriptions
- Profile `allow_tools` to filter specific tools

**Tool Object Schema:**

```json
{
  "tools": [
    {
      "name": "brave_web_search",
      "title": "Web Search",
      "description": "Search the web using Brave Search",
      "inputSchema": {
        "type": "object",
        "properties": {
          "query": {
            "type": "string",
            "description": "Search query"
          },
          "count": {
            "type": "integer",
            "description": "Number of results (1-20)",
            "default": 10,
            "minimum": 1,
            "maximum": 20
          },
          "freshness": {
            "type": "string",
            "description": "Time filter",
            "enum": ["day", "week", "month", "year"]
          }
        },
        "required": ["query"]
      },
      "outputSchema": {
        "type": "object",
        "properties": {
          "results": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "title": { "type": "string" },
                "url": { "type": "string" },
                "snippet": { "type": "string" }
              }
            }
          }
        }
      },
      "annotations": {
        "readOnlyHint": true,
        "destructiveHint": false,
        "idempotentHint": true,
        "openWorldHint": true,
        "rateLimit": "1 req/sec",
        "costPerCall": "$0.005"
      }
    }
  ]
}
```

**Tool Field Definitions:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Unique tool identifier. Use snake_case. |
| `title` | string | ⬜ | Human-friendly name for UI. |
| `description` | string | ✅ | What this tool does. Be specific for LLM discovery. |
| `inputSchema` | object | ✅ | JSON Schema defining input parameters. |
| `outputSchema` | object | ⬜ | JSON Schema defining output structure. |
| `annotations` | object | ⬜ | Hints for safety and behavior. |

**Annotation Fields:**

| Annotation | Type | Description |
|------------|------|-------------|
| `readOnlyHint` | boolean | True if tool only reads data |
| `destructiveHint` | boolean | True if tool can delete/modify data |
| `idempotentHint` | boolean | True if repeated calls produce same result |
| `openWorldHint` | boolean | True if tool accesses external services |
| `requiresApproval` | boolean | True if Scout should prompt for Human-in-the-Loop approval |
| `rateLimit` | string | Human-readable rate limit info |
| `costPerCall` | string | Estimated cost per invocation |

---

### 4.7 Package Object (Required)

Defines how to install/obtain the MCP server.

#### Package Type: `npm`

```json
{
  "package": {
    "type": "npm",
    "name": "@anthropic/brave-search-mcp",
    "version": "^1.2.0",
    "registry": "https://registry.npmjs.org"
  }
}
```

#### Package Type: `pypi`

```json
{
  "package": {
    "type": "pypi",
    "name": "brave-search-mcp",
    "version": ">=1.2.0",
    "index": "https://pypi.org/simple"
  }
}
```

#### Package Type: `wasm`

Per PRD Section 7.2, Scout uses wazero for WASM isolation.

```json
{
  "package": {
    "type": "wasm",
    "url": "https://mcp-scooter.com/wasm/brave-search.wasm",
    "local_path": "/wasm/brave-search.wasm",
    "sha256": "abc123..."
  }
}
```

#### Package Type: `docker`

```json
{
  "package": {
    "type": "docker",
    "image": "anthropic/brave-search-mcp:latest",
    "registry": "docker.io"
  }
}
```

#### Package Type: `binary`

For pre-compiled native binaries.

```json
{
  "package": {
    "type": "binary",
    "platforms": {
      "windows-x64": {
        "url": "https://github.com/.../releases/download/v1.0/tool-win-x64.exe",
        "sha256": "..."
      },
      "darwin-arm64": {
        "url": "https://github.com/.../releases/download/v1.0/tool-darwin-arm64",
        "sha256": "..."
      },
      "linux-x64": {
        "url": "https://github.com/.../releases/download/v1.0/tool-linux-x64",
        "sha256": "..."
      }
    }
  }
}
```

---

### 4.8 Runtime Object (Optional)

Defines how Scout should execute the MCP server.

```json
{
  "runtime": {
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@anthropic/brave-search-mcp"],
    "env": {
      "NODE_ENV": "production"
    },
    "cwd": null,
    "timeout": 30000,
    "healthCheck": {
      "enabled": true,
      "interval": 60000
    }
  }
}
```

**Transport Types:**

| Transport | Description | Use Case |
|-----------|-------------|----------|
| `stdio` | Standard input/output | Local processes, WASM |
| `http` | HTTP JSON-RPC | Remote servers |
| `sse` | Server-Sent Events | Streaming responses |
| `streamable-http` | HTTP with streaming | Modern MCP servers |

---

### 4.9 Metadata Object (Optional)

Additional metadata for attribution and maintenance.

```json
{
  "metadata": {
    "author": "Anthropic",
    "license": "MIT",
    "maintainers": [
      "alice@anthropic.com",
      "bob@anthropic.com"
    ],
    "created": "2025-06-15T00:00:00Z",
    "updated": "2026-01-10T00:00:00Z",
    "deprecated": false,
    "deprecation_message": null,
    "minimum_scout_version": "1.0.0"
  }
}
```

---

## 5. Complete Examples

### 5.1 Simple API Key Tool (Brave Search)

```json
{
  "$schema": "https://mcp-scooter.com/schemas/registry/v1.json",
  
  "name": "brave-search",
  "version": "1.2.0",
  "title": "Brave Search",
  "description": "Search the web privately using Brave Search API",
  
  "category": "search",
  "source": "community",
  "tags": ["web-search", "privacy", "brave"],
  
  "icon": "/mcp-logos/brave-search.svg",
  "color": "#FB542B",
  
  "about": "## Brave Search MCP\n\nSearch the web privately using the Brave Search API. Unlike other search engines, Brave doesn't track your searches or build a profile on you.\n\n### Features\n\n- **Web Search** - Full web search with snippets and metadata\n- **News Search** - Filter for recent news articles\n- **Freshness Filters** - Limit results by time period\n\n### Getting Started\n\n1. Get an API key at [brave.com/search/api](https://brave.com/search/api)\n2. Add this tool: `scout_add(\"brave-search\")`\n3. Enter your API key when prompted\n\n### Pricing\n\nBrave Search API offers 2,000 free queries/month. See [pricing](https://brave.com/search/api/#pricing) for details.",
  
  "homepage": "https://brave.com/search",
  "repository": "https://github.com/anthropics/brave-search-mcp",
  "documentation": "https://api.search.brave.com/app/documentation",
  
  "authorization": {
    "type": "api_key",
    "required": true,
    "env_var": "BRAVE_API_KEY",
    "display_name": "Brave Search API Key",
    "description": "Your Brave Search API key",
    "help_url": "https://brave.com/search/api/register",
    "validation": {
      "pattern": "^BSA[a-zA-Z0-9]{29,}$"
    }
  },
  
  "tools": [
    {
      "name": "brave_web_search",
      "title": "Web Search",
      "description": "Search the web using Brave Search. Returns titles, URLs, and snippets.",
      "inputSchema": {
        "type": "object",
        "properties": {
          "query": {
            "type": "string",
            "description": "The search query"
          },
          "count": {
            "type": "integer",
            "description": "Number of results to return",
            "default": 10,
            "minimum": 1,
            "maximum": 20
          },
          "freshness": {
            "type": "string",
            "description": "Filter by recency",
            "enum": ["day", "week", "month", "year"]
          }
        },
        "required": ["query"]
      },
      "annotations": {
        "readOnlyHint": true,
        "destructiveHint": false,
        "idempotentHint": true,
        "openWorldHint": true
      }
    },
    {
      "name": "brave_news_search",
      "title": "News Search",
      "description": "Search for recent news articles using Brave Search",
      "inputSchema": {
        "type": "object",
        "properties": {
          "query": {
            "type": "string",
            "description": "News search query"
          },
          "count": {
            "type": "integer",
            "default": 10
          }
        },
        "required": ["query"]
      },
      "annotations": {
        "readOnlyHint": true,
        "openWorldHint": true
      }
    }
  ],
  
  "package": {
    "type": "npm",
    "name": "@anthropic/brave-search-mcp",
    "version": "^1.2.0"
  },
  
  "runtime": {
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@anthropic/brave-search-mcp"]
  },
  
  "metadata": {
    "author": "Anthropic",
    "license": "MIT",
    "created": "2025-03-01T00:00:00Z",
    "updated": "2026-01-10T00:00:00Z"
  }
}
```

---

### 5.2 OAuth Tool (Google Drive)

```json
{
  "$schema": "https://mcp-scooter.com/schemas/registry/v1.json",
  
  "name": "google-drive",
  "version": "2.0.0",
  "title": "Google Drive",
  "description": "Access and manage files in Google Drive",
  
  "category": "productivity",
  "source": "official",
  "tags": ["google", "drive", "files", "cloud-storage"],
  
  "icon": "/mcp-logos/google-drive.svg",
  "color": "#4285F4",
  
  "about": "## Google Drive MCP\n\nConnect your Google Drive to search, read, and manage files directly from your AI assistant.\n\n### Features\n\n- **Search Files** - Find documents by name, content, or metadata\n- **Read Files** - Access document contents (Docs, Sheets, PDFs)\n- **List Folders** - Browse your Drive hierarchy\n- **File Metadata** - Get sharing info, last modified, etc.\n\n### Permissions\n\nThis tool requests read-only access to your Drive. It cannot modify or delete files.\n\n### Setup\n\nWhen you add this tool, Scout will open a browser window for Google sign-in. Authorize the requested permissions to connect your Drive.",
  
  "homepage": "https://drive.google.com",
  "repository": "https://github.com/anthropics/google-drive-mcp",
  
  "authorization": {
    "type": "oauth2",
    "required": true,
    "provider": "google",
    "display_name": "Google Account",
    "description": "Connect your Google account to access Drive",
    "oauth": {
      "authorization_url": "https://accounts.google.com/o/oauth2/v2/auth",
      "token_url": "https://oauth2.googleapis.com/token",
      "scopes": [
        "https://www.googleapis.com/auth/drive.readonly"
      ],
      "scope_descriptions": {
        "https://www.googleapis.com/auth/drive.readonly": "View and download files in your Google Drive"
      },
      "pkce_required": true,
      "client_id_env": "GOOGLE_CLIENT_ID",
      "client_secret_env": "GOOGLE_CLIENT_SECRET",
      "token_env": "GOOGLE_ACCESS_TOKEN",
      "refresh_token_env": "GOOGLE_REFRESH_TOKEN"
    }
  },
  
  "tools": [
    {
      "name": "drive_search",
      "title": "Search Files",
      "description": "Search for files in Google Drive by name or content",
      "inputSchema": {
        "type": "object",
        "properties": {
          "query": {
            "type": "string",
            "description": "Search query"
          },
          "file_type": {
            "type": "string",
            "description": "Filter by file type",
            "enum": ["document", "spreadsheet", "presentation", "pdf", "image", "any"]
          },
          "max_results": {
            "type": "integer",
            "default": 10
          }
        },
        "required": ["query"]
      },
      "annotations": {
        "readOnlyHint": true,
        "openWorldHint": true
      }
    },
    {
      "name": "drive_read_file",
      "title": "Read File",
      "description": "Read the contents of a file from Google Drive",
      "inputSchema": {
        "type": "object",
        "properties": {
          "file_id": {
            "type": "string",
            "description": "Google Drive file ID"
          }
        },
        "required": ["file_id"]
      },
      "annotations": {
        "readOnlyHint": true
      }
    },
    {
      "name": "drive_list_folder",
      "title": "List Folder",
      "description": "List files in a Google Drive folder",
      "inputSchema": {
        "type": "object",
        "properties": {
          "folder_id": {
            "type": "string",
            "description": "Folder ID (use 'root' for root folder)",
            "default": "root"
          }
        }
      },
      "annotations": {
        "readOnlyHint": true
      }
    }
  ],
  
  "package": {
    "type": "npm",
    "name": "@anthropic/google-drive-mcp",
    "version": "^2.0.0"
  },
  
  "runtime": {
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@anthropic/google-drive-mcp"]
  },
  
  "metadata": {
    "author": "Anthropic",
    "license": "MIT",
    "created": "2025-01-15T00:00:00Z",
    "updated": "2026-01-05T00:00:00Z"
  }
}
```

---

### 5.3 Database Tool (PostgreSQL)

```json
{
  "$schema": "https://mcp-scooter.com/schemas/registry/v1.json",
  
  "name": "postgres-mcp",
  "version": "1.5.0",
  "title": "PostgreSQL Explorer",
  "description": "Connect to and query PostgreSQL databases",
  
  "category": "database",
  "source": "community",
  "tags": ["postgresql", "database", "sql", "data"],
  
  "icon": "/mcp-logos/postgresql.svg",
  "color": "#336791",
  
  "about": "## PostgreSQL MCP\n\nConnect to PostgreSQL databases to explore schemas, run queries, and analyze data.\n\n### Features\n\n- **Schema Exploration** - List tables, columns, and relationships\n- **Query Execution** - Run SELECT queries safely\n- **Data Analysis** - Get row counts, sample data, statistics\n\n### Security\n\n⚠️ **Read-Only Mode**: By default, only SELECT queries are allowed.\n\n⚠️ **Connection Strings**: Your database credentials are stored securely in your system keychain.\n\n### Supported Versions\n\nPostgreSQL 12, 13, 14, 15, 16",
  
  "homepage": "https://postgresql.org",
  "repository": "https://github.com/community/postgres-mcp",
  
  "authorization": {
    "type": "custom",
    "required": true,
    "display_name": "Database Connection",
    "description": "PostgreSQL connection details",
    "env_vars": [
      {
        "name": "DATABASE_URL",
        "display_name": "Connection String",
        "description": "Full connection URL: postgres://user:pass@host:port/database",
        "secret": true,
        "required": true
      },
      {
        "name": "PG_SSL_MODE",
        "display_name": "SSL Mode",
        "description": "SSL connection mode",
        "secret": false,
        "required": false,
        "default": "require",
        "options": ["disable", "allow", "prefer", "require", "verify-ca", "verify-full"]
      }
    ]
  },
  
  "tools": [
    {
      "name": "pg_list_tables",
      "title": "List Tables",
      "description": "List all tables in the database with row counts",
      "inputSchema": {
        "type": "object",
        "properties": {
          "schema": {
            "type": "string",
            "description": "Schema name",
            "default": "public"
          }
        }
      },
      "annotations": {
        "readOnlyHint": true
      }
    },
    {
      "name": "pg_describe_table",
      "title": "Describe Table",
      "description": "Get column definitions and constraints for a table",
      "inputSchema": {
        "type": "object",
        "properties": {
          "table": {
            "type": "string",
            "description": "Table name"
          },
          "schema": {
            "type": "string",
            "default": "public"
          }
        },
        "required": ["table"]
      },
      "annotations": {
        "readOnlyHint": true
      }
    },
    {
      "name": "pg_query",
      "title": "Run Query",
      "description": "Execute a SELECT query (read-only)",
      "inputSchema": {
        "type": "object",
        "properties": {
          "query": {
            "type": "string",
            "description": "SQL SELECT query"
          },
          "limit": {
            "type": "integer",
            "description": "Max rows to return",
            "default": 100,
            "maximum": 1000
          }
        },
        "required": ["query"]
      },
      "annotations": {
        "readOnlyHint": true,
        "requiresApproval": true
      }
    }
  ],
  
  "package": {
    "type": "npm",
    "name": "@community/postgres-mcp",
    "version": "^1.5.0"
  },
  
  "runtime": {
    "transport": "stdio",
    "command": "npx",
    "args": ["-y", "@community/postgres-mcp"]
  },
  
  "metadata": {
    "author": "MCP Community",
    "license": "Apache-2.0",
    "maintainers": ["postgres-mcp@mcp-community.org"],
    "created": "2025-02-20T00:00:00Z",
    "updated": "2026-01-12T00:00:00Z"
  }
}
```

---

### 5.4 Local WASM Tool

```json
{
  "$schema": "https://mcp-scooter.com/schemas/registry/v1.json",
  
  "name": "test-tool",
  "version": "0.1.0",
  "title": "Test Tool",
  "description": "Minimal Go-WASM tool for verification",
  
  "category": "utility",
  "source": "local",
  "tags": ["testing", "development"],
  
  "about": "A minimal test tool to verify WASM execution in Scout. Used for development and testing purposes.",
  
  "authorization": {
    "type": "none"
  },
  
  "tools": [
    {
      "name": "echo",
      "title": "Echo",
      "description": "Echoes back the input message",
      "inputSchema": {
        "type": "object",
        "properties": {
          "message": {
            "type": "string",
            "description": "Message to echo"
          }
        },
        "required": ["message"]
      },
      "annotations": {
        "readOnlyHint": true,
        "idempotentHint": true
      }
    }
  ],
  
  "package": {
    "type": "wasm",
    "local_path": "/wasm/test-tool.wasm"
  },
  
  "runtime": {
    "transport": "stdio"
  }
}
```

---

## 6. Validation & Best Practices

### 6.1 Required Fields Checklist

Every registry JSON must have:

- [ ] `name` — Unique, lowercase with hyphens
- [ ] `version` — Semantic version string
- [ ] `title` — Human-friendly display name
- [ ] `description` — One-line summary
- [ ] `category` — From taxonomy
- [ ] `source` — `official`, `community`, `enterprise`, or `local`
- [ ] `authorization` — At minimum `{ "type": "none" }`
- [ ] `tools` — At least one tool definition
- [ ] `package` — Installation source

### 6.2 Writing Good Descriptions

**Tool `description` field is critical for LLM discovery.**

❌ Bad: "Searches stuff"
✅ Good: "Search the web using Brave Search. Returns page titles, URLs, and text snippets."

❌ Bad: "DB tool"  
✅ Good: "Execute read-only SQL queries against a PostgreSQL database"

### 6.3 Authorization Security

1. **Never hardcode secrets** in registry JSON
2. Use `env_var` to reference environment variables
3. Mark sensitive fields with `secret: true`
4. Provide `help_url` so users know where to get credentials
5. Use validation patterns when possible

### 6.4 Tool Annotations

Always include annotations for:
- `readOnlyHint` — Does this only read data?
- `destructiveHint` — Can this delete or modify data?
- `requiresApproval` — Should Scout ask for Human-in-the-Loop approval?

---

## 7. Migration from Current Format

Current registry files use a minimal format:

```json
{
  "name": "brave-search",
  "description": "Search the web with Brave Search API",
  "category": "search",
  "source": "community",
  "icon": "/mcp-logos/brave-search.svg"
}
```

To migrate to the full specification:

1. Add `version` and `title`
2. Add `about` markdown documentation
3. Define `authorization` object
4. List all `tools` with schemas
5. Add `package` and `runtime` configuration
6. Add optional `metadata`

---

## 8. JSON Schema for Validation

A formal JSON Schema file should be created at `schemas/registry-v1.schema.json` to enable automated validation of registry entries.

---

## 9. References

- [Model Context Protocol Specification](https://modelcontextprotocol.io/specification)
- [MCP Registry Publishing Guide](https://modelcontextprotocol.info/tools/registry/publishing)
- [JSON Schema](https://json-schema.org/)
- [OAuth 2.0 RFC 6749](https://tools.ietf.org/html/rfc6749)
- [PKCE RFC 7636](https://tools.ietf.org/html/rfc7636)
- MCP Scout PRD v2.9
