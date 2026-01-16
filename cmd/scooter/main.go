package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mcp-scout/scooter/internal/api"
	"github.com/mcp-scout/scooter/internal/domain/profile"
)

func main() {
	fmt.Println("MCP Scooter - Initializing...")

	// Setup profile store
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "scooter")
	os.MkdirAll(appDir, 0755)

	wasmDir := filepath.Join(appDir, "wasm")
	os.MkdirAll(wasmDir, 0755)

	store := profile.NewStore(filepath.Join(appDir, "profiles.yaml"))
	profiles, err := store.Load()
	if err != nil {
		fmt.Printf("Failed to load profiles: %v\n", err)
		os.Exit(1)
	}

	onboardingRequired := len(profiles) == 0

	// Initialize Profile Manager
	manager := api.NewProfileManager(profiles, wasmDir)

	// Initialize Control Server (Management API)
	// We'll run the control server on a fixed port, e.g., 6200
	controlServer := api.NewControlServer(store, manager, onboardingRequired)

	fmt.Println("Starting control server on :6200...")
	if err := http.ListenAndServe(":6200", controlServer); err != nil {
		fmt.Printf("Control server failed: %v\n", err)
		os.Exit(1)
	}
}
