# **Product Requirement Document (PRD): MCP Scooter**

Version: 3.0  
Status: Draft  
Date: January 24, 2026  
Author: Product Team

## **1\. Executive Summary**

**MCP Scooter** is a lightweight, source-available desktop application that acts as the universal "Operating System" for the Model Context Protocol (MCP). It solves the critical scalability issues of AI agents—context bloat, configuration fatigue, and security risks—by replacing heavy Docker containers with a native, high-performance **Dynamic Gateway**.

**The Core Promise:** *"MCP Scooter runs in your system tray, manages your professional and personal AI identities, and instantly spawns MCPs for any AI client (Cursor, Claude, Antigravity) with zero latency and \<50MB RAM usage."*

**Distribution:** The project includes a public-facing website (hosted on GitHub Pages) serving documentation, changelogs, and binary downloads directly from GitHub Releases.

## **2\. Problem Statement**

As MCP adoption explodes, developers face four compounding crises:

1. **Context Bloat (The "Hard-Coding" Trap):**  
   * Developers currently hard-code MCP definitions into their agent's config.  
   * *Result:* Connecting 50 MCPs floods the LLM's context window with 50 unused definitions, degrading performance and increasing costs.
   * *Industry Trend:* Developers are migrating to "Skills + CLI" patterns to reduce token consumption by 70%+. MCP Scooter must provide a native solution.

2. **Configuration Fragmentation:**  
   * A developer uses **Cursor** for work, **Claude Desktop** for research, and **Google Antigravity** for side projects.  
   * *Result:* They must manually copy-paste JSON configurations and API keys across 8 different config files (config.toml, settings.json, .claude/config, etc.), leading to "Configuration Drift" and security leaks.  

3. **The "Docker Weight" Problem:**  
   * The current solution (Docker MCP Gateway) requires running a heavy Linux VM on Mac/Windows.  
   * *Result:* High RAM usage (2-4GB) and slow startup times (3-5s) make it unusable for local-first development.

4. **CLI-First Developer Experience Gap:**
   * Competing tools like MCPorter offer `npx mcporter call server.tool args` for instant tool invocation.
   * *Result:* Developers who prefer terminal workflows cannot easily test or use MCP tools without a GUI.

## **3\. Product Vision & User Journey**

### **3.1 The Vision**

To become the **standard local runtime** for MCP. If MCP is the "USB port" for AI, **MCP Scooter is the Universal Hub**.

### **3.2 User Journey: "The Dual-Identity Developer"**

* **Meet Alex:** A Senior DevOps Engineer who uses **Cursor** for work and **Claude Desktop** for personal coding.  
* **09:00 AM (Work Mode):**  
  * Alex opens MCP Scooter. It sits in the tray.  
  * She selects the **"Work @ Corp"** profile.  
  * She opens Cursor. Cursor is configured to talk to 127.0.0.1:6277.  
  * She asks Cursor: *"Check the Prod DB health."*  
  * **Scooter Action:** Scooter authenticates via the Work Profile, dynamically spawns the Postgres-Prod MCP (in a secure sandbox), and pipes the result to Cursor. The LLM *never saw* the definitions for her personal Spotify MCP.  
* **06:00 PM (Personal Mode):**  
  * Alex doesn't close Cursor. She just opens **Claude Desktop** (configured to localhost:6278).  
  * She asks Claude: *"Analyze my Spotify listening history."*  
  * **Scooter Action:** Scooter detects the request on the Personal Port (:6278). It spawns the Spotify-MCP using her personal API key.  
  * *Crucial:* Her work credentials never leaked to the personal session, and her personal MCPs never cluttered her work context.

### **3.3 User Journey: "The CLI Power User"**

* **Meet Sam:** A backend engineer who lives in the terminal and uses Claude Code.
* **Workflow:**
  * Sam runs `scooter call context7.resolve-library-id libraryName=react` directly from terminal.
  * No GUI needed. The CLI talks to the Scooter daemon running in the background.
  * Sam creates a skill file: `scooter skill export context7 > .claude/skills/context7/SKILL.md`
  * Now Claude Code can use the skill with minimal token overhead.
* **Result:** Sam gets the power of MCPs with CLI convenience and token efficiency.

## **4\. Detailed Feature Specifications**

### **4.1 Native Cross-Platform Application**

