# MCP Scooter Makefile
# ====================

.PHONY: all build test validate clean dev \
        test-unit test-registry test-discovery test-profile test-api test-integration \
        test-coverage test-protocol test-auth test-meta-mcp test-e2e \
        test-fast test-all ci-full pre-commit test-run \
        test-agent-protocol test-agent-scenarios test-agent-eval test-agent test-agent-full

# Default target
all: validate build

# Build the main scooter binary
build:
	go build -o scooter.exe ./cmd/scooter

# Build the validation tool
build-validator:
	go build -o validate-registry.exe ./cmd/validate-registry

# Run all tests
test:
	go test ./...

# Level 2: Unit Tests (by domain)
test-unit:
	go test ./internal/... -v

test-registry:
	go test ./internal/domain/registry/... -v

test-discovery:
	go test ./internal/domain/discovery/... -v

test-profile:
	go test ./internal/domain/profile/... -v

test-api:
	go test ./internal/api/... -v

test-integration:
	go test ./internal/domain/integration/... -v

# Level 2: Unit Tests with coverage
test-coverage:
	go test ./internal/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Level 3: Protocol Compliance (placeholder - needs implementation)
test-protocol:
	go test ./internal/domain/protocol/... -v -tags=protocol

# Level 4: Identity & Auth (placeholder - needs implementation)
test-auth:
	go test ./internal/domain/integration/... -v -run "Keychain|OAuth"

# Level 5: Meta-MCP Lifecycle
test-meta-mcp:
	go test ./internal/domain/discovery/... -v -run "Engine"

# Level 6: End-to-End (placeholder - needs implementation)
test-e2e:
	go test ./... -v -tags=e2e -timeout=120s

# Combined Test Targets
test-fast: test-unit
	@echo "Fast unit tests passed!"

test-all: validate test-unit
	@echo "All tests passed!"

ci-full: fmt lint validate-strict test-coverage build
	@echo "Full CI checks passed!"

pre-commit: fmt lint validate test
	@echo "Pre-commit checks passed!"

# Run specific test by name pattern (Usage: make test-run TEST=TestName)
test-run:
	go test ./... -v -run $(TEST)

# Validate registry JSON files against schema
validate: build-validator
	./validate-registry.exe appdata/registry

# Validate with strict mode (warnings are errors)
validate-strict: build-validator
	./validate-registry.exe -strict appdata/registry

# Validate and output as JSON
validate-json: build-validator
	./validate-registry.exe -json appdata/registry

# Clean build artifacts
clean:
	rm -f scooter.exe validate-registry.exe
	rm -rf desktop/dist

# Development mode - run with hot reload
dev:
	go run ./cmd/scooter

# Format code
fmt:
	go fmt ./...
	cd desktop && npm run format 2>/dev/null || true

# Lint code
lint:
	go vet ./...

# Install dependencies
deps:
	go mod download
	cd desktop && npm install

# Full CI check
ci: fmt lint validate test build
	@echo "CI checks passed!"

# Release a stable version
release:
	@read -p "Enter version (e.g., 1.0.0): " VERSION; \
	git tag -a v$$VERSION -m "Release v$$VERSION"; \
	git push origin v$$VERSION
	@echo "GitHub Action will now build and release v$$VERSION"

# Release a beta version
release-beta:
	@read -p "Enter beta version (e.g., 1.0.0-beta.1): " VERSION; \
	git tag -a v$$VERSION -m "Beta release v$$VERSION"; \
	git push origin v$$VERSION
	@echo "GitHub Action will now build and release v$$VERSION"

# Agent Testing
test-agent-protocol:
	cd tests && go test ./protocol/... -v

test-agent-scenarios:
	cd tests && go test ./scenarios/... -v

test-agent-eval:
	cd tests && python evaluation/run_evaluation.py

test-agent: test-agent-protocol test-agent-scenarios
	@echo "Agent tests passed!"

test-agent-full: test-agent test-agent-eval
	@echo "Full agent test suite passed!"

# Help
help:
	@echo "MCP Scooter Build & Test Commands"
	@echo "==============================="
	@echo ""
	@echo "Build & Run:"
	@echo "  make              - Validate registry and build"
	@echo "  make build        - Build the scooter binary"
	@echo "  make dev          - Run in development mode"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Install dependencies"
	@echo ""
	@echo "Testing (Levels 1-2):"
	@echo "  make test         - Run all tests"
	@echo "  make test-unit    - Run all unit tests (verbose)"
	@echo "  make test-registry - Test registry validation logic"
	@echo "  make test-discovery - Test discovery engine"
	@echo "  make test-profile - Test profile management"
	@echo "  make test-api     - Test API/SSE server"
	@echo "  make test-integration - Test client integrations"
	@echo "  make test-coverage - Generate HTML coverage report"
	@echo ""
	@echo "Testing (Levels 3-6):"
	@echo "  make test-protocol - Protocol compliance tests (placeholder)"
	@echo "  make test-auth    - Keychain/OAuth tests (placeholder)"
	@echo "  make test-meta-mcp - Meta-MCP lifecycle tests"
	@echo "  make test-e2e     - End-to-end tests (placeholder)"
	@echo ""
	@echo "Agent Testing (New Framework):"
	@echo "  make test-agent-protocol  - Run Layer 1 Protocol tests"
	@echo "  make test-agent-scenarios - Run Layer 2 Scenario tests"
	@echo "  make test-agent-eval      - Run Layer 3 LLM Evaluation"
	@echo "  make test-agent           - Run Layer 1 & 2"
	@echo "  make test-agent-full      - Run all agent test layers"
	@echo ""
	@echo "Validation:"
	@echo "  make validate     - Validate registry JSON files"
	@echo "  make validate-strict - Validate with warnings as errors"
	@echo ""
	@echo "CI & Git:"
	@echo "  make ci           - Run standard CI checks"
	@echo "  make ci-full      - Run full CI checks with coverage"
	@echo "  make pre-commit   - Quick check before committing"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Lint code"
	@echo ""
	@echo "Release:"
	@echo "  make release      - Tag and push a stable release"
	@echo "  make release-beta - Tag and push a beta release"
	@echo ""
	@echo "Helper:"
	@echo "  make test-run TEST=name - Run specific test by name"
	@echo ""
