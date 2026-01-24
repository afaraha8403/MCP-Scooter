package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mcp-scooter/scooter/internal/api"
	"github.com/mcp-scooter/scooter/internal/domain/profile"
	"github.com/mcp-scooter/scooter/internal/logger"
)

func main() {
	if err := run(true); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(serve bool) error {
	fmt.Println("MCP Scooter - Initializing...")

	// Setup profile store
	appDir := os.Getenv("SCOOTER_CONFIG_DIR")
	if appDir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = "."
		}
		appDir = filepath.Join(configDir, "mcp-scooter")
	}

	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to create app dir: %w", err)
	}

	// Initialize Logger
	if err := logger.Init(appDir); err != nil {
		fmt.Printf("Warning: failed to initialize persistent logging: %v\n", err)
	}
	defer logger.Close()

	wasmDir := filepath.Join(appDir, "wasm")
	os.MkdirAll(wasmDir, 0755)

	registryDir := filepath.Join(appDir, "registry")
	os.MkdirAll(filepath.Join(registryDir, "official"), 0755)
	os.MkdirAll(filepath.Join(registryDir, "custom"), 0755)

	clientsDir := filepath.Join(appDir, "clients")
	os.MkdirAll(clientsDir, 0755)

	// Copy official registry files from appdata if they are different or missing
	officialRegistry := "appdata/registry/official"
	if localFiles, err := os.ReadDir(officialRegistry); err == nil {
		for _, f := range localFiles {
			if !f.IsDir() {
				sourcePath := filepath.Join(officialRegistry, f.Name())
				targetPath := filepath.Join(registryDir, "official", f.Name())
				
				sourceData, _ := os.ReadFile(sourcePath)
				targetData, err := os.ReadFile(targetPath)
				
				// Only copy if missing. We don't overwrite because verification metadata 
				// is stored in the target file and would be lost.
				if err != nil && os.IsNotExist(err) {
					fmt.Printf("Installing official tool definition: %s\n", f.Name())
					os.WriteFile(targetPath, sourceData, 0644)
				} else if string(sourceData) != string(targetData) {
					// If they are different, we might want to update, but we must be careful 
					// not to overwrite verification data. For now, we'll just log it.
					// logger.AddLog("DEBUG", fmt.Sprintf("Official tool %s differs from project source", f.Name()))
				}
			}
		}
	}

	localClients := "appdata/clients"
	if localFiles, err := os.ReadDir(localClients); err == nil {
		for _, f := range localFiles {
			if !f.IsDir() {
				sourcePath := filepath.Join(localClients, f.Name())
				targetPath := filepath.Join(clientsDir, f.Name())
				
				sourceData, _ := os.ReadFile(sourcePath)
				targetData, _ := os.ReadFile(targetPath)
				
				if string(sourceData) != string(targetData) {
					fmt.Printf("Updating client definition: %s\n", f.Name())
					os.WriteFile(targetPath, sourceData, 0644)
				}
			}
		}
	}

	store := profile.NewStore(
		filepath.Join(appDir, "profiles.yaml"),
		filepath.Join(appDir, "settings.yaml"),
	)
	profiles, settings, err := store.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize Logger Verbosity from settings
	logger.SetVerbose(settings.VerboseLogging)

	onboardingRequired := len(profiles) == 0

	// Initialize Profile Manager
	manager := api.NewProfileManager(profiles, wasmDir, registryDir, clientsDir)

	logger.AddLog("INFO", "=== MCP Scooter Backend Starting ===")
	logger.AddLog("INFO", fmt.Sprintf("App Directory: %s", appDir))
	logger.AddLog("INFO", fmt.Sprintf("McpPort: %d, ControlPort: %d", settings.McpPort, settings.ControlPort))

	// Initialize Control Server (Management API)
	controlServer := api.NewControlServer(store, manager, settings, onboardingRequired)

	// Initialize MCP Gateway (Traffic Proxy)
	mcpGateway := api.NewMcpGateway(manager, settings)

	if !serve {
		return nil
	}

	fmt.Printf("Starting MCP Gateway on :%d...\n", settings.McpPort)
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", settings.McpPort), mcpGateway); err != nil {
			fmt.Printf("MCP Gateway failed: %v\n", err)
		}
	}()

	fmt.Printf("Starting control server on :%d...\n", settings.ControlPort)
	server := &http.Server{Addr: fmt.Sprintf(":%d", settings.ControlPort), Handler: controlServer}
	
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("control server failed: %v\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	fmt.Println("\nShutting down gracefully...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server shutdown failed: %v\n", err)
	}

	return nil
}
