# MCP Scooter Architecture

This document describes the architecture of MCP Scooter, focusing on the tool lifecycle, verification process, and state synchronization.

## Core Components

### 1. Registry (Disk)
The registry consists of JSON files stored on disk (typically in `AppData/Roaming/mcp-scooter/registry/`).
- **Official Registry**: Pre-defined tool definitions (e.g., `brave-search.json`).
- **Custom Registry**: User-defined tool definitions.

These files are the **Source of Truth** for tool metadata, including:
- Name, title, description.
- Authorization requirements (API keys).
- Runtime configuration (command, args).
- **Discovered Tools**: The actual functions provided by the MCP server (updated during verification).
- **Verification Metadata**: `verified_at` timestamp.

### 2. Discovery Engine (In-Memory)
The `DiscoveryEngine` (defined in `internal/domain/discovery/discovery.go`) manages the active state of tools for a specific profile.
- **`registry`**: A slice of `ToolDefinition` loaded from disk.
- **`toolToServer`**: A map that links specific tool names (e.g., `brave_web_search`) to their parent server (e.g., `brave-search`).
- **`activeServers`**: A map of running `ToolWorker` instances (e.g., `StdioWorker`).

### 3. Control Server (API)
The `ControlServer` (defined in `internal/api/server.go`) handles management requests from the UI.
- **`handleVerifyTool`**: Triggers the verification process.
- **`handleGetTools`**: Returns the current list of tools.

---

## Tool Lifecycle & Synchronization

### 1. Initialization
When MCP Scooter starts:
1.  The backend ensures official registry files exist in the user's AppData folder.
2.  The `DiscoveryEngine` loads all JSON files from the registry into memory.
3.  The `toolToServer` map is populated based on the `tools` array in each JSON file.

### 2. Verification Process
When a user clicks **Verify** in the UI:
1.  **Handshake**: The backend starts the MCP server process and performs the `initialize` handshake.
2.  **Discovery**: The backend calls `tools/list` on the server to get the *actual* tool names and schemas.
3.  **Persistence**: The backend updates the registry JSON file on disk with the discovered tools and a `verified_at` timestamp.
4.  **Synchronization**:
    - The backend triggers `ReloadRegistry()` on all active `DiscoveryEngine` instances.
    - `ReloadRegistry()` re-reads the JSON files from disk.
    - **Crucial Step**: It rebuilds the `toolToServer` map to ensure that the new tool names (e.g., `brave_web_search`) are correctly mapped to the server.

### 3. Tool Invocation
When a user clicks **Invoke Tool**:
1.  The UI sends the tool name (e.g., `brave_web_search`) to the backend.
2.  The `DiscoveryEngine` looks up the tool name in its `toolToServer` map.
3.  If found, it routes the call to the appropriate `activeServer`.
4.  **Exact Matching**: Tool names are matched exactly. No normalization (like replacing underscores with dashes) is performed to avoid mismatches with the server's expectations.

---

## Troubleshooting

### "Unknown Tool" Error
If `Invoke Tool` returns an "Unknown Tool" error after verification:
1.  **Synchronization Failure**: Check if `ReloadRegistry()` was called and completed successfully.
2.  **Deadlock**: Ensure there are no recursive mutex locks in the `DiscoveryEngine` (e.g., `ReloadRegistry` calling `loadRegistry` which calls `Register`).
3.  **Name Mismatch**: Ensure the UI is sending the *exact* name discovered during verification.

### Spinning "Verify" Button
If the verification process hangs:
1.  **Deadlock in Start**: Check if `StdioWorker.Start` is holding a lock while waiting for a handshake that also needs the lock.
2.  **Server Timeout**: The MCP server may be failing to start or respond to the `initialize` request. Check the backend logs for stderr output from the server.
