# MCP Scooter Agent Testing Framework

This folder contains a comprehensive testing framework for validating MCP Scooter's functionality through protocol compliance, scenario-based tests, and LLM-driven evaluations.

## Prerequisites

- **Go 1.24+**: For protocol and scenario tests.
- **Python 3.12+**: For LLM evaluation.
- **Scooter Running**: Start with `./dev.ps1` (Windows) or `make dev`.
- **OpenRouter API Key**: Required for Layer 3 (LLM Evaluation). Set as `OPENROUTER_API_KEY`.

## Setup

1. **Install Python dependencies**:
   ```bash
   cd tests
   pip install -r requirements.txt
   ```

2. **Start MCP Scooter**:
   The tests require a running instance of Scooter.
   ```powershell
   # Windows (PowerShell)
   ./dev.ps1
   
   # Unix/macOS
   make dev
   ```

## Running Tests

You can run tests sequentially by layer or all at once.

### Layer 1: Protocol Tests (Go)
Tests basic MCP protocol compliance (handshake, list tools, builtin calls).
```powershell
$env:SCOOTER_URL="http://127.0.0.1:6277"
cd tests
go test ./protocol/... -v
```

### Layer 2: Scenario Tests (Go + YAML)
Tests multi-step workflows defined in YAML using a Go runner.
```powershell
$env:SCOOTER_URL="http://127.0.0.1:6277"
cd tests
go test ./scenarios/... -v
```

### Layer 3: LLM Evaluation (Python + OpenRouter)
Tests real agent reasoning and multi-turn tool orchestration using LLMs via OpenRouter.

```powershell
$env:SCOOTER_URL="http://127.0.0.1:6277"
$env:OPENROUTER_API_KEY="your-api-key-here"
cd tests
py -3 evaluation/run_evaluation.py --mode scenarios --profile work
```

**Current Test Scenarios:**
1. **Natural Web Search** - Agent discovers and uses search tools to find information
2. **Natural Code Search** - Agent searches for GitHub repositories
3. **Capability Discovery** - Agent reports available tools for a task
4. **Graceful Failure** - Agent declines impossible tasks without hallucinating

### Run Everything via Makefile
From the project root:
```bash
make test-agent-full
```

## Test Results

Evaluation results are stored in the `tests/results/` directory:
- **Markdown Reports**: Human-readable summaries (`scenario_report_*.md`)
- **JSON Results**: Raw data for analysis (`scenario_results_*.json`)

Example report output:
```
## Overall Score: 4/4 (100%)
```

## Structure

```
tests/
├── protocol/          # Go-based MCP protocol tests
├── scenarios/         # YAML-defined test scenarios + Go runner
│   └── definitions/   # Scenario YAML files
├── evaluation/        # LLM-driven evaluation
│   ├── run_evaluation.py
│   ├── openrouter_client.py
│   └── scenarios.yaml
├── fixtures/          # Mock MCP server for testing
└── results/           # Output directory for reports
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SCOOTER_URL` | Base URL of the Scooter Gateway | `http://127.0.0.1:6277` |
| `SCOOTER_API_KEY` | API key for the gateway (if required) | - |
| `OPENROUTER_API_KEY` | Required for Layer 3 LLM evaluation | - |
| `ANTHROPIC_API_KEY` | Fallback for Layer 3 (if OpenRouter not set) | - |

## Troubleshooting

**Tool argument errors**: Check `EVALUATION_PROMPT` in `evaluation/run_evaluation.py` for correct argument formats.

**Profile restrictions**: If `scooter_add` fails with "not allowed", check `AllowTools` in your profile configuration.

**Connection errors**: Ensure Scooter is running on the expected port (default: 6277).
