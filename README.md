<p align="center">
  <img src="desktop/public/logo/logo-dark.svg" alt="MCP Scout Logo" width="280" />
</p>

<h1 align="center">MCP Scout (Scooter)</h1>

<p align="center">
  <strong>The Universal Operating System for Model Context Protocol</strong>
</p>

<p align="center">
  <a href="#-why-mcp-scout">Why?</a> •
  <a href="#-features">Features</a> •
  <a href="#-how-its-different">How It's Different</a> •
  <a href="#-getting-started">Getting Started</a> •
  <a href="#-contributing">Contributing</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/status-active%20development-orange?style=flat-square" alt="Status: Active Development" />
  <img src="https://img.shields.io/badge/license-PolyForm%20Shield-purple?style=flat-square" alt="License: PolyForm Shield" />
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey?style=flat-square" alt="Platform" />
  <img src="https://img.shields.io/badge/go-1.24+-00ADD8?style=flat-square&logo=go" alt="Go 1.24+" />
  <img src="https://img.shields.io/badge/rust-tauri-orange?style=flat-square&logo=rust" alt="Tauri" />
</p>

---

> ⚠️ **Active Development Notice**  
> MCP Scout is under active development. APIs, features, and documentation may change. We're building in public and welcome early adopters and contributors!

---

## 🎯 Why MCP Scout?

As AI agents become more powerful, developers face a growing crisis:

### The Problems We're Solving

| Problem | What Happens Today | MCP Scout Solution |
|---------|-------------------|-------------------|
| **Context Bloat** | Connecting 50 tools floods your LLM with 50 unused definitions, **consuming your context window**, degrading performance and burning tokens | **Dynamic Discovery** — Tools load on-demand. Your LLM only sees what it needs for the task at hand. |
| **Configuration Chaos** | Using Cursor for work + Claude for personal? Switching between personal and work accounts (like Postman or Slack) requires manually swapping API keys and JSON configs across 8 different files | **One Hub, All Clients** — Use **Profiles** to isolate accounts. Switch context once, and all your tools follow. |
| **The Docker Tax** | Docker MCP Gateway needs 2-4GB RAM and 3-5 seconds to start. That's not "local-first." | **Native & Lightweight** — <50MB RAM, <10ms tool startup. No containers. |
| **Security Leaks** | Work credentials mixed with personal tools. No isolation. No audit trail. | **Profile Isolation** — Work and personal identities never cross-contaminate. |

### The Vision

If MCP is the "USB port" for AI, **MCP Scout is the Universal Hub**.

```
┌─────────────────────────────────────────────────────────────────┐
│                         MCP Scout                                │
│                    (System Tray / Menu Bar)                      │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ Work Profile │  │Personal Prof.│  │ Side Project │          │
│  │   :6277      │  │    :6278     │  │    :6279     │          │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                 │                 │                   │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐          │
│  │ Cursor       │  │ Claude       │  │ VS Code      │          │
│  │ Zed          │  │ Desktop      │  │ Gemini CLI   │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

---

## ✨ Features

### 🔍 Dynamic Tool Discovery
No more hard-coding tool definitions. Scout exposes three "primordial tools" to any AI client, enabling **"auto-choosing"** of tools based on the context of your question:

- **`scout_find`** — Search for tools by capability
- **`scout_add`** — Install and enable tools on-demand  
- **`scout_remove`** — Unload tools to free context space

**How it works:** Your LLM taps into the Scout discovery tool → It gets a list of available capabilities → It auto-chooses the right tool for your specific question → Scout loads only what's needed. This avoids loading the entire toolset and keeps your context window clean.

Your agent asks for "database tools" → Scout finds them → Agent installs what it needs → Done.

### 👤 Profile-Based Identity Management
Create isolated environments for different contexts:

```yaml
profiles:
  - id: work-corp
    port: 6277
    allow_tools: ["jira-mcp", "postgres-prod"]
    env:
      AWS_REGION: "us-east-1"
      
  - id: personal
    port: 6278
    allow_tools: ["spotify-mcp", "notion-mcp"]
