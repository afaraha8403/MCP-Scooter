package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"crypto/rand"
	"encoding/hex"
	"os/exec"
	"runtime"
	"github.com/mcp-scooter/scooter/internal/domain/discovery"
	"github.com/mcp-scooter/scooter/internal/domain/integration"
	"github.com/mcp-scooter/scooter/internal/domain/profile"
	"github.com/mcp-scooter/scooter/internal/domain/registry"
	"github.com/mcp-scooter/scooter/internal/logger"
)

// Helper function to extract tool names from tools
func getToolNames(tools []registry.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names
}

// ControlServer handles management requests (CRUD for profiles).
type ControlServer struct {
	mux                *http.ServeMux
	store              *profile.Store
	manager            *ProfileManager
	settings           profile.Settings
	onboardingRequired bool
}

// NewControlServer creates a new management server.
func NewControlServer(store *profile.Store, manager *ProfileManager, settings profile.Settings, onboardingRequired bool) *ControlServer {
	s := &ControlServer{
		mux:                http.NewServeMux(),
		store:              store,
		manager:            manager,
		settings:           settings,
		onboardingRequired: onboardingRequired,
	}
	s.routes()
	return s
}

func (s *ControlServer) routes() {
	s.mux.HandleFunc("GET /api/profiles", s.handleGetProfiles)
	s.mux.HandleFunc("POST /api/profiles", s.handleCreateProfile)
	s.mux.HandleFunc("PUT /api/profiles", s.handleUpdateProfile)
	s.mux.HandleFunc("DELETE /api/profiles", s.handleDeleteProfile)
	s.mux.HandleFunc("POST /api/clients/sync", s.handleInstallIntegration)
	s.mux.HandleFunc("POST /api/onboarding/start-fresh", s.handleOnboardingStartFresh)
	s.mux.HandleFunc("POST /api/onboarding/import", s.handleOnboardingImport)
	s.mux.HandleFunc("POST /api/reset", s.handleReset)
	s.mux.HandleFunc("GET /api/tools", s.handleGetTools)
	s.mux.HandleFunc("POST /api/tools", s.handleRegisterTool)
	s.mux.HandleFunc("POST /api/tools/refresh", s.handleRefreshTools)
	s.mux.HandleFunc("POST /api/tools/verify", s.handleVerifyTool)
	s.mux.HandleFunc("DELETE /api/tools", s.handleDeleteTool)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/clients", s.handleGetClients)
	s.mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	s.mux.HandleFunc("PUT /api/settings", s.handleUpdateSettings)
	s.mux.HandleFunc("POST /api/settings/regenerate-key", s.handleRegenerateKey)
	s.mux.HandleFunc("GET /api/tool-params", s.handleGetToolParams)
	s.mux.HandleFunc("PUT /api/tool-params", s.handleSaveToolParams)
	// Log management
	s.mux.HandleFunc("GET /api/logs", s.handleGetLogs)
	s.mux.HandleFunc("POST /api/logs", s.handlePostLog)
	s.mux.HandleFunc("GET /api/logs/stream", s.handleLogStream)
	s.mux.HandleFunc("DELETE /api/logs", s.handleClearLogs)
	s.mux.HandleFunc("POST /api/logs/reveal", s.handleRevealLogs)
	// Secure credential management
	s.mux.HandleFunc("POST /api/credentials", s.handleSetCredential)
	s.mux.HandleFunc("GET /api/credentials/check", s.handleCheckCredentials)
	s.mux.HandleFunc("DELETE /api/credentials", s.handleDeleteCredential)
	// AI routing credentials
	s.mux.HandleFunc("POST /api/credentials/ai-primary", s.handleSetPrimaryAIKey)
	s.mux.HandleFunc("POST /api/credentials/ai-fallback", s.handleSetFallbackAIKey)
	s.mux.HandleFunc("GET /api/credentials/ai", s.handleCheckAICredentials)
	s.mux.HandleFunc("DELETE /api/credentials/ai-primary", s.handleDeletePrimaryAIKey)
	s.mux.HandleFunc("DELETE /api/credentials/ai-fallback", s.handleDeleteFallbackAIKey)
	s.mux.HandleFunc("GET /api/status", s.handleGetStatus)
}

func (s *ControlServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"timestamp": time.Now().Unix(),
	})
}

func (s *ControlServer) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	profiles := s.manager.GetProfiles()

	type ToolStatus struct {
		Name   string `json:"name"`
		Status string `json:"status"` // "ok", "warning", "error"
	}

	type ProfileStatus struct {
		ID          string       `json:"id"`
		Running     bool         `json:"running"`
		ActiveTools int          `json:"active_tools"`
		ToolStatus  []ToolStatus `json:"tool_status"`
	}

	info := make([]ProfileStatus, len(profiles))
	s.manager.mu.RLock()
	for i, p := range profiles {
		engine, running := s.manager.engines[p.ID]

		toolStatuses := []ToolStatus{}
		activeTools := 0
		if running {
			activeNames := engine.ListActive()
			activeTools = len(activeNames)

			// Map to check if a tool is active
			activeMap := make(map[string]bool)
			for _, name := range activeNames {
				activeMap[name] = true
			}

			// Add allowed tools
			for _, name := range p.AllowTools {
				status := "idle"
				if activeMap[name] {
					status = "ok"
				}
				toolStatuses = append(toolStatuses, ToolStatus{
					Name:   name,
					Status: status,
				})
			}

			// Add active tools that might not be in AllowTools (e.g. builtins)
			for _, name := range activeNames {
				alreadyAdded := false
				for _, added := range p.AllowTools {
					if added == name {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					toolStatuses = append(toolStatuses, ToolStatus{
						Name:   name,
						Status: "ok",
					})
				}
			}
		}

		info[i] = ProfileStatus{
			ID:          p.ID,
			Running:     running,
			ActiveTools: activeTools,
			ToolStatus:  toolStatuses,
		}
		if info[i].ToolStatus == nil {
			info[i].ToolStatus = []ToolStatus{}
		}
	}
	s.manager.mu.RUnlock()

	response := struct {
		GatewayRunning  bool            `json:"gateway_running"`
		ControlPort     int             `json:"control_port"`
		McpPort         int             `json:"mcp_port"`
		ActiveProfileID string          `json:"active_profile_id"`
		Profiles        []ProfileStatus `json:"profiles"`
	}{
		GatewayRunning:  true,
		ControlPort:     s.settings.ControlPort,
		McpPort:         s.settings.McpPort,
		ActiveProfileID: s.settings.LastProfileID,
		Profiles:        info,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *ControlServer) handleRegenerateKey(w http.ResponseWriter, r *http.Request) {
	newKey := profile.GenerateAPIKey()
	s.settings.GatewayAPIKey = newKey

	if s.store != nil {
		if err := s.store.SaveSettings(s.settings); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"gateway_api_key": newKey})
}

