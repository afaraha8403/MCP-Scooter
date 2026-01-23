# Agentic Framework Testing

Run the comprehensive agent-in-the-loop testing framework to validate MCP Scooter's core functionality, multi-step scenarios, and real-world agent reasoning.

## Prerequisites

Before running these tests, ensure:
1. **Scooter is running:** `make dev` or `go run ./cmd/scooter`
2. **Environment Variables:**
   - `SCOOTER_URL`: Base URL (default: `http://127.0.0.1:6277`)
   - `ANTHROPIC_API_KEY`: Required for Layer 3 evaluations.
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
- Builtin tools (`scooter_*`) are listed and callable.
</step>

### Step 2: Multi-Step Scenarios (Layer 2)

<step>
Execute YAML-defined workflows that test tool discovery, activation, and proxied usage.

```bash
$env:SCOOTER_URL="http://127.0.0.1:6277"; cd tests; go test ./scenarios/... -v
```

**Validation:**
- Agent can add a tool (e.g., `github`).
- Proxied tools are correctly exposed after activation.
- Tool removal cleans up the environment.
</step>

### Step 3: Agent Reasoning Evaluation (Layer 3)

<step>
Run natural language tasks where the agent must figure out which tools to use without explicit instructions.

```bash
$env:SCOOTER_URL="http://127.0.0.1:6277"; cd tests; python evaluation/run_evaluation.py --mode scenarios --profile work
```

**Validation:**
- Agent uses `scooter_find` to discover capabilities.
- Agent uses `scooter_add` to activate necessary tools.
- Agent completes the end-to-end task (e.g., search web, summarize).
</step>

### Step 4: Analyze Results

<step>
Review the generated reports in `tests/results/`.

1. **Read latest report:** `ls tests/results/scenario_report_*.md`
2. **Check for failures:** Identify if the agent failed to reason correctly or if a tool returned an error.
3. **Suggest Fixes:** If reasoning failed, suggest prompt improvements or tool description updates.
</step>

## Output Summary

After completing the tests, provide a summary:

| Layer | Status | Key Findings |
|-------|--------|--------------|
| 1: Protocol | ✅/❌ | ... |
| 2: Scenarios | ✅/❌ | ... |
| 3: Eval | ✅/❌ | ... |

**Overall Score:** [e.g., 85% Task Completion]