```

Work credentials never leak to personal sessions. Personal tools never clutter work context.

### 🔌 One-Click Client Integration
Scout auto-configures your AI clients:

| Client | Status |
|--------|--------|
| Cursor | ✅ Supported |
| Claude Desktop | ✅ Supported |
| VS Code (MCP Extension) | ✅ Supported |
| Claude Code | ✅ Supported |
| Gemini CLI | ✅ Supported |
| Zed | ✅ Supported |
| Google Antigravity | 🔜 Coming Soon |

### 🔐 Secure by Design
- **Native Keychain Integration** — macOS Keychain, Windows Credential Manager, Linux Secret Service
- **OAuth 2.0/2.1 Handler** — Scout handles auth flows so your AI clients don't have to
- **Human-in-the-Loop** — Approve sensitive operations before they execute

### ⚡ Native Performance
- **<50MB RAM** idle
- **<10ms** tool startup
- **No Docker** — Pure native binary + WASM isolation
- **Feels like part of your OS** — WinUI 3 design on Windows, native on macOS/Linux

---

## 🆚 How It's Different

### vs. Docker MCP Toolkit

| Aspect | Docker MCP | MCP Scout |
|--------|-----------|-----------|
| **Architecture** | Linux containers on VM | Native binary + WASM |
| **RAM Usage** | 2-4GB | <50MB |
| **Startup Time** | 3-5 seconds | <10ms |
| **Target User** | Enterprise DevOps | Individual developers |
| **Profile Support** | Environment variables only | First-class UI |
| **One-Click Setup** | ❌ | ✅ 8+ clients |

Docker MCP is excellent for enterprise infrastructure and server deployments. **MCP Scout is for your laptop** — the developer who wants AI tools that feel instant and native.

### vs. MetaMCP

MetaMCP is a server-side proxy that aggregates MCP servers. It's great for teams running centralized infrastructure.

**MCP Scout is local-first.** It runs in your system tray, manages your personal credentials, and gives you instant tool access without network round-trips.

### vs. Manual Configuration

You *could* manually edit `~/.cursor/mcp.json`, `~/Library/.../claude_desktop_config.json`, `.vscode/mcp.json`...

Or you could click one button in Scout and have all your clients configured in seconds.

---

## 🚀 Getting Started

### 📦 Download (Recommended)

> **🚧 Releases Coming Soon!**  
> Pre-built installers for Windows, macOS, and Linux are on the way. You'll be able to download and run MCP Scout with a single click — no build tools required.
>
> ⭐ **Star this repo** to get notified when the first release drops!

---

### 🛠️ Build from Source (For Contributors)

Want to contribute or hack on MCP Scout? Here's how to build it yourself.

> **Note:** Building from source is intended for development purposes. For regular use, wait for the official releases above.

#### Prerequisites

- **Go 1.24+** — [Download](https://go.dev/dl/)
- **Node.js 18+** — [Download](https://nodejs.org/)
- **Rust** (for Tauri) — [Install](https://rustup.rs/)

#### Build & Run
```bash
# Clone the repository
git clone https://github.com/mcp-scooter/scooter.git
cd scooter

# Install dependencies
make deps
./tasks.ps1 deps

# Build everything
make all
./tasks.ps1 all

# Run in development mode
make dev
./tasks.ps1 dev
```

#### Validate Registry

```bash
# Validate all MCP definitions
make validate

# Strict mode (warnings = errors)
make validate-strict
```

#### Releasing

MCP Scout uses GitHub Actions for automated releases. To trigger a release:

```bash
# Release a stable version (e.g., 1.0.0)
make release
./tasks.ps1 release