func (s *ControlServer) handleGetToolParams(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
		return
	}

	params, err := s.store.LoadToolParams()
	if err != nil {
		// Return empty object if file doesn't exist
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(params)
}

func (s *ControlServer) handleSaveToolParams(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ToolName   string                 `json:"tool_name"`
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if s.store == nil {
		http.Error(w, "store not initialized", http.StatusInternalServerError)
		return
	}

	// Load existing params
	params, _ := s.store.LoadToolParams()
	if params == nil {
		params = make(map[string]map[string]interface{})
	}

	// Update params for this tool
	params[req.ToolName] = req.Parameters

	// Save
	if err := s.store.SaveToolParams(params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
}

func (s *ControlServer) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	logs := logger.GetLogs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": logs,
	})
}

func (s *ControlServer) handlePostLog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Level   string `json:"level"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Level == "" {
		req.Level = "INFO"
	}
	logger.AddLog(req.Level, req.Message)
	w.WriteHeader(http.StatusCreated)
}

func (s *ControlServer) handleLogStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	logChan := logger.Subscribe()
	defer logger.Unsubscribe(logChan)

	// Send initial pulse to confirm connection
	fmt.Fprintf(w, "event: connected\ndata: {\"status\": \"ok\"}\n\n")
	flusher.Flush()

	for {
		select {
		case entry := <-logChan:
			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", string(data))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *ControlServer) handleClearLogs(w http.ResponseWriter, r *http.Request) {
	if err := logger.ClearLogs(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *ControlServer) handleRevealLogs(w http.ResponseWriter, r *http.Request) {
	path := logger.GetLogFilePath()
	dir := filepath.Dir(path)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default: // linux and others
		cmd = exec.Command("xdg-open", dir)
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to open logs folder: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *ControlServer) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.settings)
}

func (s *ControlServer) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings profile.Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.settings = settings
	if s.store != nil {
		if err := s.store.SaveSettings(s.settings); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(s.settings)
}

func (s *ControlServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Global CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *ControlServer) handleGetProfiles(w http.ResponseWriter, r *http.Request) {
	profiles := s.manager.GetProfiles()

	type ProfileInfo struct {
		profile.Profile
		Running bool `json:"running"`
	}

	info := make([]ProfileInfo, len(profiles))
	s.manager.mu.RLock()
	for i, p := range profiles {
		_, running := s.manager.engines[p.ID]
		info[i] = ProfileInfo{
			Profile: p,
			Running: running,
		}
	}
	s.manager.mu.RUnlock()

	configPath := ""
	settingsPath := ""
	if s.store != nil {
		configPath = s.store.GetProfilesPath()
		settingsPath = s.store.GetSettingsPath()
	}

	response := struct {
		Profiles           []ProfileInfo    `json:"profiles"`
		Settings           profile.Settings `json:"settings"`
		OnboardingRequired bool             `json:"onboarding_required"`
		ConfigPath         string           `json:"config_path"`
		SettingsPath       string           `json:"settings_path"`
	}{
		Profiles:           info,
		Settings:           s.settings,
		OnboardingRequired: s.onboardingRequired,
		ConfigPath:         configPath,
		SettingsPath:       settingsPath,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *ControlServer) handleOnboardingStartFresh(w http.ResponseWriter, r *http.Request) {
	defaultProfile := profile.Profile{
		ID:             "work",
		RemoteAuthMode: "none",
	}

	if err := s.manager.AddProfile(defaultProfile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if s.store != nil {
		if err := s.store.SaveProfiles(s.manager.GetProfiles()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	s.onboardingRequired = false

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(defaultProfile)
}

func (s *ControlServer) handleOnboardingImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Profiles []profile.Profile `yaml:"profiles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, p := range req.Profiles {
		if err := s.manager.AddProfile(p); err != nil {
			// Skip duplicates or log them
			continue
		}
	}

	if len(s.manager.GetProfiles()) > 0 {
		s.onboardingRequired = false
	}

	if s.store != nil {
		if err := s.store.SaveProfiles(s.manager.GetProfiles()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *ControlServer) handleReset(w http.ResponseWriter, r *http.Request) {
	s.manager.ClearProfiles()
	s.onboardingRequired = true
	s.settings = profile.DefaultSettings()

	if s.store != nil {
		if err := s.store.Save(s.manager.GetProfiles(), s.settings); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "reset_successful"})
}

func (s *ControlServer) handleGetTools(w http.ResponseWriter, r *http.Request) {
	engine := discovery.NewDiscoveryEngine(context.Background(), s.manager.wasmDir, s.manager.registryDir)

	s.manager.mu.RLock()
	for _, td := range s.manager.customTools {
		engine.Register(td)
	}
	s.manager.mu.RUnlock()

	tools := engine.Find("")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	})
}

func (s *ControlServer) handleRefreshTools(w http.ResponseWriter, r *http.Request) {
	// Refresh tools by reloading the registry for all engines
	s.manager.mu.RLock()
	engines := s.manager.engines
	profileIDs := make([]string, 0, len(engines))
	for id := range engines {
		profileIDs = append(profileIDs, id)
	}
	s.manager.mu.RUnlock()

	logger.AddLog("INFO", fmt.Sprintf("Refreshing tools for %d profile(s): %v", len(profileIDs), profileIDs))

	if len(engines) == 0 {
		logger.AddLog("WARN", "No active engines found - nothing to refresh")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Refreshed 0 engines (no profiles active)",
		})
		return
	}

	refreshedCount := 0
	for profileID, engine := range engines {
		logger.AddLog("INFO", fmt.Sprintf("Refreshing registry for profile '%s'", profileID))
		if err := engine.ReloadRegistry(); err != nil {
			logger.AddLog("ERROR", fmt.Sprintf("Failed to refresh registry for profile '%s': %v", profileID, err))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		refreshedCount++
		logger.AddLog("INFO", fmt.Sprintf("Successfully refreshed registry for profile '%s'", profileID))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully refreshed %d profile engine(s)", refreshedCount),
	})
}

