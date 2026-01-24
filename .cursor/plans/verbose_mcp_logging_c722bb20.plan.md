---
name: Verbose MCP Logging
overview: Add a toggleable "Verbose Logging" setting to MCP Scooter that enables detailed TRACE-level logging at critical points in the MCP request/response flow, helping diagnose issues like the "No response" problem when external clients like Cline interact with the gateway.
todos:
  - id: settings-field
    content: Add VerboseLogging boolean field to Settings struct in settings.go
    status: pending
  - id: logger-helpers
    content: Add SetVerbose(), Trace(), and truncateForLog() functions to logger.go
    status: pending
  - id: gateway-logging
    content: Add TRACE logging points in handleMessage for request/response/SSE flow
    status: pending
  - id: settings-sync
    content: Sync VerboseLogging setting to logger in handleUpdateSettings and on startup
    status: pending
  - id: test-feature
    content: Test with Cline to verify verbose logs capture the full MCP flow
    status: pending
isProject: false
---

# Verbose MCP Gateway Logging

## Problem Context

When debugging MCP client interactions (e.g., Cline in VS Code), the current logging is insufficient to diagnose issues like:

1. **"No response" mystery**: Cline displays "(No response)" but we cannot determine if:

   - The client never sent the request
   - The request was received but parsing failed
   - The response was generated but SSE delivery failed
   - The tool activation check rejected the request

2. **Tool activation flow opacity**: When an agent calls a tool like `brave_web_search` without first calling `scooter_add`, we should see:

   - The exact request that came in
   - The activation check result (`isActive=false`)
   - The error response being sent
   - Whether SSE delivery succeeded

3. **Current logging gaps**: The existing logs show high-level events but miss:

   - Raw request/response bodies
   - SSE channel delivery confirmation
   - Detailed tool resolution steps
   - Request-response correlation

## Solution Design

Add a `VerboseLogging` boolean to Settings that, when enabled, outputs TRACE-level logs at every critical point in the MCP flow.

```
┌─────────────────────────────────────────────────────────────────┐
│                     MCP Request Flow                            │
├─────────────────────────────────────────────────────────────────┤
│  Client Request                                                 │
│       │                                                         │
│       ▼                                                         │
│  [TRACE] Raw body received ◄── NEW                              │
│       │                                                         │
│       ▼                                                         │
│  [TRACE] Parsed: method=X, id=Y ◄── NEW                         │
│       │                                                         │
│       ▼                                                         │
│  [TRACE] Tool lookup: name=Z, server=W ◄── NEW                  │
│       │                                                         │
│       ▼                                                         │
│  [TRACE] Activation check: isActive=bool ◄── NEW                │
│       │                                                         │
│       ▼                                                         │
│  [INFO] Handling 'tools/call' ◄── EXISTING                      │
│       │                                                         │
│       ▼                                                         │
│  [TRACE] Response JSON: {...} ◄── NEW                           │
│       │                                                         │
│       ▼                                                         │
│  [TRACE] SSE send result: success/timeout ◄── NEW               │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Details

### 1. Add VerboseLogging to Settings

**File:** [internal/domain/profile/settings.go](internal/domain/profile/settings.go)

Add new field to the `Settings` struct:

```go
type Settings struct {
    // ... existing fields ...
    VerboseLogging bool `yaml:"verbose_logging" json:"verbose_logging"`
}
```

### 2. Add TRACE Helper to Logger

**File:** [internal/logger/logger.go](internal/logger/logger.go)

Add a global verbose flag and helper function:

```go
var verboseEnabled bool

func SetVerbose(enabled bool) {
    mu.Lock()
    defer mu.Unlock()
    verboseEnabled = enabled
}

func Trace(message string) {
    mu.RLock()
    enabled := verboseEnabled
    mu.RUnlock()
    if enabled {
        AddLog("TRACE", message)
    }
}
```

### 3. Add Verbose Logging Points in MCP Gateway

**File:** [internal/api/server.go](internal/api/server.go)

Add TRACE logs at these critical points in `handleMessage`:

| Location | Log Content |

|----------|-------------|

| After reading body | Raw request bytes (truncated if > 2KB) |

| After JSON parse | Method name, request ID, params summary |

| Tool lookup | Tool name, resolved server, found status |

| Activation check | isActive, isInternal, isAllowed flags |

| Response construction | Response JSON (truncated if > 2KB) |

| SSE send attempt | Session ID, channel status |

| SSE send result | Success, timeout, or session-not-found |

### 4. Propagate Setting to Logger

**File:** [internal/api/server.go](internal/api/server.go)

In `handleUpdateSettings`, sync the verbose flag:

```go
func (s *ControlServer) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...
    s.settings = settings
    logger.SetVerbose(settings.VerboseLogging)  // NEW
    // ... rest of function ...
}
```

Also initialize on server startup.

### 5. Key Logging Additions

**In `handleMessage` (around line 1538):**

```go
// After reading body
logger.Trace(fmt.Sprintf("[MCP] Raw request from profile %s: %s", id, truncateForLog(string(body), 2048)))

// After JSON unmarshal
logger.Trace(fmt.Sprintf("[MCP] Parsed request: method=%s, id=%v", req.Method, req.ID))

// In tools/call handling, after tool lookup
logger.Trace(fmt.Sprintf("[MCP] Tool lookup: name=%s, serverName=%s, found=%v", params.Name, serverName, found))

// After activation check
logger.Trace(fmt.Sprintf("[MCP] Activation check: tool=%s, isActive=%v, isInternal=%v, isAllowed=%v", params.Name, isActive, isInternal, isAllowed))

// Before sending response
logger.Trace(fmt.Sprintf("[MCP] Response for request %v: %s", req.ID, truncateForLog(respJSON, 2048)))

// After SSE send attempt
logger.Trace(fmt.Sprintf("[MCP] SSE delivery to session %s: %s", sessionId, deliveryResult))
```

### 6. Add Truncation Helper

**File:** [internal/logger/logger.go](internal/logger/logger.go)

```go
func truncateForLog(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "... [truncated]"
}
```

## Files to Modify

1. **[internal/domain/profile/settings.go](internal/domain/profile/settings.go)** - Add `VerboseLogging` field
2. **[internal/logger/logger.go](internal/logger/logger.go)** - Add `SetVerbose`, `Trace`, and `truncateForLog`
3. **[internal/api/server.go](internal/api/server.go)** - Add TRACE logging points and sync setting to logger

## Testing the Feature

After implementation, to diagnose the original Cline issue:

1. Enable Verbose Logging in MCP Scooter settings
2. Connect Cline to `http://127.0.0.1:6277/profiles/test-brave/sse`
3. Ask Cline to "search the web"
4. Check logs for:

   - `[TRACE] Raw request` entries showing what Cline actually sent
   - `[TRACE] Activation check `showing why `brave_web_search` was rejected
   - `[TRACE] SSE delivery` confirming the error response was sent

This will reveal whether:

- Cline never sent the `brave_web_search` request (no TRACE log)
- Cline sent it but it was rejected (TRACE shows activation check failed)
- Response was sent but SSE delivery failed (TRACE shows timeout/error)