* **Requirement:** Must be a native binary, not an Electron wrapper or Docker container.  
* **OS Support:** Windows 11 (ARM/x64), macOS (Apple Silicon/Intel), Linux (Debian/RPM).  
* **Performance Budget:** \<50MB RAM idle, \<10ms MCP startup time.  
* **Native UI (Windows Priority):**
  * The application must feel **100% Native to Windows**.
  * **Visuals:** Use **WinUI 3** design language (Segoe UI Variable font, standard spacing).
  * **Window Effects:** Implement **Mica** or **Acrylic** background materials. The window should feel like part of the OS, not a web page.
  * **Controls:** Use native-style inputs, toggles, and title bars. 
  * **No Web-like styling:** Avoid shadows, large paddings, or non-native scrollbars typical of generic web apps.

* **System Integration (Background Service):**  
  * **Windows:** Runs as a background process with a **Notification Area (System Tray)** icon. Closing the dashboard window minimizes to tray; right-click the icon to Quit completely.  
  * **macOS:** Lives in the **Menu Bar**. Can operate in "Headless Mode" (no Dock icon) or Standard Mode. The Dashboard toggles via the Menu Bar icon.  
  * **Linux:** Uses libappindicator to reside in the System Tray (Top Right on GNOME/Ubuntu, Bottom Right on KDE).  
  * **Persistence:** The server daemon remains active to handle MCP requests even when the GUI dashboard is closed.  
* **Networking Strategy:**
  * **Default Ports:** Scooter defaults to uncommon high ports to avoid conflicts with standard web development tools (like 3000, 8000, 8080).
    * **Primary Profile:** 6277 (M-C-P-S on keypad).
    * **Secondary Profile:** 6278\.
  * **Conflict Detection:** If 6277 is in use by another app, Scooter must detect this at startup and prompt the user to choose an alternative or auto-select 6279\.

### **4.2 "Scooter Profiles" & Security**

The killer feature. Users create isolated environments (Profiles) and secure their gateway with API keys.

*   **Gateway Security:** 
    *   The MCP Gateway (:6277) is secured via an API Key (`sk-scooter-...`).
    *   AI Clients must provide this key in the `Authorization: Bearer` or `X-Scooter-API-Key` header.
    *   Scooter automatically configures this key during the one-click integration flow for supported clients.

* **Profile Configuration (profiles.yaml):**  
  ```yaml
  settings:
    gateway_api_key: "sk-scooter-..." # Secures IDE connections
  profiles:  
    - id: work-profile  
      remote_auth_mode: "oauth2" # For connecting to a protected Remote MCP Server (renamed from auth_mode)
      remote_server_url: "https://mcp.acme-corp.com"  
      env:  
        AWS_REGION: "us-east-1"  
      allow_tools:  
        - "jira-mcp"
  ```

* **Authentication Engine (OAuth 2.0 & 2.1):**
  * **Remote Server Auth (OAuth 2.1):**
    * Scooter implements **RFC 8414 (Authorization Server Metadata)**. When connecting to a protected remote MCP server (e.g., Enterprise Data Gateway), Scooter automatically handles the 401 Unauthorized challenge.
    * **Flow:** Scooter detects the challenge \-\> Initiates PKCE Flow \-\> Opens System Browser for SSO Login \-\> Captures Callback \-\> Stores Token in Keychain \-\> Retries Request.
    * **Benefit:** The AI Client (e.g., Cursor) does *not* need to implement OAuth. It just talks to Scooter, and Scooter handles the auth.
  * **Local Tool Auth (3rd Party Tokens):**
    * For local MCPs that need user context (e.g., google-drive-mcp), Scooter acts as a **Token Manager**.
    * Scooter maintains a refresh loop for Google/Slack/GitHub tokens and injects them into the local MCP process as environment variables (e.g., GOOGLE\_ACCESS\_TOKEN) at runtime.
* **Secure Credential Storage:** Scooter integrates with **macOS Keychain**, **Windows Credential Manager**, and **Linux Secret Service**. Tokens are never stored in plain text.

### **4.3 The "Scooter Gateway" (Dynamic Discovery Engine)**

This mimics the "Docker MCP Toolkit" pattern but runs natively. Instead of hard-coding tools, **Scooter exposes a Discovery Protocol** to the agent. The key principle is that **external tools are never exposed until explicitly activated**.