// handleVerifyTool verifies a specific MCP tool by starting its server,
// performing the handshake, fetching actual tools, and updating the registry if needed.
func (s *ControlServer) handleVerifyTool(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var req struct {
		ToolName string `json:"tool_name"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	if req.ToolName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "tool_name is required",
		})
		return
	}

	logger.AddLog("INFO", fmt.Sprintf("[Verify] Starting verification for tool: %s", req.ToolName))

	// Step 1: Find the tool definition in the registry
	logger.AddLog("INFO", fmt.Sprintf("[Verify] Step 1: Looking up tool '%s' in registry...", req.ToolName))
	
	engine := discovery.NewDiscoveryEngine(r.Context(), s.manager.wasmDir, s.manager.registryDir)
	tools := engine.Find("")
	
	var toolDef *discovery.ToolDefinition
	for i := range tools {
		if tools[i].Name == req.ToolName {
			toolDef = &tools[i]
			break
		}
	}

	if toolDef == nil {
		logger.AddLog("ERROR", fmt.Sprintf("[Verify] Tool '%s' not found in registry", req.ToolName))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Tool '%s' not found in registry", req.ToolName),
		})
		return
	}

	logger.AddLog("INFO", fmt.Sprintf("[Verify] Found tool '%s' in registry (source: %s)", req.ToolName, toolDef.Source))

	// Step 2: Check if the tool has a runtime configuration
	if toolDef.Runtime == nil {
		logger.AddLog("ERROR", fmt.Sprintf("[Verify] Tool '%s' has no runtime configuration", req.ToolName))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Tool '%s' has no runtime configuration - cannot verify", req.ToolName),
		})
		return
	}

	logger.AddLog("INFO", fmt.Sprintf("[Verify] Step 2: Runtime config found - transport: %s, command: %s", toolDef.Runtime.Transport, toolDef.Runtime.Command))

	// Step 3: Start the MCP server and perform handshake
	logger.AddLog("INFO", fmt.Sprintf("[Verify] Step 3: Starting MCP server for '%s'...", req.ToolName))
	logger.AddLog("INFO", fmt.Sprintf("[Verify] Command: %s %v", toolDef.Runtime.Command, toolDef.Runtime.Args))

	// Get credentials for this tool
	credManager := engine.GetCredentialManager()
	toolEnv := make(map[string]string)
	
	// Check if we have credentials in the request (from the UI form)
	var credReq struct {
		Credentials map[string]string `json:"credentials"`
	}
	if err := json.Unmarshal(body, &credReq); err == nil && len(credReq.Credentials) > 0 {
		for k, v := range credReq.Credentials {
			toolEnv[k] = v
			logger.AddLog("INFO", fmt.Sprintf("[Verify] Using provided credential: %s", k))
		}
	}

	// Also layer in stored credentials if not provided in request
	if toolDef.Authorization != nil {
		creds, err := credManager.GetCredentialsForTool(req.ToolName, toolDef.Authorization)
		if err == nil {
			for k, v := range creds {
				if _, exists := toolEnv[k]; !exists {
					toolEnv[k] = v
					logger.AddLog("INFO", fmt.Sprintf("[Verify] Injected stored credential: %s", k))
				}
			}
		}
	}

	// Create a temporary stdio worker to verify the tool
	verifyResult, err := discovery.VerifyMCPTool(r.Context(), toolDef, toolEnv)
	if err != nil {
		logger.AddLog("ERROR", fmt.Sprintf("[Verify] Failed to verify tool '%s': %v", req.ToolName, err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Verification failed: %v", err),
		})
		return
	}

	// Step 4: Compare tools from server with registry
	logger.AddLog("INFO", fmt.Sprintf("[Verify] Step 4: Comparing tools from server with registry..."))
	logger.AddLog("INFO", fmt.Sprintf("[Verify] Registry has %d tools, server reported %d tools", len(toolDef.Tools), len(verifyResult.ServerTools)))

	// Check for differences
	registryToolNames := make(map[string]bool)
	for _, t := range toolDef.Tools {
		registryToolNames[t.Name] = true
	}

	serverToolNames := make(map[string]bool)
	for _, t := range verifyResult.ServerTools {
		serverToolNames[t.Name] = true
	}

	// Find tools in server but not in registry
	var newTools []string
	for name := range serverToolNames {
		if !registryToolNames[name] {
			newTools = append(newTools, name)
		}
	}

	// Find tools in registry but not in server
	var missingTools []string
	for name := range registryToolNames {
		if !serverToolNames[name] {
			missingTools = append(missingTools, name)
		}
	}

	toolsChanged := len(newTools) > 0 || len(missingTools) > 0

	if len(newTools) > 0 {
		logger.AddLog("INFO", fmt.Sprintf("[Verify] New tools from server: %v", newTools))
	}
	if len(missingTools) > 0 {
		logger.AddLog("WARN", fmt.Sprintf("[Verify] Tools in registry but not in server: %v", missingTools))
	}

	// Step 5: Update registry
	logger.AddLog("INFO", fmt.Sprintf("[Verify] Step 5: Updating registry JSON with %d tools and verification timestamp...", len(verifyResult.ServerTools)))
	
	err = s.updateRegistryTools(req.ToolName, verifyResult.ServerTools)
	var registryUpdated bool
	if err != nil {
		logger.AddLog("ERROR", fmt.Sprintf("[Verify] Failed to update registry: %v", err))
	} else {
		registryUpdated = true
		logger.AddLog("INFO", fmt.Sprintf("[Verify] Registry updated successfully"))
		
		// Step 5b: Reload the in-memory registry for all active profile engines
		// This ensures the updated tool names are immediately available for invocation
		logger.AddLog("INFO", "[Verify] Step 5b: Reloading in-memory registry for all active engines...")
		s.manager.mu.RLock()
		for profileID, profileEngine := range s.manager.engines {
			logger.AddLog("INFO", fmt.Sprintf("[Verify] Reloading registry for profile '%s'", profileID))
			if reloadErr := profileEngine.ReloadRegistry(); reloadErr != nil {
				logger.AddLog("WARN", fmt.Sprintf("[Verify] Failed to reload registry for profile '%s': %v", profileID, reloadErr))
			}
		}
		s.manager.mu.RUnlock()
		logger.AddLog("INFO", "[Verify] In-memory registry reload complete")
	}

	// Build response
	response := map[string]interface{}{
		"success":          true,
		"tool_name":        req.ToolName,
		"server_info":      verifyResult.ServerInfo,
		"registry_tools":   len(toolDef.Tools),
		"server_tools":     len(verifyResult.ServerTools),
		"new_tools":        newTools,
		"missing_tools":    missingTools,
		"tools_changed":    toolsChanged,
		"registry_updated": registryUpdated,
	}

	// Include the actual tool definitions from server
	serverToolDetails := make([]map[string]interface{}, 0, len(verifyResult.ServerTools))
	for _, t := range verifyResult.ServerTools {
		serverToolDetails = append(serverToolDetails, map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
		})
	}
	response["server_tool_details"] = serverToolDetails

	logger.AddLog("INFO", fmt.Sprintf("[Verify] Verification complete for '%s'", req.ToolName))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateRegistryTools updates the tools array in the registry JSON file for a specific tool.
func (s *ControlServer) updateRegistryTools(toolName string, newTools []registry.Tool) error {
	if s.manager.registryDir == "" {
		return fmt.Errorf("registry directory not configured")
	}

	// Check both official and custom directories
	subdirs := []string{"official", "custom"}
	for _, subdir := range subdirs {
		filePath := filepath.Join(s.manager.registryDir, subdir, fmt.Sprintf("%s.json", toolName))
		
		// Check if file exists
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Try next directory
		}

		logger.AddLog("INFO", fmt.Sprintf("[Verify] Found registry file: %s", filePath))

		// Parse existing entry
		var entry registry.MCPEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return fmt.Errorf("failed to parse registry file: %w", err)
		}

		// Update tools and verification timestamp
		entry.Tools = newTools
		if entry.Metadata == nil {
			entry.Metadata = &registry.Metadata{}
		}
		now := time.Now().Format(time.RFC3339)
		entry.Metadata.VerifiedAt = now

		// Write back with pretty formatting
		updatedData, err := json.MarshalIndent(entry, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize updated entry: %w", err)
		}

		if err := os.WriteFile(filePath, updatedData, 0644); err != nil {
			return fmt.Errorf("failed to write registry file: %w", err)
		}

		logger.AddLog("INFO", fmt.Sprintf("[Verify] Updated registry file: %s", filePath))
		return nil
	}

	return fmt.Errorf("registry file not found for tool: %s", toolName)
}

type ClientDefinition struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Icon               string   `json:"icon"`
	IconDark           string   `json:"icon_dark,omitempty"`
	Description        string   `json:"description"`
	ManualInstructions string   `json:"manual_instructions"`
	Version            string   `json:"version,omitempty"`
	Developer          string   `json:"developer,omitempty"`
	Category           string   `json:"category,omitempty"`
	Tags               []string `json:"tags,omitempty"`
	About              string   `json:"about,omitempty"`
	Homepage           string   `json:"homepage,omitempty"`
	Repository         string   `json:"repository,omitempty"`
	Documentation      string   `json:"documentation,omitempty"`
	DownloadURL        string   `json:"download_url,omitempty"`
	Platforms          []string `json:"platforms,omitempty"`
	Installed          bool     `json:"installed"`
	MCPSupport         *struct {
		Transports []string `json:"transports"`
		Features   []string `json:"features"`
		Status     string   `json:"status"`
	} `json:"mcp_support,omitempty"`
	Features []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"features,omitempty"`
	Metadata *struct {
		License string `json:"license,omitempty"`
		Pricing string `json:"pricing,omitempty"`
	} `json:"metadata,omitempty"`
}

func (s *ControlServer) handleGetClients(w http.ResponseWriter, r *http.Request) {
	clients := []ClientDefinition{}
	clientsDir := s.manager.clientsDir

	if clientsDir != "" {
		files, err := os.ReadDir(clientsDir)
		if err == nil {
			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					data, err := os.ReadFile(filepath.Join(clientsDir, file.Name()))
					if err != nil {
						continue
					}
					var cd ClientDefinition
					if err := json.Unmarshal(data, &cd); err == nil {
						// Simple installation detection
						cd.Installed = s.isClientInstalled(cd.ID)
						clients = append(clients, cd)
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"clients": clients,
	})
}

func (s *ControlServer) isClientInstalled(id string) bool {
	home, _ := os.UserHomeDir()
	appData := os.Getenv("APPDATA")
	localAppData := os.Getenv("LOCALAPPDATA")

	switch id {
	case "vscode":
		// Check for VS Code executable or config
		paths := []string{
			filepath.Join(home, ".vscode"),
			filepath.Join(appData, "Code"),
			filepath.Join(localAppData, "Programs", "Microsoft VS Code"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	case "cursor":
		paths := []string{
			filepath.Join(home, ".cursor"),
			filepath.Join(appData, "Cursor"),
			filepath.Join(localAppData, "Programs", "cursor"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	case "claude-desktop":
		paths := []string{
			filepath.Join(appData, "Claude"),
			filepath.Join(localAppData, "Programs", "Claude"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	case "zed":
		paths := []string{
			filepath.Join(home, ".zed"),
			filepath.Join(appData, "Zed"),
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	case "claude-code":
		// Usually installed via npm globally
		return true // Assume true for CLI if we can't easily check
	case "gemini-cli":
		// Usually installed via npm globally
		return true // Assume true for CLI if we can't easily check
	}
	return false
}

func (s *ControlServer) handleRegisterTool(w http.ResponseWriter, r *http.Request) {
	var td discovery.ToolDefinition
	if err := json.NewDecoder(r.Body).Decode(&td); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Persist to custom registry folder
	if s.manager.registryDir != "" {
		customDir := filepath.Join(s.manager.registryDir, "custom")
		os.MkdirAll(customDir, 0755)

		filePath := filepath.Join(customDir, fmt.Sprintf("%s.json", td.Name))
		data, err := json.MarshalIndent(td, "", "  ")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to serialize tool: %v", err), http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(filePath, data, 0644); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save tool file: %v", err), http.StatusInternalServerError)
			return
		}
	}

	s.manager.mu.Lock()
	// Check for duplicates in memory
	found := false
	for i, existing := range s.manager.customTools {
		if existing.Name == td.Name {
			s.manager.customTools[i] = td
			found = true
			break
		}
	}
	if !found {
		s.manager.customTools = append(s.manager.customTools, td)
	}
	s.manager.mu.Unlock()

	logger.AddLog("INFO", fmt.Sprintf("Registered and persisted tool: %s", td.Name))

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(td)
}

// handleSetCredential securely stores a credential in the system keychain.
func (s *ControlServer) handleSetCredential(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ToolName string `json:"tool_name"`
		EnvVar   string `json:"env_var"`
		Value    string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ToolName == "" || req.EnvVar == "" {
		http.Error(w, "tool_name and env_var are required", http.StatusBadRequest)
		return
	}

	// Get credential manager from an active engine
	engine := discovery.NewDiscoveryEngine(r.Context(), s.manager.wasmDir, s.manager.registryDir)
	credManager := engine.GetCredentialManager()

	if err := credManager.SetCredential(req.ToolName, req.EnvVar, req.Value); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store credential: %v", err), http.StatusInternalServerError)
		return
	}

	logger.AddLog("INFO", fmt.Sprintf("Stored credential %s for tool %s", req.EnvVar, req.ToolName))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stored"})
}

// handleCheckCredentials checks if required credentials are present for a tool.
func (s *ControlServer) handleCheckCredentials(w http.ResponseWriter, r *http.Request) {
	toolName := r.URL.Query().Get("tool_name")
	if toolName == "" {
		http.Error(w, "tool_name is required", http.StatusBadRequest)
		return
	}

	// Get tool definition to check authorization requirements
	engine := discovery.NewDiscoveryEngine(r.Context(), s.manager.wasmDir, s.manager.registryDir)
	tools := engine.Find("")

	var toolDef *discovery.ToolDefinition
	for i := range tools {
		if tools[i].Name == toolName {
			toolDef = &tools[i]
			break
		}
	}

	if toolDef == nil {
		http.Error(w, "Tool not found", http.StatusNotFound)
		return
	}

	credManager := engine.GetCredentialManager()
	hasAll, missing := credManager.HasRequiredCredentials(toolName, toolDef.Authorization)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"has_required": hasAll,
		"missing":      missing,
	})
}

func (s *ControlServer) handleDeleteTool(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Remove from custom registry folder
	if s.manager.registryDir != "" {
		filePath := filepath.Join(s.manager.registryDir, "custom", fmt.Sprintf("%s.json", name))
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("Failed to delete tool file: %v", err), http.StatusInternalServerError)
			return
		}
	}

	s.manager.mu.Lock()
	// Remove from memory
	for i, existing := range s.manager.customTools {
		if existing.Name == name {
			s.manager.customTools = append(s.manager.customTools[:i], s.manager.customTools[i+1:]...)
			break
		}
	}
	s.manager.mu.Unlock()

	logger.AddLog("INFO", fmt.Sprintf("Deleted tool: %s", name))

	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteCredential removes a credential from the keychain.
func (s *ControlServer) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	toolName := r.URL.Query().Get("tool_name")
	envVar := r.URL.Query().Get("env_var")

	if toolName == "" || envVar == "" {
		http.Error(w, "tool_name and env_var are required", http.StatusBadRequest)
		return
	}

	engine := discovery.NewDiscoveryEngine(r.Context(), s.manager.wasmDir, s.manager.registryDir)
	credManager := engine.GetCredentialManager()

	if err := credManager.DeleteCredential(toolName, envVar); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete credential: %v", err), http.StatusInternalServerError)
		return
	}

	logger.AddLog("INFO", fmt.Sprintf("Deleted credential %s for tool %s", envVar, toolName))
	w.WriteHeader(http.StatusNoContent)
}