# Release a beta version (e.g., 1.0.0-beta.1)
make release-beta
./tasks.ps1 release-beta
```

This will automatically tag the current commit and push it to GitHub, triggering the build and release workflow.

---

## 📁 Project Structure

```
MCP Scout/
├── appdata/
│   ├── clients/        # AI client configurations
│   ├── registry/       # MCP server definitions (JSON)
│   └── schemas/        # JSON Schema for validation
├── cmd/
│   ├── scooter/        # Main application
│   └── validate-registry/  # Registry validation CLI
├── desktop/            # Tauri + React frontend
│   ├── src/            # React components
│   └── src-tauri/      # Rust/Tauri backend
├── internal/
│   ├── api/            # HTTP/SSE server
│   └── domain/         # Core business logic
│       ├── discovery/  # Tool discovery engine
│       ├── integration/# Client integrations
│       ├── profile/    # Profile management
│       └── registry/   # Registry validation
└── web/                # Public website
```

---

## 🗺️ Roadmap

### Current Status: **Phase 1 (MVP) — In Progress**

We're building the foundation. Here's what's done and what's next:

#### ✅ Completed

| Component | Status | Description |
|-----------|--------|-------------|
| **Registry Schema** | ✅ Done | JSON Schema for MCP server definitions |
| **Registry Validation** | ✅ Done | CLI tool to validate registry entries |
| **Profile Management** | ✅ Done | Create, update, delete profiles with persistence |
| **Discovery Engine** | ✅ Done | `scooter_find`, `scooter_add`, `scooter_remove`, `scooter_list_active` |
| **Code Interpreter** | ✅ Done | `scooter_code_interpreter` with V8 sandbox |
| **MCP Gateway** | ✅ Done | SSE server handling JSON-RPC for all profiles |
| **Client Integrations** | ✅ Done | Cursor, Claude Desktop, Claude Code, VS Code, Gemini CLI, Zed, Codex |
| **Tauri Desktop Shell** | ✅ Done | Native window with React frontend |
| **Keychain Integration** | ✅ Done | Secure credential storage (Windows/macOS/Linux) |

#### 🚧 In Progress

| Component | Status | Description |
|-----------|--------|-------------|
| **Desktop UI** | 🚧 Building | Profile management UI, tool browser, settings |
| **OAuth 2.0 Handler** | 🚧 Building | Automatic auth flows for Google, GitHub, Slack |
| **Tool Playground** | 🚧 Building | Manual tool testing interface |
| **WASM Runtime** | 🚧 Building | Run WASM-compiled MCP servers |

#### 📋 Phase 1 Remaining

- [ ] System tray integration (minimize to tray)
- [ ] Port conflict detection
- [ ] Human-in-the-loop approval UI
- [ ] Log inspector (Network tab for AI)

### Phase 2: Skills & Ecosystem

| Feature | Description |
|---------|-------------|
| **Scout Store** | Community registry of WASM tools |
| **Skills Library** | Pre-configured tool bundles ("Full Stack Dev", "Data Analyst") |
| **Remote MCP Support** | Connect to enterprise MCP gateways with OAuth 2.1 |
| **Antigravity Integration** | Google's AI client support |

### Phase 3: Enterprise

| Feature | Description |
|---------|-------------|
| **Team Sync** | Share profiles via encrypted cloud config |
| **Audit Logs** | Compliance-ready logging |
| **SSO Integration** | Enterprise identity providers |

---

## 🤝 Contributing

**We're building MCP Scout in public and we'd love your help!**

### Ways to Contribute

- 🐛 **Report bugs** — Found something broken? Open an issue.
- 💡 **Suggest features** — Have an idea? Let's discuss it.
- 📝 **Improve docs** — Documentation can always be better.
- 🔧 **Submit PRs** — Code contributions are welcome!
- 🎨 **Add MCP definitions** — Help grow the registry.

### Adding New MCP Definitions

1. Create a JSON file in `appdata/registry/{name}.json`
2. Follow the schema in `appdata/schemas/mcp-registry.schema.json`
3. Run `make validate` to verify
4. Submit a PR!

See `.doc/mcp-registry-specification.md` for the full specification.

### Development Setup

The project uses a tiered testing strategy. You can use **make** (macOS/Linux) or **tasks.ps1** (Windows PowerShell).

#### Level 1-2: Unit Tests & Validation
```bash
# Run all tests
make test
./tasks.ps1 test

# Run all unit tests (verbose)
make test-unit
./tasks.ps1 test-unit

# Test specific domains
make test-registry
./tasks.ps1 test-registry

# Generate HTML coverage report
make test-coverage
./tasks.ps1 test-coverage

# Validate registry definitions
make validate
./tasks.ps1 validate
```

#### Level 5: Meta-MCP Lifecycle
```bash
# Test the Meta-MCP primordial tools and lifecycle
make test-meta-mcp
./tasks.ps1 test-meta-mcp
```

#### Combined Checks
```bash
# Quick check before committing
make pre-commit
./tasks.ps1 pre-commit

# Full CI check
make ci
./tasks.ps1 ci
```

---

## 📬 Get in Touch

- **GitHub Issues** — For bugs and feature requests
- **GitHub Discussions** — For questions and ideas

---

## 📜 License

**PolyForm Shield 1.0.0** — See [LICENSE](LICENSE) for details.

**TL;DR:** You can use MCP Scout freely, build products with it, and modify it for your needs. You **cannot** offer it (or a fork) as a competing product or hosted service without permission.

---

<p align="center">
  <strong>MCP Scout</strong> — Native. Lightweight. Dynamic.
</p>

<p align="center">
  <sub>Crafted with ❤️ by <a href="https://balacode.io">Balacode.io</a></sub>
</p>