* **The "Zero-Config" Experience:**
  * When a user installs Scooter, they don't need to manually "install" 50 tools.
  * Scooter simply connects to the AI Client and exposes **4 Primordial Tools** (the "meta-layer"):
    * `scooter_find` - Search for available MCPs in the registry
    * `scooter_activate` - Turn on an MCP server for the current session
    * `scooter_deactivate` - Turn off an MCP server (or all servers with `all: true`)
    * `scooter_list_active` - List currently active MCP servers and their tools
  * **Future:** `scooter_ai` (AI-powered intent routing) is planned for a future release.
  * **Critical:** External MCPs (like `brave_web_search`) are **NOT** visible to the AI client until the agent explicitly calls `scooter_activate`. This prevents context bloat and matches Docker MCP Toolkit behavior.

* **Why 4 Primordial Tools?**
  * These 4 tools provide complete lifecycle management for MCP servers.
  * The base token footprint remains minimal (~100 tokens total).
  * `scooter_activate` response includes tool schemas inline for immediate use.
  * Code interpreter functionality is available but not exposed as a primordial tool to keep context lean.

* **Tool Activation Flow (Explicit Loading Pattern):**  
  1. **Initial State:** AI client connects and calls `tools/list`. It only sees the 4 primordial tools.
  2. **Discovery:** User asks *"Search the web for AI news"*. Agent calls `scooter_find(query="search")`.
  3. **Results:** Scooter returns available servers: *"Found 'brave-search'. Tools: brave\_web\_search, brave\_local\_search"*.
  4. **Activation:** Agent calls `scooter_activate("brave-search")`.
  5. **Server Startup:** Scooter checks if `brave-search` is in the profile's `AllowTools`, then starts the MCP server process.
  6. **Notification:** Scooter sends `notifications/tools/list_changed` via SSE to notify the client.
  7. **Tool Available:** Client refreshes `tools/list` and now sees `brave_web_search` and `brave_local_search`.
  8. **Usage:** Agent calls the tool directly (e.g., `brave_web_search({query: "AI news"})`). MCP Scooter routes the call to the correct backend.

* **No Auto-Loading:** Unlike some implementations, Scooter does **NOT** auto-load MCPs when called. If an agent tries to call `brave_web_search` without first calling `scooter_activate("brave-search")`, Scooter returns a helpful error: *"MCP 'brave\_web\_search' is not active. Use scooter\_activate('brave-search') to turn it on first."*

* **Permission Model:**
  * MCPs must be in the profile's `AllowTools` list before they can be activated via `scooter_activate`.
  * Attempting to add an unauthorized MCP returns: *"MCP 'github' is not allowed for this profile. Add it to AllowTools in your profile configuration."*

* **Resource Hygiene:** Scooter monitors usage. If an MCP server hasn't been used in 10 minutes, Scooter automatically unloads it to save RAM and Context Window space. When this happens, SSE clients are notified via `tools/list_changed`.

* **One-Click Setup (The "Integrations" Tab)**

Scooter automates the configuration of 3rd party clients. The "Integrations" tab allows users to click "Install" for:

| Client | Configuration Strategy | Target File |
| :---- | :---- | :---- |
| **Cursor** | Injects mcp.json | \~/.cursor/mcp.json |
| **Claude Desktop** | Edits config file | \~/Library/.../claude\_desktop\_config.json |
| **VS Code** | Uses MCP Extension API | .vscode/mcp.json |
| **Claude Code** | CLI Configuration | \~/.claude/settings.json |
| **Google Antigravity** | Updates Agent Config | .gemini/settings.json or mcp\_config.json |
| **Gemini CLI** | Edits CLI settings | .gemini/settings.json |
| **Codex** | Edits TOML config | \~/.codex/config.toml |
| **Zed** | Edits Settings | \~/.config/zed/settings.json |

* **Implementation:** Scooter acts as a local proxy. It writes a configuration that points the client to `http://127.0.0.1:6277/sse` (Server-Sent Events) and includes the **Gateway API Key** in the request headers, effectively routing all traffic through Scooter securely.

### **4.5 Custom MCP & Export/Import**

* **Custom MCP Wizard:**  
  * UI to add local MCPs (e.g., "Run Python Script").  
  * Inputs: Command (python/node), Args, Env Vars.  
  * **Auth Wrapper:** Checkbox for *"Manage OAuth for this tool"*. If checked, Scooter handles the Google/Slack login flow and passes the token to the script.
  * **Validation:** Scooter dry-runs the MCP to verify it speaks MCP protocol.  
* **Export/Import:**  
  * **Format:** scout-bundle.json.  
  * **Scope:** Exports Profiles, Tool Lists, and non-sensitive Configs.  
  * **Security:** Secrets (API Keys) are **stripped** on export. On import, the user is prompted to re-enter missing keys ("Enter JIRA\_API\_KEY for 'Work Profile'").

