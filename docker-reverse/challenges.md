# MCP Gateway Proxy Challenge: Tool Routing Problem

## Executive Summary

MCP Scooter is designed to be a **single MCP server** that acts as a gateway/proxy to multiple underlying tool servers. However, when dynamically-added tools appear in the tool list, MCP clients like Cline incorrectly route tool calls to phantom servers instead of through the gateway.

This document captures all the data needed to analyze and solve this problem.

---

## Table of Contents

1. [Background: What is MCP Scooter?](#background-what-is-mcp-scooter)
2. [The Architecture](#the-architecture)
3. [The Problem](#the-problem)
4. [Evidence and Logs](#evidence-and-logs)
5. [Comparison with Docker MCP Toolkit](#comparison-with-docker-mcp-toolkit)
6. [Root Cause Analysis](#root-cause-analysis)
7. [Why This Matters](#why-this-matters)
8. [Potential Solutions](#potential-solutions)
9. [Open Questions](#open-questions)

---

## Background: What is MCP Scooter?

MCP Scooter is a **Model Context Protocol (MCP) gateway** that allows AI agents to dynamically discover and use tools without pre-configuring every tool server in the client application.

### The Value Proposition

Instead of configuring 10+ MCP servers in your IDE:

```json
{
  "mcpServers": {
    "brave-search": { "command": "npx", "args": ["@anthropic/mcp-brave-search"] },
    "github": { "command": "npx", "args": ["@anthropic/mcp-github"] },
    "filesystem": { "command": "npx", "args": ["@anthropic/mcp-filesystem"] },
    "postgres": { "command": "npx", "args": ["@anthropic/mcp-postgres"] },
    // ... many more
  }
}
```

You configure just ONE server:

```json
{
  "mcpServers": {
    "mcp-scooter": {
      "type": "sse",
      "url": "http://127.0.0.1:6277/profiles/work/sse"
    }
  }
}
```

MCP Scooter then provides "meta-tools" that let the AI agent discover and activate tools on-demand:

- `scooter_find` - Search for available tools
- `scooter_add` - Activate a tool server
- `scooter_remove` - Deactivate a tool server
- `scooter_list_active` - List currently active tools

### How It Should Work

1. User asks: "Search for AI news"
2. AI calls `scooter_find("search")` → Returns `brave-search` is available
3. AI calls `scooter_add("brave-search")` → Activates the brave-search server
4. AI calls `brave_web_search("AI news")` → **This call should go through MCP Scooter**
5. MCP Scooter proxies the request to the underlying brave-search server
6. Results are returned to the AI

---

## The Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        AI Client (Cline)                         │
│                                                                  │
│  Configured MCP Servers:                                         │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ mcp-scooter (SSE) → http://127.0.0.1:6277/profiles/work/sse ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                                │
                                │ All MCP traffic
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      MCP Scooter Gateway                         │
│                                                                  │
│  Built-in Tools:           Active Tool Servers:                  │
│  ┌──────────────────┐     ┌──────────────────────────────────┐  │
│  │ scooter_find     │     │ brave-search (activated)         │  │
│  │ scooter_add      │     │  └─ brave_web_search             │  │
│  │ scooter_remove   │     │  └─ brave_local_search           │  │
│  │ scooter_list     │     │                                  │  │
│  └──────────────────┘     │ github (not activated)           │  │
│                           │ context7 (not activated)         │  │
│                           └──────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                │
                                │ Proxied to underlying server
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│              Brave Search MCP Server (npx process)               │
│                                                                  │
│  Actual implementation of brave_web_search, brave_local_search   │
└─────────────────────────────────────────────────────────────────┘
```

---

## The Problem

### What Happens

After `scooter_add("brave-search")` succeeds, the `brave_web_search` tool appears in MCP Scooter's `tools/list` response. However, when the AI tries to call `brave_web_search`, **the call never reaches MCP Scooter**.

### Observed Behavior in Cline

```
Cline wants to use a tool on the `mcp-scooter` MCP server:
scooter_find
Arguments: {"query":"search news AI"}
Response: [brave-search is available...]

Cline wants to use a tool on the `mcp-scooter` MCP server:
scooter_add  
Arguments: {"tool_name":"brave-search"}
Response: [Server 'brave-search' is now active...]

Cline wants to use a tool on the `brave-search` MCP server:  ← WRONG!
brave_web_search
Arguments: {"query": "AI news January 2026"}
[No response - hangs indefinitely]
```

### The Critical Issue

Notice the third call says **`brave-search` MCP server**, not `mcp-scooter`.

Cline is attempting to route `brave_web_search` to a server called `brave-search`, but:
- `brave-search` is NOT configured in the user's MCP settings
- Only `mcp-scooter` is configured
- The call should go to `mcp-scooter`, which would proxy it to the underlying brave-search process

### What the Logs Show

MCP Scooter's logs show:
1. ✅ `scooter_find` request received and processed
2. ✅ `scooter_add` request received and processed  
3. ❌ `brave_web_search` request **never received**

The request is being sent to a non-existent server, so it never arrives at the gateway.

---

## Evidence and Logs

### MCP Scooter Configuration (User's ~/.cursor/mcp.json)

```json
{
  "mcpServers": {
    "mcp-scooter": {
      "type": "sse",
      "url": "http://127.0.0.1:6277/profiles/test-brave/sse"
    }
  }
}
```

Note: There is NO `brave-search` entry. Only `mcp-scooter` is configured.

### MCP Scooter's tools/list Response (After scooter_add)

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "result": {
    "tools": [
      {
        "name": "scooter_find",
        "description": "Searches the Local Registry..."
      },
      {
        "name": "scooter_add",
        "description": "Activates an MCP tool server..."
      },
      {
        "name": "scooter_remove",
        "description": "Deactivates an MCP tool server..."
      },
      {
        "name": "scooter_list_active",
        "description": "Lists all currently active..."
      },
      {
        "name": "brave_web_search",
        "description": "Performs a web search using the Brave Search API..."
      },
      {
        "name": "brave_local_search", 
        "description": "Searches for local businesses..."
      }
    ]
  }
}
```

The tools are returned as a flat list with no server attribution metadata.

### Direct Test (Bypassing Cline)

When we call `brave_web_search` directly through MCP Scooter (not through Cline), it works perfectly:

```powershell
$body = '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"brave_web_search","arguments":{"query":"AI news"}}}'
Invoke-RestMethod -Uri "http://127.0.0.1:6277/profiles/test-brave/message" -Method POST -Body $body
```

Response: ✅ Returns search results successfully

This proves the gateway is working correctly. The problem is client-side routing.

---

## Comparison with Docker MCP Toolkit

We reverse-engineered Docker's MCP Toolkit to see how they handle this.

### Docker MCP Gateway Architecture

Docker MCP Toolkit also uses a gateway pattern:
- Single MCP server: `docker-mcp`
- Proxies to containerized tool servers (brave, github, etc.)

### Key Difference: No Dynamic Activation

Docker MCP does NOT have `docker_mcp_add_server` or similar tools:

```json
{"code":-32602,"message":"unknown tool \"docker_mcp_add_server\""}
```

Servers are enabled via CLI commands BEFORE the gateway starts:
```bash
docker mcp server enable brave
docker mcp gateway run
```

### Docker's tools/list Response

```json
{
  "tools": [
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
  ]
}
```

No server attribution metadata. Same flat structure as MCP Scooter.

### Does Docker Have This Problem?

**Unknown.** Docker's model avoids the issue by:
1. Pre-loading all tools at startup
2. Not allowing runtime tool activation
3. Tools are always present from the first `tools/list` call

The dynamic activation pattern in MCP Scooter may be triggering client-side heuristics that don't apply to Docker's static model.

---

## Root Cause Analysis

### Hypothesis 1: Cline Infers Server Names from Tool Names

Cline may be using heuristics to group tools by server:

1. Sees tool named `brave_web_search`
2. Extracts prefix `brave`
3. Assumes there's a server called `brave-search` or `brave`
4. Routes calls to this inferred server

Evidence supporting this:
- The phantom server is called `brave-search`
- The tool is called `brave_web_search`
- The naming pattern matches

### Hypothesis 2: Cline Caches Server-Tool Mappings

Cline may cache which server provides which tool:

1. Initial `tools/list` returns only `scooter_*` tools
2. Cline maps these to `mcp-scooter`
3. After `scooter_add`, new tools appear
4. Cline doesn't update its mapping, or creates a new server entry

### Hypothesis 3: SSE tools/list_changed Handling

After `scooter_add`, MCP Scooter sends a `notifications/tools/list_changed` event.

Cline may:
1. Receive the notification
2. Call `tools/list` to get updated tools
3. Incorrectly process the new tools as belonging to a different server

### Hypothesis 4: MCP Protocol Limitation

The MCP protocol may not have a standard way to indicate tool provenance:

```json
{
  "name": "brave_web_search",
  "server": "mcp-scooter",  // ← This field doesn't exist in MCP spec
  ...
}
```

Without this, clients must infer server ownership, which can fail.

---

## Why This Matters

### This Isn't Just a Cline Bug

Cline is one of the most popular MCP clients and is built following MCP best practices. If this issue exists in Cline, it likely exists in:

- Claude Desktop
- VS Code's MCP support
- Zed's MCP integration
- Any other MCP client

### The Gateway Pattern is Valuable

The ability to have a single MCP server that proxies to many tools is extremely valuable:

1. **Simpler configuration** - One server instead of many
2. **Dynamic tool discovery** - AI can find and use tools on-demand
3. **Centralized management** - One place to manage credentials, permissions, logging
4. **Resource efficiency** - Only activate tools when needed

If clients can't properly route tools through gateways, this pattern becomes unusable.

### MCP Scooter's Unique Challenge

MCP Scooter's dynamic activation model (`scooter_add`) is more advanced than Docker's static model. This means:

1. Tools appear/disappear at runtime
2. The tool list changes during a session
3. Clients must handle dynamic tool lists correctly

This is a harder problem than Docker's static approach.

---

## Potential Solutions

### Solution 1: Tool Namespacing (Workaround)

Prefix all proxied tools with `scooter_`:

```json
{
  "name": "scooter_brave_web_search",  // Instead of brave_web_search
  ...
}
```

**Pros:**
- Clearly indicates tools belong to scooter
- May prevent client heuristics from inferring wrong server

**Cons:**
- Ugly tool names
- AI must learn new names
- Doesn't fix the underlying client issue

### Solution 2: Report and Fix Client Bug

File issues with Cline, Claude Desktop, etc. to fix their tool routing logic.

**Pros:**
- Fixes the root cause
- Benefits all gateway-style MCP servers

**Cons:**
- Depends on third-party developers
- May take time to fix
- Each client may have different bugs

### Solution 3: Add Server Metadata (Non-Standard)

Add a custom field to tool schemas:

```json
{
  "name": "brave_web_search",
  "_mcp_gateway": "mcp-scooter",
  "_mcp_source_server": "brave-search",
  ...
}
```

**Pros:**
- Provides explicit provenance information
- Could become a standard if adopted

**Cons:**
- Non-standard, clients may ignore it
- Doesn't help with current clients

### Solution 4: Propose MCP Protocol Extension

Work with Anthropic to add server attribution to the MCP spec:

```json
{
  "tools": [
    {
      "name": "brave_web_search",
      "providedBy": "brave-search",  // New standard field
      "routeThrough": "mcp-scooter", // New standard field
      ...
    }
  ]
}
```

**Pros:**
- Proper long-term solution
- Benefits entire MCP ecosystem

**Cons:**
- Slow process
- Requires ecosystem adoption

### Solution 5: Static Pre-Loading (Docker's Approach)

Abandon dynamic activation. Pre-load all allowed tools at startup.

**Pros:**
- Avoids the dynamic tool list issue
- Simpler client behavior

**Cons:**
- Loses the dynamic discovery value proposition
- Higher resource usage
- Less flexible

---

## Open Questions

1. **Does Docker MCP Toolkit have this problem?**
   - We couldn't test with Cline due to a Docker bug
   - Need to verify if static tool loading avoids the issue

2. **How does Cline determine tool-to-server mapping?**
   - Need to examine Cline's source code
   - Is it using tool name heuristics?
   - Is it caching mappings incorrectly?

3. **Do other MCP clients have this issue?**
   - Test with Claude Desktop
   - Test with VS Code MCP
   - Test with Zed

4. **Is there an MCP spec for tool provenance?**
   - Review MCP specification
   - Check if there's a standard way to indicate tool source

5. **What triggers Cline to create a phantom server?**
   - Is it the `tools/list_changed` notification?
   - Is it the tool naming pattern?
   - Is it something in the tool schema?

---

## Files in This Directory

| File | Description |
|------|-------------|
| `manual-test.js` | Script to send JSON-RPC messages to Docker MCP gateway |
| `manual-test-log.jsonl` | Captured protocol messages from Docker MCP test |
| `mcp-logger.js` | Wrapper to log all MCP stdio traffic |
| `mcp-protocol-log.jsonl` | Captured protocol messages from Cline test |
| `cursor-mcp-config.json` | Sample Cursor MCP configuration |
| `FINDINGS.md` | Summary of Docker MCP reverse engineering |
| `challenges.md` | This document |

---

## Next Steps

1. **Reproduce with Docker MCP** - Verify if Docker's static model avoids the issue
2. **Test with other clients** - Claude Desktop, VS Code, Zed
3. **Examine Cline source** - Understand how it routes tool calls
4. **File bug reports** - With evidence from this analysis
5. **Implement workaround** - Tool namespacing as interim solution
6. **Engage MCP community** - Discuss gateway patterns and tool provenance

---

## Contact

This analysis was conducted as part of the MCP Scooter project by Balacode.io.

For questions or contributions, see the main project repository.
