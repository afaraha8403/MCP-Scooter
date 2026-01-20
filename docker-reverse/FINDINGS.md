# Docker MCP Gateway - Reverse Engineering Findings

## Test Date: 2026-01-20

## Summary

We reverse-engineered the Docker MCP Toolkit gateway to understand how it handles tool proxying and compare it to MCP Scooter's approach.

## Key Findings

### 1. No Dynamic Server Management

Docker MCP Gateway does **NOT** have `docker_mcp_add_server` or `docker_mcp_remove_server` tools.

```json
{"code":-32602,"message":"unknown tool \"docker_mcp_add_server\""}
```

Servers are enabled/disabled via CLI commands (`docker mcp server enable brave`) and the gateway loads all enabled servers at startup.

### 2. Tools Are Pre-Loaded

The `tools/list` response immediately after `initialize` includes all tools from all enabled servers. There's no runtime activation step.

### 3. No Server Attribution in Tool Schema

Tools returned by `tools/list` have no metadata indicating which "sub-server" they belong to:

```json
{
  "name": "brave_web_search",
  "title": "brave_web_search",
  "description": "...",
  "inputSchema": {...},
  "annotations": {
    "openWorldHint": true,
    "title": "Brave Web Search"
  }
}
```

No `server`, `source`, `provider`, or similar field exists.

### 4. Gateway Acts as True Proxy

The client only knows about one MCP server: `docker-mcp`. All tool calls go to this server, which internally routes to the appropriate container.

### 5. Tool List from Docker MCP Gateway (with brave enabled)

- `brave_image_search`
- `brave_local_search`
- `brave_news_search`
- `brave_summarizer`
- `brave_video_search`
- `brave_web_search`

## Comparison with MCP Scooter

| Feature | Docker MCP | MCP Scooter |
|---------|-----------|-------------|
| Server activation | CLI only (`docker mcp server enable`) | Runtime via `scooter_add` |
| Tool discovery | Pre-loaded at startup | Dynamic via `scooter_find` |
| Server removal | CLI only | Runtime via `scooter_remove` |
| Tool attribution | None | None (same issue) |

## The Cline Problem

When MCP Scooter returns `brave_web_search` in `tools/list`, Cline incorrectly attributes it to a separate "brave-search" MCP server instead of routing through `mcp-scooter`.

This is likely a **Cline bug** where it:
1. Parses tool names looking for patterns (e.g., `brave_*`)
2. Infers a server name from the prefix
3. Creates a phantom server entry

### Evidence

In the user's Cline session:
```
Cline wants to use a tool on the `brave-search` MCP server:
brave_web_search
```

But `brave-search` is not configured in Cursor's MCP settings. Only `mcp-scooter` is configured.

## Recommendations for MCP Scooter

1. **Report Cline Bug**: This appears to be a client-side issue where Cline incorrectly infers server names from tool names.

2. **Consider Tool Namespacing**: Prefix all proxied tools with `scooter_` to make it clear they belong to the scooter gateway:
   - `scooter_brave_web_search`
   - `scooter_brave_local_search`
   
   This is ugly but would prevent client confusion.

3. **Add Server Metadata (Non-Standard)**: Add a custom field to tool schemas:
   ```json
   {
     "name": "brave_web_search",
     "_scooter_server": "brave-search",
     ...
   }
   ```
   
   This wouldn't help with Cline's bug but could help other clients.

4. **Document the Limitation**: Make it clear that some MCP clients may have issues with dynamically-added tools.

## Raw Protocol Capture

See `manual-test-log.jsonl` for the complete JSON-RPC message exchange.