// handleSetPrimaryAIKey stores the primary AI routing API key.
func (s *ControlServer) handleSetPrimaryAIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Value == "" {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}

	// Get credential manager from an active engine
	engine := discovery.NewDiscoveryEngine(r.Context(), s.manager.wasmDir, s.manager.registryDir)
	credManager := engine.GetCredentialManager()

	if err := credManager.SetCredential("mcp-scooter:ai_primary", "MCP_SCOOTER_PRIMARY_AI_KEY", req.Value); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store primary AI key: %v", err), http.StatusInternalServerError)
		return
	}

	logger.AddLog("INFO", "Stored primary AI routing credential")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stored"})
}

// handleSetFallbackAIKey stores the fallback AI routing API key.
func (s *ControlServer) handleSetFallbackAIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Value == "" {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}

	// Get credential manager from an active engine
	engine := discovery.NewDiscoveryEngine(r.Context(), s.manager.wasmDir, s.manager.registryDir)
	credManager := engine.GetCredentialManager()

	if err := credManager.SetCredential("mcp-scooter:ai_fallback", "MCP_SCOOTER_FALLBACK_AI_KEY", req.Value); err != nil {
		http.Error(w, fmt.Sprintf("Failed to store fallback AI key: %v", err), http.StatusInternalServerError)
		return
	}

	logger.AddLog("INFO", "Stored fallback AI routing credential")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stored"})
}

