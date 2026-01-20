# Docker MCP Reverse Engineering

This directory contains tools to capture and analyze the MCP protocol messages between Cursor and the Docker MCP Toolkit gateway.

## Setup

1. Copy the config from `cursor-mcp-config.json` into your `~/.cursor/mcp.json`
2. Restart Cursor to pick up the new MCP server
3. Use the `docker-mcp` server in Cursor/Cline

## Files

- `mcp-logger.js` - Node.js script that wraps any stdio MCP server and logs all JSON-RPC messages
- `cursor-mcp-config.json` - Sample Cursor MCP configuration to use the logger
- `mcp-protocol-log.jsonl` - Output log file (created when you use the server)

## Log Format

Each line in `mcp-protocol-log.jsonl` is a JSON object:

```json
{
  "timestamp": "2026-01-20T04:40:00.000Z",
  "direction": "CLIENT->SERVER" | "SERVER->CLIENT",
  "data": "raw message string",
  "parsed": { ... }  // parsed JSON if valid
}
```

## What to Look For

1. **tools/list response** - How does Docker MCP return tools after `docker_mcp_add_server`?
2. **Tool attribution** - Is there any metadata indicating which "sub-server" a tool belongs to?
3. **Tool call routing** - When calling `brave_web_search`, does the client route it correctly?

## Usage

After setting up, use Cline to:
1. Call `docker_mcp_list_servers` to see available servers
2. Call `docker_mcp_add_server` to add brave-search
3. Call `brave_web_search` to perform a search
4. Check `mcp-protocol-log.jsonl` for the captured messages