### **4.6 Developer Experience (DX) & Tool Playground**

* **Log Inspector:** A "Network Tab" for AI. View raw JSON-RPC requests and responses.  
* **Tool Playground:** A "Try it now" interface. Users can manually invoke jira-mcp with sample JSON input to verify connectivity *before* using it in an agent. This solves the "Silent Failure" problem.  
* **Human-in-the-Loop Security:**  
  * **Gateway API Key Management:** UI for viewing, copying, and regenerating the gateway security key. Regeneration automatically prompts to re-sync all integration clients.
  * **Permission Modes:** Per-tool settings: "Always Allow," "Allow Read-Only," or "Ask for Approval."  
  * **Approval UI:** If an agent tries to use a "Sensitive" tool (e.g., delete_database), Scooter pops a native OS notification asking the user to Approve/Deny.

### **4.7 "Code Mode" Lite (Sandboxed Composition)**

Just like Docker's "Code Mode," Scooter allows agents to write scripts to chain tools together, but faster.

* **The Efficiency Problem:** Standard agents return massive JSON blobs (e.g., 50 GitHub repos) to the LLM, wasting tokens.  
* **The Solution:**  
  1. The Agent writes a JavaScript function to: search\_github("query") \-\> filter(repo \=\> repo.stars \> 1000\) \-\> format\_output().  
  2. Scooter executes this script in a secure **V8 Isolate** (goja).
  3. Scooter returns *only* the filtered, formatted result to the LLM.  
* **State Persistence:** Includes a "Volume" API so agents can download a dataset to a temp folder, process it, and delete it, without the data ever touching the Context Window.
* **Note:** The code interpreter exists in the codebase but is not exposed as a primordial tool to minimize base context footprint. It can be enabled per-profile as an opt-in feature.

### **4.8 Scooter CLI (Developer Experience)**

The Scooter CLI is a **companion tool** to the Desktop Application, providing terminal access for power users who prefer command-line workflows.

> **Important:** The CLI requires the MCP Scooter Desktop Application to be installed and running. The CLI does not function standalone—it connects to the daemon managed by the Desktop App. Profile creation, tool registration, credential management, and client integrations must be configured via the Desktop UI first.

* **Prerequisites:**
  * MCP Scooter Desktop Application installed and running (system tray)
  * At least one profile configured via the Desktop UI
  * Required tools added to profile's `AllowTools` list

* **Installation (Future - Phase 2):**
```bash
# Via npm (cross-platform)
npm install -g @mcp-scooter/cli

# Via Homebrew (macOS/Linux)
brew install mcp-scooter/tap/scooter

# Via winget (Windows)
winget install mcp-scooter

# Via npx (no install)
npx @mcp-scooter/cli list
```

* **Core Commands:**
```bash
scooter list                     # List available MCPs in registry
scooter find "search"            # Search for MCPs by capability
scooter activate brave-search    # Activate an MCP server
scooter call brave-search.search query="AI news"  # Call a tool directly
scooter status                   # Show active servers and connections
scooter deactivate brave-search  # Unload an MCP server
```

* **Profile Support:**
  ```bash
  scooter --profile work call github.list_repos
  scooter --profile personal call spotify.get_playlists
  scooter profile list
  scooter profile switch work
  ```

* **Skill Commands (Phase 2):**
  ```bash
  scooter skill list                           # List available skills
  scooter skill export brave-search            # Generate skill file for a tool
  scooter skill install full-stack-dev         # Install a skill bundle from catalog
  ```

* **Implementation:** 
  * The CLI communicates with the running Scooter daemon via the Control API (port 6200).
  * The daemon is managed by the Desktop Application (runs in system tray).
  * Output formats: `--json` for machine-readable, `--raw` for unformatted, default for human-friendly.

* **What the CLI Cannot Do (Use Desktop UI Instead):**
  * Create or delete profiles
  * Register custom MCP tools
  * Set API keys or credentials
  * Configure client integrations (Cursor, Claude, etc.)
  * Manage gateway settings

### **4.9 Scooter Skills System**

Skills provide a token-efficient way to extend agent capabilities without loading full MCP tool schemas into context. This addresses the "Skills + CLI" pattern that is becoming industry standard.

#### **4.9.1 What is a Skill?**

A Skill is a lightweight instruction set that tells an AI agent how to use tools without loading the full tool schemas into context.

