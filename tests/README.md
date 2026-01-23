# MCP Scooter Agent Testing Framework

This folder contains a comprehensive testing framework for validating MCP Scooter's functionality through protocol compliance, scenario-based tests, and LLM-driven evaluations.

## Prerequisites

- **Go 1.24+**: For protocol and scenario tests.
- **Python 3.10+**: For LLM evaluation.
- **Scooter Binary**: Run `make build` in the root directory.
- **Anthropic API Key**: Required for Layer 3 (LLM Evaluation). Set as `ANTHROPIC_API_KEY`.

## Setup

1. **Install Python dependencies**:
   ```bash
   cd tests
   pip install -r requirements.txt
   ```

2. **Start MCP Scooter**:
   The tests require a running instance of Scooter.
   ```bash
   # In a separate terminal
   make dev
   ```

## Running Tests

You can run tests sequentially by layer or all at once.

### Layer 1: Protocol Tests (Go)
Tests basic MCP protocol compliance (handshake, list tools, builtin calls).
```bash
# Set URL to your running Scooter instance (default port is often 6277 or 3001)
$env:SCOOTER_URL="http://127.0.0.1:6277"
cd tests
go test ./protocol/... -v
```

### Layer 2: Scenario Tests (Go + YAML)
Tests multi-step workflows defined in YAML using a Go runner.
```bash
$env:SCOOTER_URL="http://127.0.0.1:6277"
cd tests
go test ./scenarios/... -v
```

### Layer 3: LLM Evaluation (Python + Claude)
Tests real agent reasoning and multi-turn tool orchestration.

#### Option A: Natural Language Scenarios (Recommended)
Tests if the agent can discover and use tools based on pure intent.
```bash
$env:SCOOTER_URL="http://127.0.0.1:6277"
cd tests
python evaluation/run_evaluation.py --mode scenarios --profile work
```

#### Option B: Fixed QA Pairs
Tests specific tool outputs with hardcoded answers.
```bash
python evaluation/run_evaluation.py --mode qa --profile work
```

### Run Everything via Makefile
From the project root:
```bash
make test-agent-full
```

## Test Results

Evaluation results are stored in the `tests/results/` directory:
- **Markdown Reports**: Human-readable summaries of the evaluation (`scenario_report_*.md`).
- **JSON Results**: Raw data for programmatic analysis (`scenario_results_*.json`).

## Structure

- `protocol/`: Go-based MCP HTTP/SSE client and protocol tests.
- `scenarios/`: YAML-defined test scenarios and Go execution engine.
- `evaluation/`: LLM-driven evaluation using Claude.
- `fixtures/`: Mock MCP server and test registry entries.
- `results/`: Output directory for evaluation reports.

## Environment Variables

- `SCOOTER_URL`: Base URL of the Scooter Gateway (default: `http://127.0.0.1:6277`)
- `SCOOTER_API_KEY`: API key for the gateway (if required).
- `ANTHROPIC_API_KEY`: Required for Layer 3.