// handleCheckAICredentials checks if AI routing credentials are configured.
func (s *ControlServer) handleCheckAICredentials(w http.ResponseWriter, r *http.Request) {
	engine := discovery.NewDiscoveryEngine(r.Context(), s.manager.wasmDir, s.manager.registryDir)
	credManager := engine.GetCredentialManager()

	primaryKey, err1 := credManager.GetCredential("mcp-scooter:ai_primary", "MCP_SCOOTER_PRIMARY_AI_KEY")
	fallbackKey, err2 := credManager.GetCredential("mcp-scooter:ai_fallback", "MCP_SCOOTER_FALLBACK_AI_KEY")

	hasPrimary := err1 == nil && primaryKey != ""
	hasFallback := err2 == nil && fallbackKey != ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"primary_configured":  hasPrimary && s.settings.PrimaryAIProvider != "",
		"fallback_configured": hasFallback && s.settings.FallbackAIProvider != "",
	})
}

// handleDeletePrimaryAIKey removes the primary AI routing API key.
func (s *ControlServer) handleDeletePrimaryAIKey(w http.ResponseWriter, r *http.Request) {
	engine := discovery.NewDiscoveryEngine(r.Context(), s.manager.wasmDir, s.manager.registryDir)
	credManager := engine.GetCredentialManager()

	if err := credManager.DeleteCredential("mcp-scooter:ai_primary", "MCP_SCOOTER_PRIMARY_AI_KEY"); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete primary AI key: %v", err), http.StatusInternalServerError)
		return
	}

	logger.AddLog("INFO", "Deleted primary AI routing credential")
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteFallbackAIKey removes the fallback AI routing API key.
func (s *ControlServer) handleDeleteFallbackAIKey(w http.ResponseWriter, r *http.Request) {
	engine := discovery.NewDiscoveryEngine(r.Context(), s.manager.wasmDir, s.manager.registryDir)
	credManager := engine.GetCredentialManager()

	if err := credManager.DeleteCredential("mcp-scooter:ai_fallback", "MCP_SCOOTER_FALLBACK_AI_KEY"); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete fallback AI key: %v", err), http.StatusInternalServerError)
		return
	}

	logger.AddLog("INFO", "Deleted fallback AI routing credential")
	w.WriteHeader(http.StatusNoContent)
}