* **Format:** Markdown file (`SKILL.md`) containing:
  * Description of when to use the skill
  * Available commands/tools and their parameters
  * Usage examples and best practices
  * Optional: System prompt additions

* **Token Efficiency:**
  * Full MCP tool schema: ~500-2000 tokens per tool
  * Skill file: ~50-200 tokens per skill
  * **Result:** 70%+ reduction in context consumption

#### **4.9.2 Skill Structure**

```markdown
# Brave Search Skill

Use this skill when the user wants to search the web for current information,
news, documentation, or any real-time data.

## Available Commands

### Web Search
```bash
scooter call brave-search.search query="<search query>"
```

**Parameters:**
- `query` (required): The search query string
- `count` (optional): Number of results (default: 10)

### Local Search
```bash
scooter call brave-search.local_search query="<query>" location="<city>"
```

## Usage Notes
- Use for current events, documentation, news
- Results include title, URL, and snippet
- Prefer specific queries over broad ones

## Examples
- "Search for latest React 19 features" → `scooter call brave-search.search query="React 19 new features 2026"`
- "Find coffee shops in Seattle" → `scooter call brave-search.local_search query="coffee shops" location="Seattle"`
```

#### **4.9.3 Skill Types**

| Type | Description | Example |
|------|-------------|---------|
| **Tool Skill** | Maps to a single MCP server | `brave-search`, `github`, `linear` |
| **Composite Skill** | Bundles multiple tools for a workflow | `full-stack-dev`, `data-analyst` |
| **Custom Skill** | User-created for project-specific needs | `my-company-api`, `deploy-workflow` |

#### **4.9.4 Skills Catalog**

MCP Scooter hosts a **Skills Catalog** — a curated registry of skills that users can browse and install.

* **Catalog Structure:**
  ```
  appdata/
  ├── registry/           # MCP server definitions
  │   ├── official/
  │   └── custom/
  └── skills/             # Skill definitions
      ├── official/       # Curated skills from Scooter team
      │   ├── brave-search.skill.json
      │   ├── github.skill.json
      │   └── full-stack-dev.skill.json
      ├── community/      # Community-contributed skills
      └── custom/         # User-created skills
  ```

* **Skill Definition Schema (`*.skill.json`):**
  ```json
  {
    "$schema": "../schemas/skill.schema.json",
    "name": "brave-search",
    "title": "Brave Search",
    "description": "Search the web using Brave Search API",
    "version": "1.0.0",
    "category": "search",
    "requires_tools": ["brave-search"],
    "skill_content": "# Brave Search Skill\n\nUse this skill...",
    "tags": ["search", "web", "research"],
    "author": "MCP Scooter Team",
    "homepage": "https://mcp-scooter.com/skills/brave-search"
  }
  ```

* **Composite Skill Example (`full-stack-dev.skill.json`):**
  ```json
  {
    "name": "full-stack-dev",
    "title": "Full Stack Developer",
    "description": "Complete toolkit for full-stack web development",
    "requires_tools": ["github", "postgres", "browser-mcp", "context7"],
    "skill_content": "# Full Stack Developer Skill\n\n...",
    "system_prompt_addition": "You are a full-stack developer with access to GitHub, PostgreSQL, browser automation, and documentation lookup."
  }
  ```

#### **4.9.5 Skill Installation & Distribution**

* **One-Click Install (Desktop UI):**
  * Browse Skills Catalog in the Scooter dashboard
  * Click "Install" to download skill + required MCP tools
  * Skill is automatically placed in the correct location for each integrated client

* **CLI Install:**
  ```bash
  # Install from catalog
  scooter skill install brave-search
  
  # Install composite skill (installs all required tools)
  scooter skill install full-stack-dev
  
  # Install from URL
  scooter skill install --url https://example.com/my-skill.skill.json
  ```

* **Auto-Placement:** When installing a skill, Scooter places the generated `SKILL.md` in the correct location for each integrated client:
  * Cursor: `.cursor/skills/{skill-name}/SKILL.md`
  * Claude Code: `.claude/skills/{skill-name}/SKILL.md`
  * VS Code: `.vscode/skills/{skill-name}/SKILL.md`
  * Generic: `~/.scooter/skills/{skill-name}/SKILL.md`

#### **4.9.6 Skill Export**

Generate skill files from any registered MCP server:

```bash
# Export a single tool as a skill
scooter skill export brave-search > SKILL.md

# Export with full parameter documentation
scooter skill export brave-search --verbose

# Export to specific client location
scooter skill export brave-search --client cursor
# Creates: .cursor/skills/brave-search/SKILL.md

# Export all installed tools as skills
scooter skill export-all --client cursor
```

