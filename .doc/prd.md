# **Product Requirement Document (PRD): MCP Scooter**

Version: 2.9  
Status: Draft  
Date: January 16, 2026  
Author: Product Team

## **1\. Executive Summary**

**MCP Scooter** is a lightweight, source-available desktop application that acts as the universal "Operating System" for the Model Context Protocol (MCP). It solves the critical scalability issues of AI agents—context bloat, configuration fatigue, and security risks—by replacing heavy Docker containers with a native, high-performance **Dynamic Gateway**.

**The Core Promise:** *"MCP Scooter runs in your system tray, manages your professional and personal AI identities, and instantly spawns tools for any AI client (Cursor, Claude, Antigravity) with zero latency and \<50MB RAM usage."*

**Distribution:** The project includes a public-facing website (hosted on GitHub Pages) serving documentation, changelogs, and binary downloads directly from GitHub Releases.

## **2\. Problem Statement**

As MCP adoption explodes, developers face three compounding crises:

1. **Context Bloat (The "Hard-Coding" Trap):**  
   * Developers currently hard-code tool definitions into their agent's config.  
   * *Result:* Connecting 50 tools floods the LLM's context window with 50 unused definitions, degrading performance and increasing costs.  
2. **Configuration Fragmentation:**  
   * A developer uses **Cursor** for work, **Claude Desktop** for research, and **Google Antigravity** for side projects.  
   * *Result:* They must manually copy-paste JSON configurations and API keys across 8 different config files (config.toml, settings.json, .claude/config, etc.), leading to "Configuration Drift" and security leaks.  
3. **The "Docker Weight" Problem:**  
   * The current solution (Docker MCP Gateway) requires running a heavy Linux VM on Mac/Windows.  
   * *Result:* High RAM usage (2-4GB) and slow startup times (3-5s) make it unusable for local-first development.

## **3\. Product Vision & User Journey**

### **3.1 The Vision**

To become the **standard local runtime** for MCP. If MCP is the "USB port" for AI, **MCP Scooter is the Universal Hub**.

### **3.2 User Journey: "The Dual-Identity Developer"**

* **Meet Alex:** A Senior DevOps Engineer who uses **Cursor** for work and **Claude Desktop** for personal coding.  
* **09:00 AM (Work Mode):**  
  * Alex opens MCP Scooter. It sits in the tray.  
  * She selects the **"Work @ Corp"** profile.  
  * She opens Cursor. Cursor is configured to talk to localhost:6277.  
  * She asks Cursor: *"Check the Prod DB health."*  
  * **Scooter Action:** Scooter authenticates via the Work Profile, dynamically spawns the Postgres-Prod tool (in a secure sandbox), and pipes the result to Cursor. The LLM *never saw* the definitions for her personal Spotify tool.  
* **06:00 PM (Personal Mode):**  
  * Alex doesn't close Cursor. She just opens **Claude Desktop** (configured to localhost:6278).  
  * She asks Claude: *"Analyze my Spotify listening history."*  
  * **Scooter Action:** Scooter detects the request on the Personal Port (:6278). It spawns the Spotify-MCP tool using her personal API key.  
  * *Crucial:* Her work credentials never leaked to the personal session, and her personal tools never cluttered her work context.

## **4\. Detailed Feature Specifications**

### **4.1 Native Cross-Platform Application**

* **Requirement:** Must be a native binary, not an Electron wrapper or Docker container.  
* **OS Support:** Windows 11 (ARM/x64), macOS (Apple Silicon/Intel), Linux (Debian/RPM).  
* **Performance Budget:** \<50MB RAM idle, \<10ms tool startup time.  
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
    * For local tools that need user context (e.g., google-drive-mcp), Scooter acts as a **Token Manager**.
    * Scooter maintains a refresh loop for Google/Slack/GitHub tokens and injects them into the local tool process as environment variables (e.g., GOOGLE\_ACCESS\_TOKEN) at runtime.
* **Secure Credential Storage:** Scooter integrates with **macOS Keychain**, **Windows Credential Manager**, and **Linux Secret Service**. Tokens are never stored in plain text.

### **4.3 The "Scooter Gateway" (Dynamic Discovery Engine)**

This mimics the "Docker Dynamic Mode" but runs natively. Instead of hard-coding tools, **Scooter exposes a Discovery Protocol** to the agent.

