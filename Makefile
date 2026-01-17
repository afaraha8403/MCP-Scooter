# MCP Scout Makefile
# ====================

.PHONY: all build test validate clean dev

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

# Help
help:
	@echo "MCP Scout Build Commands"
	@echo "========================"
	@echo ""
	@echo "  make              - Validate registry and build"
	@echo "  make build        - Build the scooter binary"
	@echo "  make test         - Run all tests"
	@echo "  make validate     - Validate registry JSON files"
	@echo "  make validate-strict - Validate with warnings as errors"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make dev          - Run in development mode"
	@echo "  make ci           - Run full CI checks"
	@echo ""
