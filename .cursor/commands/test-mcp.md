# Test MCP Server

Test an MCP server from the registry to verify it works correctly with the discovery engine.

## Input

Provide the **MCP name** (lowercase, matching the registry filename):
- e.g., `brave-search`, `github`, `context7`

## Workflow Steps

### Step 0: Pre-flight Checks

<step>
Before starting, verify all prerequisites:

1. **Verify registry file exists:**
   ```bash
   ls appdata/registry/official/{name}.json
   ```

2. **Validate against schema:**
   ```bash
   go run cmd/validate-registry/main.go appdata/registry/official/{name}.json
   ```

3. **Check Go environment:**
   ```bash
   go version
   ```

4. **Verify discovery package compiles:**
   ```bash
   go build ./internal/domain/discovery/...
   ```

**On failure:** Report the specific error and stop. Do not proceed with broken prerequisites.

**Output:** All pre-flight checks passed.
</step>

### Step 1: Registry Inspection

<step>
Read the registry file at `appdata/registry/official/{name}.json` and extract:

1. **Runtime configuration:**
   - Transport type: `stdio`, `wasm`, or `sse`
   - Command and arguments
   - Working directory (if specified)

2. **Authorization requirements:**
   - Auth type: `none`, `api_key`, `oauth2`, `bearer_token`, `custom`
   - All required environment variable(s) - may be multiple for `custom` auth
   - Help URL for obtaining credentials

3. **Available tools:**
   - List all tool names
   - Identify the primary tool for testing
   - Note required parameters

**Primary tool selection:** Prefer a tool with `readOnlyHint: true` for safe testing.

**Output:** Summary of MCP configuration and requirements.
</step>

### Step 2: Environment Setup

<step>
Check and configure the profile environment:

1. **Locate profiles.yaml:**
   - Windows: `%AppData%\Roaming\mcp-scooter\profiles.yaml`
   - macOS: `~/Library/Application Support/mcp-scooter/profiles.yaml`
   - Linux: `~/.config/mcp-scooter/profiles.yaml`
   - Fallback: workspace root `profiles.yaml`

2. **If file doesn't exist:** Create it with a default profile structure.

3. **Check ALL required environment variables:**
   - Read the current profile's `env` map
   - Identify missing required variables from the registry's `authorization` section

4. **If ANY variables are missing:**
   - List all missing variables with their `display_name` and `help_url`
   - Ask the user for each value
   - Do NOT proceed until all are provided
   - Update the `env` section with the provided values

**Security:** Never log or echo API keys/secrets in output.

**Constraint:** Do not proceed without ALL required credentials.

**Output:** Confirmation that environment is configured.
</step>

### Step 3: Implementation Verification

<step>
Verify the discovery engine supports the MCP's runtime based on `runtime.transport`:

**For `stdio` transport:**
- Check `internal/domain/discovery/stdio.go` exists and implements:
  - Process spawning via `os/exec`
  - JSON-RPC message framing (newline-delimited)
  - Stdin/stdout pipe management
  - Process lifecycle (start, health check, graceful shutdown)
- If missing, implement using `os/exec.Command` pattern.

**For `wasm` transport:**
- Check `internal/domain/discovery/wasm.go` exists and implements:
  - Wazero runtime initialization
  - WASI support for filesystem/network access
  - Module loading and instantiation
- If missing, implement using `wazero` runtime.

**For `sse` transport:**
- Check `internal/domain/discovery/sse.go` exists and implements:
  - HTTP client with SSE support
  - Event stream parsing
  - Reconnection logic
- If missing, implement using standard `net/http` with SSE handling.

**On missing implementation:** Implement the transport before proceeding, or report as blocker.

**Output:** Confirmation that runtime support is available.
</step>

### Step 4: Create Integration Test

<step>
Create a temporary integration test file at `internal/domain/discovery/{name}_integration_test.go`:

```go
//go:build integration

package discovery

import (
    "context"
    "testing"
    "time"
)

func Test{Name}Integration(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    engine, err := NewDiscoveryEngine()
    if err != nil {
        t.Fatalf("Failed to create discovery engine: %v", err)
    }
    defer engine.Close()

    // Step 1: Add the MCP server
    t.Log("Adding {name} MCP server...")
    if err := engine.Add(ctx, "{name}"); err != nil {
        t.Fatalf("Failed to add MCP: %v", err)
    }

    // Step 2: List available tools
    t.Log("Listing tools...")
    tools, err := engine.ListTools(ctx, "{name}")
    if err != nil {
        t.Fatalf("Failed to list tools: %v", err)
    }
    t.Logf("Available tools: %v", tools)

    // Step 3: Call primary tool (prefer readOnlyHint: true)
    t.Log("Calling {primary_tool}...")
    result, err := engine.CallTool(ctx, "{name}", "{primary_tool}", map[string]interface{}{
        // Add test parameters here
    })
    if err != nil {
        t.Fatalf("Failed to call tool: %v", err)
    }
    t.Logf("Result: %+v", result)

    // Step 4: Cleanup - remove MCP
    t.Log("Removing {name} MCP server...")
    if err := engine.Remove(ctx, "{name}"); err != nil {
        t.Errorf("Failed to remove MCP: %v", err)
    }
}
```

