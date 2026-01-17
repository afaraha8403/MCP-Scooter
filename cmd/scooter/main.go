package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mcp-scooter/scooter/internal/api"
	"github.com/mcp-scooter/scooter/internal/domain/profile"
)

func main() {
	fmt.Println("MCP Scooter - Initializing...")

	// Setup profile store
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "mcp-scooter")
	os.MkdirAll(appDir, 0755)

	wasmDir := filepath.Join(appDir, "wasm")
	os.MkdirAll(wasmDir, 0755)

	registryDir := filepath.Join(appDir, "registry")
	os.MkdirAll(registryDir, 0755)

	clientsDir := filepath.Join(appDir, "clients")
	os.MkdirAll(clientsDir, 0755)

	// If registry is empty, copy from local appdata/registry if it exists
	// This is for development convenience and initial setup
	files, _ := os.ReadDir(registryDir)
	if len(files) == 0 {
		localRegistry := "appdata/registry"
		if localFiles, err := os.ReadDir(localRegistry); err == nil {
			for _, f := range localFiles {
				if !f.IsDir() {
					input, _ := os.ReadFile(filepath.Join(localRegistry, f.Name()))
					os.WriteFile(filepath.Join(registryDir, f.Name()), input, 0644)
				}
			}
		}
	}

	// Copy clients from appdata/clients to the user's clients dir if empty
	clientFiles, _ := os.ReadDir(clientsDir)
	if len(clientFiles) == 0 {
		localClients := "appdata/clients"
		if localFiles, err := os.ReadDir(localClients); err == nil {
			for _, f := range localFiles {
				if !f.IsDir() {
					input, _ := os.ReadFile(filepath.Join(localClients, f.Name()))
					os.WriteFile(filepath.Join(clientsDir, f.Name()), input, 0644)
				}
			}
		}
	}

	store := profile.NewStore(filepath.Join(appDir, "profiles.yaml"))
	profiles, settings, err := store.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	onboardingRequired := len(profiles) == 0

	// Initialize Profile Manager
	manager := api.NewProfileManager(profiles, wasmDir, registryDir, clientsDir)

	// Initialize Control Server (Management API)
	controlServer := api.NewControlServer(store, manager, settings, onboardingRequired)

	// Initialize MCP Gateway (Traffic Proxy)
	mcpGateway := api.NewMcpGateway(manager)

	fmt.Printf("Starting MCP Gateway on :%d...\n", settings.McpPort)
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", settings.McpPort), mcpGateway); err != nil {
			fmt.Printf("MCP Gateway failed: %v\n", err)
		}
	}()

	fmt.Printf("Starting control server on :%d...\n", settings.ControlPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", settings.ControlPort), controlServer); err != nil {
		fmt.Printf("Control server failed: %v\n", err)
		os.Exit(1)
	}
}