* **The "Zero-Config" Experience:**
  * When a user installs Scooter, they don't need to manually "install" 50 tools.
  * Scooter simply connects to the AI Client and exposes **3 Primordial Tools**: scout\_find, scout\_add, scout\_remove.  
* **Autonomous Tool Loading (The "Anthropic Tool Search" Pattern):**  
  1. **Trigger:** User asks *"Check my linear issues"*.  
  2. **Search:** The Agent (Claude/Cursor) calls scout\_find(query="linear").  
  3. **Discovery:** Scooter searches the local registry and the "Scooter Store" (Community Catalog). It returns: *"Found 'linear-mcp'. Capabilities: Manage issues, view cycles."*  
  4. **Installation:** The Agent calls scout\_add("linear-mcp").  
  5. **Execution:** Scooter authenticates (via OAuth), spins up the WASM module, and **hot-swaps** the Linear tool definitions into the active session.
  6. **Usage:** The Agent can now call linear\_list\_issues().
* **Resource Hygiene:** Scooter monitors usage. If linear-mcp hasn't been used in 10 turns, Scooter automatically unloads it to save RAM and Context Window space.

### **4.4 One-Click Setup (The "Integrations" Tab)**

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

* **Implementation:** Scooter acts as a local proxy. It writes a configuration that points the client to `http://localhost:6277/sse` (Server-Sent Events) and includes the **Gateway API Key** in the request headers, effectively routing all traffic through Scooter securely.

### **4.5 Custom MCP & Export/Import**

* **Custom MCP Wizard:**  
  * UI to add local tools (e.g., "Run Python Script").  
  * Inputs: Command (python/node), Args, Env Vars.  
  * **Auth Wrapper:** Checkbox for *"Manage OAuth for this tool"*. If checked, Scooter handles the Google/Slack login flow and passes the token to the script.
  * **Validation:** Scooter dry-runs the tool to verify it speaks MCP protocol.  
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
  2. Scooter executes this script in a secure **V8 Isolate**.
  3. Scooter returns *only* the filtered, formatted result to the LLM.  
* **State Persistence:** Includes a "Volume" API so agents can download a dataset to a temp folder, process it, and delete it, without the data ever touching the Context Window.

## **5\. Exposed Tools & Capabilities Registry**

Unlike standard MCP servers that expose a static list, Scooter exposes a dynamic hierarchy. The following **Primordial Tools** are intrinsic to the platform and always available.

### **5.1 The "Meta-Layer" (Discovery Tools)**

Every AI client connected to Scooter sees these tools by default.

* **scout\_find(query: string)**  
  * *Description:* Searches the Local Registry and Community Catalog for tools.  
  * *Returns:* Tool Names, Descriptions, and Installation status.  
* **scout\_add(tool\_name: string)**  
  * *Description:* Installs and enables an MCP tool for the current session.  
  * *Action:* Authenticates (if needed), loads the WASM, and injects tool definitions.  
* **scout\_remove(tool\_name: string)**  
  * *Description:* Unloads a tool to free up context window space.  
* **scout\_list\_active()**  
  * *Description:* Returns a list of currently loaded tools.

### **5.2 The "Core Suite" (Native Implementations)**

Scooter includes high-performance, native Go implementations of these standard utilities.

* **scout\_filesystem**: Safe read/write with strict path scoping.  
* **scout\_fetch**: Headless browser/HTTP client for web retrieval.  
* **scout\_code\_interpreter**: The "Code Mode" runtime for executing ephemeral scripts.

## **6\. Competitive Landscape**

| Feature | Docker MCP | MetaMCP | Manual Config | MCP Scooter |
| :---- | :---- | :---- | :---- | :---- |
| **Primary Use Case** | Enterprise Infra | Server-side Proxy | Hobbyist | **Pro Developer** |
| **Architecture** | Linux Containers | Docker Container | Local Process | **Native Binary \+ WASM** |
| **Discovery** | ✅ mcp-find (Container) | ❌ | ❌ | **✅ scout\_find (WASM)** |
| **Code Mode** | ✅ (Docker Sandbox) | ❌ | ❌ | **✅ (V8 Isolate)** |
| **Profile Support** | ❌ (Env Vars only) | ❌ (Namespace only) | ❌ (Manual switching) | **✅ (First-class UI)** |
| **Resource Usage** | Heavy (2GB+) | Heavy (Server) | Light | **Ultra-Light (\<50MB)** |
| **One-Click Setup** | ❌ | ❌ | ❌ | **✅ (8+ Clients)** |
| **Dynamic Loading** | ✅ (Slow) | ✅ | ❌ | **✅ (Instant)** |

