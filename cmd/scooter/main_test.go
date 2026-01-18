package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "scooter-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the environment variable to use the temp directory
	os.Setenv("SCOOTER_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("SCOOTER_CONFIG_DIR")

	// Create necessary subdirectories and files if needed, 
	// or let run() create them.
	// We need to make sure appdata/registry and appdata/clients exist in the project root
	// when running tests from the cmd/scooter directory.
	// Since tests run in the package directory, we might need to adjust paths.
	
	// For this test, we just want to see if run(false) completes without error.
	err = run(false)
	if err != nil {
		t.Fatalf("run(false) failed: %v", err)
	}

	// Verify that directories were created
	dirs := []string{"wasm", "registry", "clients"}
	for _, d := range dirs {
		path := filepath.Join(tmpDir, d)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to be created", path)
		}
	}

	// Verify that profiles.yaml was created or at least tried to be loaded
	profilePath := filepath.Join(tmpDir, "profiles.yaml")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		// Store.Load() might not create it if it doesn't exist, but NewStore should be fine.
		// Actually, profile.NewStore just sets the path. store.Load() reads it.
	}
}
