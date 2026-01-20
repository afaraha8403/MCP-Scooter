# Changelog

All notable changes to MCP Scooter will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2026-01-20

### Changed
- Refactored MCP Registry structure: moved official definitions to `appdata/registry/official/`
- Enhanced MCP Registry schema with better validation rules and documentation fields
- Renamed primordial tools from `scout_*` to `scooter_*` for consistency:
  - `scooter_find`
  - `scooter_add`
  - `scooter_remove`
- Added new primordial tools to core engine:
  - `scooter_list_active` - List currently active tools and servers
  - `scooter_code_interpreter` - Sandboxed JavaScript execution environment
  - `scooter_filesystem` - Secure, scoped file operations
  - `scooter_fetch` - Local-first HTTP client
- Updated client integration documentation for Cursor, Claude Desktop, Claude Code, Zed, VS Code, and Gemini CLI

### Added
- New asset gallery for the public website
- Performance comparison metrics against legacy Docker-based toolkits

## [Unreleased] - 2026-01-19

### Added
- Initial project structure with Go backend and Tauri/React frontend
- MCP Registry schema and validation system
- Profile management with persistence
- Discovery engine with primordial tools:
  - `scooter_find` - Search for MCP tools
  - `scooter_add` - Install tools on-demand
  - `scooter_remove` - Unload tools
  - `scooter_list_active` - List active tools
  - `scooter_code_interpreter` - Sandboxed JavaScript execution
- MCP Gateway with SSE/JSON-RPC support
- Client integrations:
  - Cursor
  - Claude Desktop
  - Claude Code
  - VS Code
  - Gemini CLI
  - Zed
  - Codex
- Native keychain integration (Windows, macOS, Linux)
- Registry entries for:
  - Brave Search
  - Context7
  - GitHub

### In Progress
- Desktop UI (Tauri + React)
- OAuth 2.0 handler for third-party services
- WASM runtime for sandboxed MCP servers
- Tool Playground for manual testing

---

## Version History

*No releases yet. First release coming soon!*

---

*MCP Scooter is maintained by [Balacode.io](https://balacode.io)*
