# Agentic Framework Testing

Run the comprehensive agent-in-the-loop testing framework to validate MCP Scooter's core functionality, multi-step scenarios, and real-world agent reasoning.

## Prerequisites

Before running these tests, ensure:
1. **Scooter is running:** `./dev.ps1` or `make dev`
2. **Environment Variables:**
   - `SCOOTER_URL`: Base URL (default: `http://127.0.0.1:6277`)
   - `OPENROUTER_API_KEY`: Required for Layer 3 evaluations (or `ANTHROPIC_API_KEY` as fallback)
3. **Python Environment:** `pip install -r tests/requirements.txt`

## Workflow Steps

### Step 1: Protocol Health (Layer 1)

<step>
Verify the MCP Gateway is responding correctly to standard protocol messages.

```bash
$env:SCOOTER_URL="http://127.0.0.1:6277"; cd tests; go test ./protocol/... -v
```

**Validation:**
- Handshake returns correct protocol version.
- Builtin tools (`scooter_find`, `scooter_activate`, `scooter_deactivate`, `scooter_list_active`) are listed and callable.
</step>

### Step 2: Multi-Step Scenarios (Layer 2)

<step>
Execute YAML-defined workflows that test tool discovery, activation, and proxied usage.

```bash
$env:SCOOTER_URL="http://127.0.0.1:6277"; cd tests; go test ./scenarios/... -v
```

**Validation:**
- Agent can activate a tool (e.g., `github`) using `scooter_activate`.
- Proxied tools are correctly exposed after activation.
</step>

### Step 3: Agent Reasoning Evaluation (Layer 3)

<step>
Run natural language tasks where the agent must figure out which tools to use without explicit instructions.

```bash
$env:PYTHONIOENCODING="utf-8"; $env:SCOOTER_URL="http://127.0.0.1:6277"; cd tests; py -3 evaluation/run_evaluation.py --mode scenarios --profile work
```

> **Note:** Ensure `OPENROUTER_API_KEY` is set in your environment before running.

**Validation:**
- Agent uses `scooter_find` to discover capabilities.
- Agent uses `scooter_activate` to turn on necessary tools.
- Agent calls the activated tools directly (e.g., `brave_web_search`).
- Agent completes the end-to-end task (e.g., search web, find repository).

**Current Scenarios:**
1. **Natural Web Search** - Find who created MCP and when
2. **Natural Code Search** - Find GitHub's official MCP server repository
3. **Capability Discovery** - Report available search tools
4. **Graceful Failure** - Decline impossible tasks without hallucinating
</step>

### Step 4: Analyze Results

<step>
Review the generated reports in `tests/results/`.

1. **Read latest report:** `ls tests/results/scenario_report_*.md`
2. **Check for failures:** Identify if the agent failed to reason correctly or if a tool returned an error.
3. **Common failure causes:**
   - Tool argument format errors
   - Profile restrictions blocking tool access (check `AllowTools` in profile config)
   - Missing tool schemas or descriptions
4. **Suggest Fixes:** If reasoning failed, suggest prompt improvements or tool description updates.
</step>

## Output Summary

After completing the tests, provide a summary:

| Layer | Status | Key Findings |
|-------|--------|--------------|
| 1: Protocol | ✅/❌ | ... |
| 2: Scenarios | ✅/❌ | ... |
| 3: Eval | ✅/❌ | ... |

**Overall Score:** [e.g., 85% Task Completion]