#### **4.9.7 Hybrid Mode: Skills + Gateway**

Skills can work in two modes:

| Mode | How It Works | Best For |
|------|--------------|----------|
| **CLI Mode** | Agent runs `scooter call ...` commands | Any agent with shell access |
| **Gateway Mode** | Agent uses `scooter_activate` then calls tools via MCP | MCP-native clients |

* **CLI Mode Skill:**
  ```markdown
  ## Commands
  ```bash
  scooter call brave-search.search query="..."
  ```
  ```

* **Gateway Mode Skill:**
  ```markdown
  ## Usage
  1. Call `scooter_activate("brave-search")` to enable the tool
  2. Call `brave_web_search({query: "..."})` directly
  ```

* **Hybrid Skill (Recommended):**
  ```markdown
  ## CLI Usage (Recommended for token efficiency)
  ```bash
  scooter call brave-search.search query="..."
  ```
  
  ## Gateway Usage (For MCP-native clients)
  1. Activate: `scooter_activate("brave-search")`
  2. Call: `brave_web_search({query: "..."})`
  ```

## **5\. Exposed Tools & Capabilities Registry**

Unlike standard MCP servers that expose a static list, Scooter exposes a dynamic hierarchy. The following **Primordial Tools** are intrinsic to the platform and always available. **External tools are only visible after explicit activation via `scooter_activate`.**

### **5.1 The "Meta-Layer" (Primordial Tools)**

Every AI client connected to Scooter sees these 4 tools by default. These are the **only** tools visible until the agent activates external tools.

* **scooter\_find(query: string)**  
  * *Description:* Searches the Local Registry and Community Catalog for MCPs.  
  * *Returns:* Server names, descriptions, and the list of tools each server provides.
  * *Example Response:* `{tools: [{name: "brave-search", tools: ["brave_web_search", "brave_local_search"]}]}`

* **scooter\_activate(tool\_name: string)**  
  * *Description:* Turns on an MCP server for the current session.  
  * *Action:* Validates against AllowTools, authenticates (if needed), starts the server process, and notifies clients via SSE.
  * *Returns:* `{status: "on", server: "mcp-scooter", activated_from: "brave-search", available_tools: ["brave_web_search", "brave_local_search"], tool_schemas: [...]}`

* **scooter\_deactivate(tool\_name: string, all: boolean)**  
  * *Description:* Turns off an MCP server for the current session.  
  * *Action:* Stops the server process and removes its tools from the available list. Use `all: true` to deactivate all servers.
  * *Returns:* `{status: "off", server: "brave-search", message: "Server 'brave-search' has been deactivated."}`

* **scooter\_list\_active()**  
  * *Description:* Lists all currently active MCP servers and their tools.  
  * *Returns:* `{active_servers: [{server: "brave-search", tools: ["brave_web_search"], count: 1}], count: 1}`

### **5.2 Future Tools (Planned)**

The following tools are planned for future releases:

* **scooter\_ai(intent: string)** *(Future)*
  * *Description:* AI-powered intent routing that automatically selects and calls the appropriate tool based on natural language intent.
  * *Status:* Reserved for future release. The backend implementation exists but is not yet exposed.

* **scooter\_code(script: string, args: object)** *(Opt-in)*
  * *Description:* Execute sandboxed JavaScript to chain tools and process data.
  * *Status:* Implementation exists (V8/goja). Can be enabled per-profile as opt-in to avoid base context bloat.

## **6\. Competitive Landscape**

| Feature | Docker MCP | MCPorter | MetaMCP | Manual Config | **MCP Scooter** |
| :---- | :---- | :---- | :---- | :---- | :---- |
| **Primary Use Case** | Enterprise Infra | CLI Power Users | Server-side Proxy | Hobbyist | **Pro Developer** |
| **Architecture** | Linux Containers | Node.js CLI | Docker Container | Local Process | **Native Binary + WASM** |
| **CLI Interface** | ❌ | ✅ Excellent | ❌ | ❌ | **✅ (Phase 1.5)** |
| **Skills Support** | ❌ | ✅ (via CLI) | ❌ | ❌ | **✅ (Phase 2)** |
| **Skills Catalog** | ❌ | ❌ | ❌ | ❌ | **✅ (Phase 2)** |
| **Dynamic Discovery** | ✅ (Container) | ❌ (Static) | ✅ | ❌ | **✅ Protocol-level** |
| **Code Mode** | ✅ (Docker) | ❌ | ❌ | ❌ | **✅ (V8 Isolate)** |
| **Profile Support** | ❌ (Env Vars) | ❌ | ❌ (Namespace) | ❌ (Manual) | **✅ First-class** |
| **Resource Usage** | Heavy (2GB+) | Medium (Node.js) | Heavy (Server) | Light | **Ultra-Light (<50MB)** |
| **One-Click Setup** | ❌ | ❌ | ❌ | ❌ | **✅ (8+ Clients)** |
| **Token Efficiency** | ❌ (All loaded) | ✅ (CLI-based) | ❌ | ❌ | **✅ (Dynamic + Skills)** |