## **7\. Technical Architecture (The "Go-WASM" Stack)**

### **7.1 Backend (Go)**

* **net/http**: Handles SSE (Server-Sent Events) connections from clients (Claude, Cursor).  
* **goroutines**: Manages concurrent profile servers (Port 8080, 8081).  
* **fsnotify**: Watches config files for real-time updates.  
* **oauth2\_client**: Implements RFC 8414 (Metadata Discovery) & RFC 7636 (PKCE) to authenticate against Remote MCP Servers.

### **7.2 Isolation Engine**

* **wazero**: Runs WASM-compiled MCP servers.  
* **v8go**: Runs JavaScript-based "Code Mode" scripts.  
* **os/exec**: Fallback for running local Node/Python scripts (Legacy Mode).

### **7.3 Frontend**

* **Tauri (Rust)**: Wraps a React/Tailwind web app into a native window.  
* **Communication:** Inter-Process Communication (IPC) between Tauri frontend and Go backend.

## **8\. Roadmap & Phasing**

* **Phase 1 (MVP - Foundation & Gateway):**  
  * Release of Native App (Win/Mac/Linux).  
  * Implementation of Profile Management System with Keychain integration.  
  * **Secure Gateway:** API Key authentication for IDE-to-Scooter communication.
  * **OAuth 2.0 Handler** implementation for key providers (Google, GitHub, Slack).  
  * One-Click Setup integrations for Cursor & Claude.  
  * Deployment of "Scooter Gateway" protocol (scout_find, scout_add) for dynamic lazy-loading.  
  * **Tool Playground** for manual testing.  
* **Phase 2 (Skills & Ecosystem):**  
  * **"Scooter Skills Library":** A "One-Click" Marketplace for AI Agent Skills.  
    * *Concept:* Instead of installing 5 separate tools, users install a "Skill" (e.g., "Full Stack Dev Skill" or "Data Analyst Skill").  
    * *Action:* Scooter automatically downloads and configures the necessary WASM bundles (e.g., postgres-mcp, python-mcp, browser-mcp) and sets up the System Prompt for that specific role.  
  * "Scooter Store" (Community registry of WASM tools).  
  * One-Click Setup for remaining clients (Zed, Antigravity, etc.).  
* **Phase 3 (Enterprise):**  
  * Team Sync (Share profiles via encrypted cloud config).  
  * Advanced Audit Logs & Compliance features.

## **9\. Web Presence & Distribution (Open Source)**

The project includes a front-facing website hosted on **GitHub Pages** to serve as the single source of truth for users and contributors.

### **9.1 Hosting Strategy**

* **Domain:** mcp-scooter.com (CNAME pointing to mcp-scooter.github.io).  
* **Deployment:** Automated via GitHub Actions on push to main.  
* **Technology:** Static HTML with Tailwind CSS (Single Page Application architecture for speed).

### **9.2 Website Structure**

* **Landing Page:**  
  * Value Proposition: "Native, Lightweight, Dynamic."  
  * Visuals: Terminal recording showing scout find and scout profile commands.  
  * Call to Action: Smart "Download" button that detects User OS (Mac/Win/Linux).  
* **Documentation Hub:**  
  * **Installation:** brew, winget, and curl scripts.  
  * **Configuration Guide:** Detailed syntax for profiles.yaml.  
  * **Tool Authoring:** Guide on how to compile existing Python tools to WASM for Scooter.  
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
* **LICENSE:**  
  * **PolyForm Shield 1.0.0** — Allows free use and modification, but prohibits competing products/services.  
  * Users can build products *with* MCP Scooter, but cannot fork it to create a competing MCP gateway.  
* **CONTRIBUTING.md:**  
  * Instructions for setting up the Go/Rust dev environment.  
  * Guidelines for submitting new tools to the "Scooter Store."  
* **CHANGELOG.md:**  
  * Maintained automatically via Semantic Release, but reflected on the website.