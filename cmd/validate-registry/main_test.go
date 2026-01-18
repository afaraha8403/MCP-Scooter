package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	// Test with a non-existent path
	exitCode := run([]string{"non-existent-path"}, false, false, true)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for non-existent path, got %d", exitCode)
	}

	// Create a temporary directory with a valid and an invalid JSON
	tmpDir, err := os.MkdirTemp("", "validate-registry-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	validJSON := `{
		"$schema": "../../appdata/schemas/mcp-registry.schema.json",
		"name": "test-mcp",
		"version": "1.0.0",
		"title": "Test MCP",
		"description": "A test MCP",
		"category": "utility",
		"source": "local",
		"authorization": {
			"type": "none"
		},
		"tools": [
			{
				"name": "test_tool",
				"description": "A test tool",
				"inputSchema": {
					"type": "object",
					"properties": {}
				}
			}
		],
		"package": {
			"type": "npm",
			"name": "test-mcp"
		},
		"runtime": {
			"transport": "stdio",
			"command": "node",
			"args": ["index.js"]
		}
	}`

	invalidJSON := `{
		"name": "invalid-mcp"
	}`

	validPath := filepath.Join(tmpDir, "valid.json")
	if err := os.WriteFile(validPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("Failed to write valid JSON: %v", err)
	}

	invalidPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	// Test with valid JSON
	exitCode = run([]string{validPath}, false, false, true)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for valid JSON, got %d", exitCode)
	}

	// Test with invalid JSON
	exitCode = run([]string{invalidPath}, false, false, true)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for invalid JSON, got %d", exitCode)
	}

	// Test with directory
	exitCode = run([]string{tmpDir}, false, false, true)
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for directory with invalid JSON, got %d", exitCode)
	}
}