### **6.1 Competitive Advantages**

| Advantage | Description |
|-----------|-------------|
| **Protocol-Level Dynamic Loading** | Unlike MCPorter's CLI wrapper approach, Scooter's `scooter_find` → `scooter_activate` works at the MCP protocol level, maintaining full type safety and schema validation. |
| **Skills Catalog** | MCPorter requires manual skill creation. Scooter provides a curated catalog with one-click install. |
| **Profile Isolation** | No competitor offers first-class credential isolation between work/personal contexts. |
| **Native Performance** | <50MB RAM vs 2GB+ for Docker-based solutions. |
| **Centralized Gateway** | Multiple clients connect to one Scooter instance, sharing credentials and active tools. |

## **7\. Technical Architecture (The "Go-WASM" Stack)**

### **7.1 Backend (Go)**

* **net/http**: Handles SSE (Server-Sent Events) connections from clients (Claude, Cursor).  
* **goroutines**: Manages concurrent profile servers (Port 6277, 6278).  
* **fsnotify**: Watches config files for real-time updates.  
* **oauth2\_client**: Implements RFC 8414 (Metadata Discovery) & RFC 7636 (PKCE) to authenticate against Remote MCP Servers.

### **7.2 Isolation Engine**

* **wazero**: Runs WASM-compiled MCP servers.  
* **goja**: Runs JavaScript-based "Code Mode" scripts (V8-compatible).  
* **os/exec**: Fallback for running local Node/Python scripts (Legacy Mode).

### **7.3 Frontend**

* **Tauri (Rust)**: Wraps a React/Tailwind web app into a native window.  
* **Communication:** Inter-Process Communication (IPC) between Tauri frontend and Go backend.

### **7.4 CLI**

* **Language:** Go (same codebase as backend)
* **Distribution:** Single binary + npm package (`@mcp-scooter/cli`)
* **Communication:** HTTP to Control API (port 6200)

## **8\. Roadmap & Phasing**

### **Phase 1 (MVP - Foundation & Gateway):** ✅ Mostly Complete

| Component | Status | Description |
|-----------|--------|-------------|
| Registry Schema | ✅ Done | JSON Schema for MCP server definitions |
| Registry Validation | ✅ Done | CLI tool to validate registry entries |
| Profile Management | ✅ Done | Create, update, delete profiles with persistence |
| Discovery Engine | ✅ Done | `scooter_find`, `scooter_activate`, `scooter_deactivate`, `scooter_list_active` |
| MCP Gateway | ✅ Done | SSE server handling JSON-RPC for all profiles |
| Client Integrations | ✅ Done | Cursor, Claude Desktop, Claude Code, VS Code, Gemini CLI, Zed, Codex |
| Keychain Integration | ✅ Done | Secure credential storage (Windows/macOS/Linux) |
| Tauri Desktop Shell | ✅ Done | Native window with React frontend |
| Desktop UI | 🚧 Building | Profile management UI, tool browser, settings |
| OAuth 2.0 Handler | 🚧 Building | Automatic auth flows for Google, GitHub, Slack |
| Tool Playground | 🚧 Building | Manual tool testing interface |

### **Phase 1.5 (CLI & Developer Experience):** 🆕 NEW

| Component | Status | Description |
|-----------|--------|-------------|
| **Scooter CLI** | 📋 Planned | Terminal interface for power users |
| CLI Core Commands | 📋 Planned | `scooter list`, `scooter find`, `scooter call`, `scooter status` |
| Ad-hoc Connections | 📋 Planned | `--url`, `--stdio` flags for instant server connections |
| Profile Switching | 📋 Planned | `--profile` flag, `scooter profile switch` |
| npx Support | 📋 Planned | `npx @mcp-scooter/cli call ...` |
| Skill Export | 📋 Planned | `scooter skill export` command |