func (s *ControlServer) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	var p profile.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := p.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.manager.AddProfile(p); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	s.onboardingRequired = false

	if s.store != nil {
		if err := s.store.SaveProfiles(s.manager.GetProfiles()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (s *ControlServer) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	var p profile.Profile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.manager.UpdateProfile(p); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if s.store != nil {
		if err := s.store.SaveProfiles(s.manager.GetProfiles()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(p)
}

func (s *ControlServer) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := s.manager.RemoveProfile(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Update onboardingRequired if no profiles left
	if len(s.manager.GetProfiles()) == 0 {
		s.onboardingRequired = true
	}

	if s.store != nil {
		if err := s.store.SaveProfiles(s.manager.GetProfiles()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *ControlServer) handleInstallIntegration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Target  string `json:"target"` // "cursor", "claude-desktop", "claude-code"
		Profile string `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use configured McpPort from settings
	mcpPort := s.settings.McpPort
	apiKey := s.settings.GatewayAPIKey

	var err error
	switch req.Target {
	case "cursor":
		c := &integration.CursorIntegration{}
		err = c.Configure(mcpPort, req.Profile, apiKey)
	case "claude-desktop":
		c := &integration.ClaudeIntegration{}
		err = c.Configure(mcpPort, req.Profile, apiKey)
	case "claude-code":
		c := &integration.ClaudeIntegration{}
		err = c.ConfigureCode(mcpPort, req.Profile, apiKey)
	case "vscode":
		v := &integration.VSCodeIntegration{}
		err = v.Configure(mcpPort, req.Profile, apiKey)
	case "antigravity", "gemini-cli":
		g := &integration.GeminiIntegration{}
		err = g.Configure(mcpPort, req.Profile, apiKey)
	case "codex":
		c := &integration.CodexIntegration{}
		err = c.Configure(mcpPort, req.Profile, apiKey)
	case "zed":
		z := &integration.ZedIntegration{}
		err = z.Configure(mcpPort, req.Profile, apiKey)
	default:
		err = fmt.Errorf("unknown integration target")
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// McpGateway handles MCP traffic for all profiles on a single port.
type McpGateway struct {
	manager      *ProfileManager
	mux          *http.ServeMux
	settings     profile.Settings
	sseClients   map[string][]chan string // profileID -> list of SSE notification channels
	sseSessions  map[string]chan string   // sessionId -> specific session channel
	sseClientsMu sync.RWMutex
}

func NewMcpGateway(manager *ProfileManager, settings profile.Settings) *McpGateway {
	g := &McpGateway{
		manager:     manager,
		mux:         http.NewServeMux(),
		settings:    settings,
		sseClients:  make(map[string][]chan string),
		sseSessions: make(map[string]chan string),
	}
	g.routes()

	// Set up cleanup callbacks for all engines to notify SSE clients when tools are auto-unloaded
	for _, p := range manager.GetProfiles() {
		if engine, ok := manager.GetEngine(p.ID); ok {
			profileID := p.ID // Capture for closure
			engine.SetCleanupCallback(func(serverName string) {
				logger.AddLog("INFO", fmt.Sprintf("Tool '%s' auto-unloaded, notifying SSE clients", serverName))
				g.NotifyToolsChanged(profileID)
			})
		}
	}

	return g
}

// NotifyToolsChanged sends a tools/list_changed notification to all SSE clients for a profile.
// This is called after scooter_add, scooter_remove, or auto-cleanup.
func (g *McpGateway) NotifyToolsChanged(profileID string) {
	g.sseClientsMu.RLock()
	clients := g.sseClients[profileID]
	g.sseClientsMu.RUnlock()

	notification := `{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`

	for _, ch := range clients {
		select {
		case ch <- notification:
			// Sent successfully
		default:
			// Channel full, skip (client will catch up on next poll)
		}
	}

	if len(clients) > 0 {
		logger.AddLog("INFO", fmt.Sprintf("Sent tools/list_changed to %d SSE clients for profile '%s'", len(clients), profileID))
	}
}

func (g *McpGateway) routes() {
	// Standard MCP routes with profile ID in the path
	g.mux.HandleFunc("GET /profiles/{id}/sse", g.handleSSE)
	g.mux.HandleFunc("POST /profiles/{id}/sse", g.handleMessage) // Streamable HTTP: POST to same endpoint
	g.mux.HandleFunc("POST /profiles/{id}/message", g.handleMessage)

	// Default routes for "work" profile (compatibility)
	g.mux.HandleFunc("GET /sse", func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("id", "work")
		g.handleSSE(w, r)
	})
	g.mux.HandleFunc("POST /sse", func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("id", "work")
		g.handleMessage(w, r) // Streamable HTTP: POST to same endpoint
	})
	g.mux.HandleFunc("POST /message", func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("id", "work")
		g.handleMessage(w, r)
	})
}

func (g *McpGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Global CORS headers for MCP clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Scooter-API-Key, X-Scooter-Internal")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Internal requests from Scooter Desktop bypass authentication
	isInternal := r.Header.Get("X-Scooter-Internal") == "true"

	// Check authentication if a key is configured (skip for internal requests)
	if g.settings.GatewayAPIKey != "" && !isInternal {
		authHeader := r.Header.Get("Authorization")
		apiKey := r.Header.Get("X-Scooter-API-Key")

		if authHeader != "" {
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			} else {
				apiKey = authHeader
			}
		}

		if apiKey != g.settings.GatewayAPIKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	g.mux.ServeHTTP(w, r)
}

func (g *McpGateway) handleSSE(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_, ok := g.manager.GetEngine(id)
	if !ok {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	logger.AddLog("INFO", fmt.Sprintf("SSE connection opened for profile: %s", id))

	flusher, ok := w.(http.Flusher)
	if !ok {
		logger.AddLog("ERROR", "Streaming unsupported for SSE")
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Register this SSE client for notifications and responses
	sessionId := generateSessionID()
	notifyChan := make(chan string, 10)
	g.sseClientsMu.Lock()
	g.sseSessions[sessionId] = notifyChan
	g.sseClients[id] = append(g.sseClients[id], notifyChan)
	g.sseClientsMu.Unlock()

	// Cleanup on disconnect
	defer func() {
		g.sseClientsMu.Lock()
		delete(g.sseSessions, sessionId)
		channels := g.sseClients[id]
		for i, ch := range channels {
			if ch == notifyChan {
				g.sseClients[id] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		g.sseClientsMu.Unlock()
		close(notifyChan)
		logger.AddLog("INFO", fmt.Sprintf("SSE connection closed for profile: %s (session: %s)", id, sessionId))
	}()

	// Send endpoint event for client to know where to POST messages
	// Standard MCP SSE transport requires the client to POST to this endpoint
	fmt.Fprintf(w, "event: endpoint\ndata: http://127.0.0.1:%d/profiles/%s/sse?sessionId=%s\n\n", g.settings.McpPort, id, sessionId)
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second) // Increased heartbeat interval
	defer ticker.Stop()

	for {
		select {
		case notification := <-notifyChan:
			// Send MCP message (notification or response)
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", notification)
			flusher.Flush()
		case <-ticker.C:
			// Keep-alive pulse (non-standard but helpful)
			fmt.Fprintf(w, "event: pulse\ndata: {\"profile\": \"%s\", \"session\": \"%s\", \"status\": \"ok\", \"timestamp\": \"%s\"}\n\n", id, sessionId, time.Now().Format(time.RFC3339))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (g *McpGateway) handleMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	engine, ok := g.manager.GetEngine(id)
	if !ok {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	var req JSONRPCRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.AddLog("ERROR", fmt.Sprintf("Failed to read MCP request body: %v", err))
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		logger.AddLog("ERROR", fmt.Sprintf("Failed to decode MCP request: %v. Body: %s", err, string(body)))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(NewJSONRPCErrorResponse(nil, ParseError, "Parse error"))
		return
	}

	// Handle notifications (no ID)
	if req.ID == nil {
		logger.AddLog("INFO", fmt.Sprintf("Received MCP Notification from profile %s: %s", id, req.Method))
		if req.Method == "notifications/initialized" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// Other notifications are ignored for now
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var resp JSONRPCResponse
	logger.AddLog("INFO", fmt.Sprintf("MCP Request [%v] from profile %s: %s", req.ID, id, req.Method))

	switch req.Method {
	case "initialize":
		logger.AddLog("INFO", "Handling 'initialize' request")
		resp = NewJSONRPCResponse(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]string{
				"name":    "mcp-scooter",
				"version": "0.1.0",
			},
		})

	case "tools/list", "list_tools":
		logger.AddLog("INFO", "Handling 'tools/list' request")
		p, ok := g.manager.GetProfile(id)
		if ok {
			engine.SetDisabledTools(p.DisabledSystemTools)
		}

		var mcpTools []registry.Tool

		// 1. Always include builtin (primordial) tools - these are the "meta-layer"
		//    that allows agents to discover and activate other tools dynamically.
		for _, td := range discovery.PrimordialTools() {
			if !engine.IsToolDisabled(td.Name) {
				mcpTools = append(mcpTools, td.Tools...)
			}
		}

		// 2. Include tools ONLY from active servers (not all allowed tools).
		//    This is the "Docker MCP Toolkit" pattern - tools must be explicitly
		//    activated via scooter_add before they appear in the tool list.
		activeServers := engine.ListActive()
		logger.AddLog("DEBUG", fmt.Sprintf("Active servers: %v", activeServers))
		for _, serverName := range activeServers {
			serverTools := engine.GetActiveToolsForServer(serverName)
			logger.AddLog("DEBUG", fmt.Sprintf("Server '%s' provides %d tools: %v", serverName, len(serverTools), getToolNames(serverTools)))
			mcpTools = append(mcpTools, serverTools...)
		}

		resp = NewJSONRPCResponse(req.ID, map[string]interface{}{
			"tools": mcpTools,
		})
		logger.AddLog("INFO", fmt.Sprintf("Returned %d tools (builtins + %d active servers)", len(mcpTools), len(engine.ListActive())))

	case "resources/list":
		logger.AddLog("INFO", "Handling 'resources/list' request")
		resp = NewJSONRPCResponse(req.ID, map[string]interface{}{
			"resources": []interface{}{},
		})

	case "prompts/list":
		logger.AddLog("INFO", "Handling 'prompts/list' request")
		resp = NewJSONRPCResponse(req.ID, map[string]interface{}{
			"prompts": []interface{}{},
		})

	case "tools/call", "call_tool":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			msg := fmt.Sprintf("Invalid params for call_tool: %v", err)
			logger.AddLog("ERROR", msg)
			resp = NewJSONRPCErrorResponse(req.ID, InvalidParams, msg)
			break
		}

		logger.AddLog("INFO", fmt.Sprintf("Handling 'tools/call' for '%s' (Profile: %s)", params.Name, id))

		// Sync profile settings with engine
		p, profileOk := g.manager.GetProfile(id)
		if profileOk {
			engine.SetEnv(p.Env)
			engine.SetDisabledTools(p.DisabledSystemTools)
			engine.SetSettings(g.settings)
		}

		// Check if this is a builtin tool (always allowed)
		isBuiltin := false
		for _, primordial := range discovery.PrimordialTools() {
			if primordial.Name == params.Name {
				isBuiltin = true
				break
			}
		}

		// Special permission check for scooter_add - the tool being added must be in AllowTools
		if params.Name == "scooter_add" {
			toolToAdd, _ := params.Arguments["tool_name"].(string)
			if toolToAdd != "" && profileOk {
				isInternal := r.Header.Get("X-Scooter-Internal") == "true"
				isAllowed := isInternal

				if !isAllowed {
					for _, allowed := range p.AllowTools {
						if allowed == toolToAdd {
							isAllowed = true
							break
						}
					}
				}

				if !isAllowed {
					msg := fmt.Sprintf("Tool '%s' is not allowed for this profile. Add it to AllowTools in your profile configuration before using scooter_add.", toolToAdd)
					logger.AddLog("ERROR", msg)
					resp = NewJSONRPCErrorResponse(req.ID, InvalidParams, msg)
					break
				}
			}
		}

		if !isBuiltin {
			// For non-builtin tools, check if the server is active
			serverName, found := engine.GetServerForTool(params.Name)
			if !found {
				// Tool not found in registry at all
				msg := fmt.Sprintf("Tool '%s' not found. Use scooter_find to discover available tools.", params.Name)
				logger.AddLog("ERROR", msg)
				resp = NewJSONRPCErrorResponse(req.ID, MethodNotFound, msg)
				break
			}

			// Check if server is active
			isActive := false
			for _, active := range engine.ListActive() {
				if active == serverName {
					isActive = true
					break
				}
			}

			if !isActive {
				// Tool exists but server is not active - NO auto-loading (Docker MCP Toolkit pattern)
				// Check if it's an internal request (tool testing) - internal requests bypass activation requirement
				internalHeaderValue := r.Header.Get("X-Scooter-Internal")
				isInternal := internalHeaderValue == "true"
				logger.AddLog("DEBUG", fmt.Sprintf("Tool '%s': isActive=%v, internalHeaderValue='%s', isInternal=%v", params.Name, isActive, internalHeaderValue, isInternal))
				if isInternal {
					// For internal requests (tool testing), temporarily activate the tool
					logger.AddLog("DEBUG", fmt.Sprintf("Tool '%s': Internal request detected, temporarily activating server '%s'", params.Name, serverName))
					err := engine.Add(serverName)
					if err != nil {
						logger.AddLog("ERROR", fmt.Sprintf("Failed to temporarily activate server '%s' for internal request: %v", serverName, err))
						resp = NewJSONRPCErrorResponse(req.ID, MethodNotFound, fmt.Sprintf("Tool error: Failed to activate server '%s': %v", serverName, err))
						break
					}
					logger.AddLog("DEBUG", fmt.Sprintf("Tool '%s': Server '%s' temporarily activated for testing", params.Name, serverName))
				} else {
					// For external requests, check if tool is allowed for this profile
					isAllowed := false
					if profileOk {
						for _, allowed := range p.AllowTools {
							if allowed == serverName {
								isAllowed = true
								break
							}
						}
					}

					if isAllowed {
						msg := fmt.Sprintf("Tool '%s' is not active. Use scooter_add('%s') to enable it first.", params.Name, serverName)
						logger.AddLog("ERROR", msg)
						resp = NewJSONRPCErrorResponse(req.ID, MethodNotFound, msg)
					} else {
						msg := fmt.Sprintf("Tool '%s' is not allowed for this profile. Add '%s' to AllowTools in your profile configuration.", params.Name, serverName)
						logger.AddLog("ERROR", msg)
						resp = NewJSONRPCErrorResponse(req.ID, MethodNotFound, msg)
					}
					break
				}
			}
		}

		// Call unified tool executor
		startTime := time.Now()
		result, err := engine.CallTool(params.Name, params.Arguments)
		duration := time.Since(startTime)

		if err != nil {
			msg := fmt.Sprintf("Tool execution error for '%s': %v", params.Name, err)
			logger.AddLog("ERROR", msg)
			resp = NewJSONRPCErrorResponse(req.ID, MethodNotFound, fmt.Sprintf("Tool error: %v", err))
		} else {
			logger.AddLog("INFO", fmt.Sprintf("Tool '%s' executed successfully in %v", params.Name, duration))
			// If scooter_add or scooter_remove succeeded, notify SSE clients to refresh tools
			if params.Name == "scooter_add" || params.Name == "scooter_remove" {
				g.NotifyToolsChanged(id)
			}

			// If result is already a map with "content", use it directly
			if resMap, ok := result.(map[string]interface{}); ok {
				if _, hasContent := resMap["content"]; hasContent {
					resp = NewJSONRPCResponse(req.ID, resMap)
				} else {
					resp = NewJSONRPCResponse(req.ID, map[string]interface{}{
						"content": []map[string]interface{}{
							{"type": "text", "text": fmt.Sprintf("%v", result)},
						},
					})
				}
			} else {
				resp = NewJSONRPCResponse(req.ID, map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": fmt.Sprintf("%v", result)},
					},
				})
			}
		}

	default:
		resp = NewJSONRPCErrorResponse(req.ID, MethodNotFound, "Method not found")
	}

	// For standard MCP SSE transport, the response SHOULD be sent via the SSE stream,
	// and the POST request should return 202 Accepted or 200 OK with no body.
	sessionId := r.URL.Query().Get("sessionId")
	if sessionId != "" {
		g.sseClientsMu.RLock()
		ch, ok := g.sseSessions[sessionId]
		g.sseClientsMu.RUnlock()

		if ok {
			respData, _ := json.Marshal(resp)
			select {
			case ch <- string(respData):
				logger.AddLog("INFO", fmt.Sprintf("Sent response to SSE session %s", sessionId))
				w.WriteHeader(http.StatusAccepted)
				return
			case <-time.After(2 * time.Second):
				logger.AddLog("ERROR", fmt.Sprintf("Timeout sending response to SSE session %s. Falling back to HTTP body.", sessionId))
				// Fallback to sending in body if channel is blocked
			}
		} else {
			logger.AddLog("WARNING", fmt.Sprintf("Session %s not found for MCP message. Falling back to HTTP body.", sessionId))
		}
	}

	// Fallback/Legacy: send response in the HTTP body (Streamable HTTP style)
	logger.AddLog("INFO", fmt.Sprintf("Sending MCP response in HTTP body (Profile: %s)", id))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ProfileManager manages discovery engines for active profiles.
type ProfileManager struct {
	mu          sync.RWMutex
	profiles    []profile.Profile
	engines     map[string]*discovery.DiscoveryEngine
	wasmDir     string
	registryDir string
	clientsDir  string
	customTools []discovery.ToolDefinition
}

func NewProfileManager(initial []profile.Profile, wasmDir string, registryDir string, clientsDir string) *ProfileManager {
	pm := &ProfileManager{
		profiles:    initial,
		engines:     make(map[string]*discovery.DiscoveryEngine),
		wasmDir:     wasmDir,
		registryDir: registryDir,
		clientsDir:  clientsDir,
		customTools: []discovery.ToolDefinition{},
	}
	for _, p := range initial {
		pm.engines[p.ID] = discovery.NewDiscoveryEngine(context.Background(), wasmDir, registryDir)
	}
	return pm
}

func (pm *ProfileManager) GetProfiles() []profile.Profile {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.profiles
}

func (pm *ProfileManager) GetProfile(id string) (profile.Profile, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	for _, p := range pm.profiles {
		if p.ID == id {
			return p, true
		}
	}
	return profile.Profile{}, false
}

func (pm *ProfileManager) GetEngine(id string) (*discovery.DiscoveryEngine, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	engine, ok := pm.engines[id]
	return engine, ok
}

func (pm *ProfileManager) ClearProfiles() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.profiles = []profile.Profile{}
	pm.engines = make(map[string]*discovery.DiscoveryEngine)
}

func (pm *ProfileManager) AddProfile(p profile.Profile) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, existing := range pm.profiles {
		if existing.ID == p.ID {
			return fmt.Errorf("profile already exists")
		}
	}

	pm.profiles = append(pm.profiles, p)
	pm.engines[p.ID] = discovery.NewDiscoveryEngine(context.Background(), pm.wasmDir, pm.registryDir)
	return nil
}

func (pm *ProfileManager) UpdateProfile(p profile.Profile) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i, existing := range pm.profiles {
		if existing.ID == p.ID {
			pm.profiles[i] = p
			return nil
		}
	}
	return fmt.Errorf("profile not found")
}

func (pm *ProfileManager) RemoveProfile(id string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i, p := range pm.profiles {
		if p.ID == id {
			delete(pm.engines, id)
			pm.profiles = append(pm.profiles[:i], pm.profiles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("profile not found")
}