**Customize:**
- Replace `{Name}` with PascalCase MCP name (e.g., `BraveSearch`)
- Replace `{name}` with lowercase MCP name (e.g., `brave-search`)
- Replace `{primary_tool}` with the main tool to test (prefer `readOnlyHint: true`)
- Add appropriate test parameters

**Output:** Test file created and ready to run.
</step>

### Step 5: Execute Test

<step>
Run the integration test:

```bash
go test -v -tags=integration -timeout=120s ./internal/domain/discovery/ -run Test{Name}Integration
```

**Capture output:** Save both stdout and stderr for debugging.

**Expected outcomes:**

1. **Success:**
   - MCP server starts without errors
   - Tools are listed correctly
   - Primary tool returns valid response
   - Cleanup completes successfully

2. **On test failure:**
   - Check stderr for MCP server errors
   - Verify environment variables are correctly passed
   - Check network connectivity for remote services
   - Report specific failure point and error message

**Output:** Test results with JSON output from tool calls.
</step>

### Step 6: Cleanup & Report

<step>
After testing is complete:

1. **Always remove temporary test file** (even on failure):
   ```bash
   rm internal/domain/discovery/{name}_integration_test.go
   ```

2. **Generate report using this format:**

```
## MCP Test: {name}

| Step | Status | Details |
|------|--------|---------|
| Pre-flight | ✅/❌ | ... |
| Registry | ✅/❌ | ... |
| Environment | ✅/❌ | ... |
| Implementation | ✅/❌ | ... |
| Execution | ✅/❌ | ... |

### Tools Discovered
- tool_name_1
- tool_name_2

### Test Output
[JSON response or error message]

### Overall: ✅ PASS / ❌ FAIL
```

3. **If failed, provide:**
   - Error message and stack trace
   - Suggested fixes from Error Recovery table
   - Links to relevant documentation

**Output:** Final test report to the user.
</step>

## Output Summary

After completing the workflow, provide:

| Item | Status |
|------|--------|
| Pre-flight checks | ✅/❌ |
| Registry loaded | ✅/❌ |
| Environment configured | ✅/❌ |
| Runtime supported | ✅/❌ |
| Server started | ✅/❌ |
| Tools listed | ✅/❌ (count) |
| Primary tool called | ✅/❌ |
| Cleanup completed | ✅/❌ |

## Example Usage

**Input:** `brave-search`

**Output:**
```
## MCP Test: brave-search

| Step | Status | Details |
|------|--------|---------|
| Pre-flight | ✅ | Schema valid, Go 1.21.0 |
| Registry | ✅ | 2 tools, stdio transport |
| Environment | ✅ | BRAVE_API_KEY configured |
| Implementation | ✅ | stdio.go present |
| Execution | ✅ | All tests passed |

### Tools Discovered
- brave_web_search (readOnlyHint: true)
- brave_local_search (readOnlyHint: true)

### Test Output
Test: brave_web_search({ "query": "test" })
Response time: 1.2s

{
  "results": [
    {
      "title": "Test - Wikipedia",
      "url": "https://en.wikipedia.org/wiki/Test",
      "description": "..."
    }
  ]
}

### Overall: ✅ PASS
```

## Error Recovery

| Error | Recovery Action |
|-------|-----------------|
| Registry file not found | Check spelling, list available MCPs in `appdata/registry/official/` |
| Schema validation failed | Fix registry JSON before proceeding |
| Missing env var | Prompt user, provide `help_url` from registry |
| Transport not implemented | Implement or report as blocker |
| MCP process crash | Check command/args, capture stderr |
| Tool call timeout | Increase timeout, check network |
| Auth failure (401/403) | Verify API key is valid and has required permissions |
| Command not found | Install required package (npm, pip, etc.), check PATH |
| Connection refused | Verify MCP server started, check for port conflicts |
| Tool not found | Verify tool name matches registry definition |