### **Phase 2 (Skills & Ecosystem):**

| Component | Status | Description |
|-----------|--------|-------------|
| **Skills System** | 📋 Planned | Token-efficient tool descriptions |
| Skill Schema | 📋 Planned | JSON schema for skill definitions |
| Skills Catalog | 📋 Planned | Curated registry of official + community skills |
| Skill Export | 📋 Planned | Generate SKILL.md from any MCP server |
| Skill Install | 📋 Planned | One-click install with auto-placement |
| Composite Skills | 📋 Planned | Skill bundles (e.g., "Full Stack Dev") |
| **Scooter Store** | 📋 Planned | Community registry of tools AND skills |
| Code Interpreter Opt-in | 📋 Planned | Re-expose `scooter_code` as opt-in per-profile |
| Remote MCP Support | 📋 Planned | Enterprise gateway connections with OAuth 2.1 |
| TypeScript SDK | 📋 Planned | `@mcp-scooter/client` package |

### **Phase 3 (Enterprise):**

| Component | Status | Description |
|-----------|--------|-------------|
| Team Sync | 📋 Planned | Share profiles via encrypted cloud config |
| Audit Logs | 📋 Planned | Compliance-ready logging |
| SSO Integration | 📋 Planned | Enterprise identity providers |
| Admin Dashboard | 📋 Planned | Centralized team management |

## **9\. Web Presence & Distribution (Open Source)**

The project includes a front-facing website hosted on **GitHub Pages** to serve as the single source of truth for users and contributors.

### **9.1 Hosting Strategy**

* **Domain:** mcp-scooter.com (CNAME pointing to mcp-scooter.github.io).  
* **Deployment:** Automated via GitHub Actions on push to main.  
* **Technology:** Static HTML with Tailwind CSS (Single Page Application architecture for speed).

### **9.2 Website Structure**

* **Landing Page:**  
  * Value Proposition: "Native, Lightweight, Dynamic."  
  * Visuals: Terminal recording showing `scooter find` and `scooter call` commands.  
  * Call to Action: Smart "Download" button that detects User OS (Mac/Win/Linux).  
* **Documentation Hub:**  
  * **Installation:** brew, winget, npm, and curl scripts.  
  * **Configuration Guide:** Detailed syntax for profiles.yaml.  
  * **CLI Reference:** Full command documentation.
  * **Skills Guide:** How to create, export, and use skills.
  * **Tool Authoring:** Guide on how to compile existing Python tools to WASM for Scooter.  
* **Skills Catalog (Web):**
  * Browse available skills with search and filtering.
  * One-click copy of install commands.
  * Community contribution guidelines.
* **Releases Page (Dynamic):**  
  * **Integration:** Fetches data directly from **GitHub Releases API**.  
  * **Content:** Displays the latest Version (e.g., v1.0.0), Release Date, and Changelog notes (parsed from Markdown).  
  * **Assets:** Direct links to .dmg, .exe, and .deb binaries.

## **10\. Repository Standards**

To ensure a high-quality source-available ecosystem, the repository must adhere to the following:

* **README.md:**  
  * Must include **Shields.io** badges for Build Status, License (PolyForm Shield), and Platform Support.  
  * Quick Start guide with download links for releases.  
  * Build-from-source instructions for contributors.  
  * High-level Architecture Diagram (Text-based Mermaid.js).
  * **Note:** README must accurately reflect the current primordial tools (currently 2: `scooter_find`, `scooter_activate`).
* **LICENSE:**  
  * **PolyForm Shield 1.0.0** — Allows free use and modification, but prohibits competing products/services.  
  * Users can build products *with* MCP Scooter, but cannot fork it to create a competing MCP gateway.  
* **CONTRIBUTING.md:**  
  * Instructions for setting up the Go/Rust dev environment.  
  * Guidelines for submitting new tools to the "Scooter Store."
  * Guidelines for contributing skills to the Skills Catalog.
* **CHANGELOG.md:**  
  * Maintained automatically via Semantic Release, but reflected on the website.

## **11\. Success Metrics**

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Token Efficiency** | 70%+ reduction vs static MCP | Compare context usage with/without skills |
| **CLI Adoption** | 30% of users use CLI | Analytics on CLI vs GUI usage |
| **Skills Catalog Growth** | 50+ skills in 6 months | Count of official + community skills |
| **Developer Satisfaction** | NPS > 50 | User surveys |
| **Performance** | <50MB RAM, <10ms startup | Automated benchmarks |
