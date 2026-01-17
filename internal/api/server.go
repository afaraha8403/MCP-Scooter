package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/discovery"
	"github.com/mcp-scooter/scooter/internal/domain/integration"
	"github.com/mcp-scooter/scooter/internal/domain/profile"
)

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
	s.mux.HandleFunc("GET /api/clients", s.handleGetClients)
	s.mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	s.mux.HandleFunc("PUT /api/settings", s.handleUpdateSettings)
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
		if err := s.store.Save(s.manager.GetProfiles(), s.settings); err != nil {
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

	response := struct {
		Profiles           []ProfileInfo    `json:"profiles"`
		Settings           profile.Settings `json:"settings"`
		OnboardingRequired bool             `json:"onboarding_required"`
	}{
		Profiles:           info,
		Settings:           s.settings,
		OnboardingRequired: s.onboardingRequired,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *ControlServer) handleOnboardingStartFresh(w http.ResponseWriter, r *http.Request) {
	defaultProfile := profile.Profile{
		ID:       "work",
		AuthMode: "none",
	}

	if err := s.manager.AddProfile(defaultProfile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if s.store != nil {
		if err := s.store.Save(s.manager.GetProfiles(), s.settings); err != nil {
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
		if err := s.store.Save(s.manager.GetProfiles(), s.settings); err != nil {
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

	if s.store != nil {
		if err := s.store.Save([]profile.Profile{}, s.settings); err != nil {
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

type ClientDefinition struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Icon               string `json:"icon"`
	Description        string `json:"description"`
	ManualInstructions string `json:"manual_instructions"`
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

func (s *ControlServer) handleRegisterTool(w http.ResponseWriter, r *http.Request) {
	var td discovery.ToolDefinition
	if err := json.NewDecoder(r.Body).Decode(&td); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.manager.mu.Lock()
	// Check for duplicates
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

	addLog(fmt.Sprintf("Registered tool: %s", td.Name))

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(td)
}

// Simple global log helper for the server
func addLog(msg string) {
	fmt.Printf("[API] %s\n", msg)
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
		if err := s.store.Save(s.manager.GetProfiles(), s.settings); err != nil {
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
		if err := s.store.Save(s.manager.GetProfiles(), s.settings); err != nil {
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
		if err := s.store.Save(s.manager.GetProfiles(), s.settings); err != nil {
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

	var err error
	switch req.Target {
	case "cursor":
		c := &integration.CursorIntegration{}
		err = c.Configure(mcpPort, req.Profile)
	case "claude-desktop":
		c := &integration.ClaudeIntegration{}
		err = c.Configure(mcpPort, req.Profile)
	case "claude-code":
		c := &integration.ClaudeIntegration{}
		err = c.ConfigureCode(mcpPort, req.Profile)
	case "vscode":
		v := &integration.VSCodeIntegration{}
		err = v.Configure(mcpPort, req.Profile)
	case "antigravity", "gemini-cli":
		g := &integration.GeminiIntegration{}
		err = g.Configure(mcpPort, req.Profile)
	case "codex":
		c := &integration.CodexIntegration{}
		err = c.Configure(mcpPort, req.Profile)
	case "zed":
		z := &integration.ZedIntegration{}
		err = z.Configure(mcpPort, req.Profile)
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
	manager *ProfileManager
	mux     *http.ServeMux
}

func NewMcpGateway(manager *ProfileManager) *McpGateway {
	g := &McpGateway{
		manager: manager,
		mux:     http.NewServeMux(),
	}
	g.routes()
	return g
}

func (g *McpGateway) routes() {
	// Standard MCP routes with profile ID in the path
	g.mux.HandleFunc("GET /profiles/{id}/sse", g.handleSSE)
	g.mux.HandleFunc("POST /profiles/{id}/message", g.handleMessage)
	
	// Default routes for "work" profile (compatibility)
	g.mux.HandleFunc("GET /sse", func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("id", "work")
		g.handleSSE(w, r)
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
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	g.mux.ServeHTTP(w, r)
}

func (g *McpGateway) handleSSE(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	engine, ok := g.manager.GetEngine(id)
	if !ok {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Notify about active tools on connection
	active := engine.ListActive()
	activeData, _ := json.Marshal(active)
	fmt.Fprintf(w, "event: tools\ndata: %s\n\n", string(activeData))
	flusher.Flush()

	for {
		select {
		case <-ticker.C:
			fmt.Fprintf(w, "event: pulse\ndata: {\"profile\": \"%s\", \"status\": \"ok\", \"timestamp\": \"%s\"}\n\n", id, time.Now().Format(time.RFC3339))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (g *McpGateway) handleMessage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	engine, ok := g.manager.GetEngine(id)
	if !ok {
		http.Error(w, "Profile not found", http.StatusNotFound)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(NewJSONRPCErrorResponse(nil, ParseError, "Parse error"))
		return
	}

	var resp JSONRPCResponse
	switch req.Method {
	case "initialize":
		resp = NewJSONRPCResponse(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"serverInfo": map[string]string{
				"name":    "mcp-scooter",
				"version": "0.1.0",
			},
		})

	case "list_tools":
		tools := engine.Find("")
		resp = NewJSONRPCResponse(req.ID, map[string]interface{}{
			"tools": tools,
		})

	case "call_tool":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp = NewJSONRPCErrorResponse(req.ID, InvalidParams, fmt.Sprintf("Invalid params for call_tool: %v", err))
			break
		}

		// Call unified tool executor
		result, err := engine.CallTool(params.Name, params.Arguments)
		if err != nil {
			resp = NewJSONRPCErrorResponse(req.ID, MethodNotFound, fmt.Sprintf("Tool error: %v", err))
		} else {
			resp = NewJSONRPCResponse(req.ID, map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("%v", result)},
				},
			})
		}

	default:
		resp = NewJSONRPCErrorResponse(req.ID, MethodNotFound, "Method not found")
	}

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
