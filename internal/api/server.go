package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/mcp-scout/scooter/internal/domain/discovery"
	"github.com/mcp-scout/scooter/internal/domain/integration"
	"github.com/mcp-scout/scooter/internal/domain/profile"
)

// ControlServer handles management requests (CRUD for profiles).
type ControlServer struct {
	mux                *http.ServeMux
	store              *profile.Store
	manager            *ProfileManager
	onboardingRequired bool
}

// NewControlServer creates a new management server.
func NewControlServer(store *profile.Store, manager *ProfileManager, onboardingRequired bool) *ControlServer {
	s := &ControlServer{
		mux:                http.NewServeMux(),
		store:              store,
		manager:            manager,
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
	s.mux.HandleFunc("POST /api/integrations/install", s.handleInstallIntegration)
	s.mux.HandleFunc("POST /api/onboarding/start-fresh", s.handleOnboardingStartFresh)
	s.mux.HandleFunc("POST /api/onboarding/import", s.handleOnboardingImport)
}

func (s *ControlServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		_, running := s.manager.servers[p.ID]
		info[i] = ProfileInfo{
			Profile: p,
			Running: running,
		}
	}
	s.manager.mu.RUnlock()

	response := struct {
		Profiles           []ProfileInfo `json:"profiles"`
		OnboardingRequired bool          `json:"onboarding_required"`
	}{
		Profiles:           info,
		OnboardingRequired: s.onboardingRequired,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

func (s *ControlServer) handleOnboardingStartFresh(w http.ResponseWriter, r *http.Request) {
	defaultProfile := profile.Profile{
		ID:       "work",
		Port:     6277,
		AuthMode: "none",
	}

	if err := s.manager.AddProfile(defaultProfile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if s.store != nil {
		if err := s.store.Save(s.manager.GetProfiles()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	s.onboardingRequired = false

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

	if s.store != nil {
		if err := s.store.Save(s.manager.GetProfiles()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	s.onboardingRequired = false

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
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

	if s.store != nil {
		if err := s.store.Save(s.manager.GetProfiles()); err != nil {
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
		if err := s.store.Save(s.manager.GetProfiles()); err != nil {
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

	if s.store != nil {
		if err := s.store.Save(s.manager.GetProfiles()); err != nil {
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

	// Find port for the profile
	port := 6277
	for _, p := range s.manager.GetProfiles() {
		if p.ID == req.Profile {
			port = p.Port
			break
		}
	}

	var err error
	switch req.Target {
	case "cursor":
		c := &integration.CursorIntegration{}
		err = c.Configure(port)
	case "claude-desktop":
		c := &integration.ClaudeIntegration{}
		err = c.Configure(port)
	case "claude-code":
		c := &integration.ClaudeIntegration{}
		err = c.ConfigureCode(port)
	case "vscode":
		v := &integration.VSCodeIntegration{}
		err = v.Configure(port)
	case "antigravity", "gemini-cli":
		g := &integration.GeminiIntegration{}
		err = g.Configure(port)
	case "codex":
		c := &integration.CodexIntegration{}
		err = c.Configure(port)
	case "zed":
		z := &integration.ZedIntegration{}
		err = z.Configure(port)
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

// ProfileServer handles MCP traffic for a specific profile.
type ProfileServer struct {
	profile   profile.Profile
	mux       *http.ServeMux
	server    *http.Server
	discovery *discovery.DiscoveryEngine
	wasmDir   string
}

func NewProfileServer(p profile.Profile, wasmDir string) *ProfileServer {
	ps := &ProfileServer{
		profile:   p,
		mux:       http.NewServeMux(),
		discovery: discovery.NewDiscoveryEngine(context.Background(), wasmDir),
		wasmDir:   wasmDir,
	}
	ps.routes()
	return ps
}

func (ps *ProfileServer) routes() {
	ps.mux.HandleFunc("GET /sse", ps.handleSSE)
	ps.mux.HandleFunc("POST /message", ps.handleMessage) // MCP JSON-RPC endpoint
}

func (ps *ProfileServer) Start() error {
	ps.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ps.profile.Port),
		Handler: ps.mux,
	}
	fmt.Printf("Starting profile server '%s' on port %d\n", ps.profile.ID, ps.profile.Port)
	return ps.server.ListenAndServe()
}

func (ps *ProfileServer) Stop() error {
	if ps.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return ps.server.Shutdown(ctx)
	}
	return nil
}

func (ps *ProfileServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	// ... (SSE implementation remains similar, but now tied to discovery session)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// Notify about active tools on connection
	active := ps.discovery.ListActive()
	activeData, _ := json.Marshal(active)
	fmt.Fprintf(w, "event: tools\ndata: %s\n\n", string(activeData))
	flusher.Flush()

	for {
		select {
		case <-ticker.C:
			fmt.Fprintf(w, "event: pulse\ndata: {\"profile\": \"%s\", \"status\": \"ok\", \"timestamp\": \"%s\"}\n\n", ps.profile.ID, time.Now().Format(time.RFC3339))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (ps *ProfileServer) handleMessage(w http.ResponseWriter, r *http.Request) {
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
				"name":    "mcp-scout",
				"version": "0.1.0",
			},
		})

	case "list_tools":
		tools := ps.discovery.Find("")
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
		result, err := ps.discovery.CallTool(params.Name, params.Arguments)
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

// ProfileManager manages active profile servers.
type ProfileManager struct {
	mu       sync.RWMutex
	profiles []profile.Profile
	servers  map[string]*ProfileServer
	wasmDir  string
}

func NewProfileManager(initial []profile.Profile, wasmDir string) *ProfileManager {
	pm := &ProfileManager{
		profiles: initial,
		servers:  make(map[string]*ProfileServer),
		wasmDir:  wasmDir,
	}
	for _, p := range initial {
		pm.startServer(p)
	}
	return pm
}

func (pm *ProfileManager) GetProfiles() []profile.Profile {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.profiles
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
	pm.startServer(p)
	return nil
}

func (pm *ProfileManager) UpdateProfile(p profile.Profile) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i, existing := range pm.profiles {
		if existing.ID == p.ID {
			pm.stopServer(existing.ID)
			pm.profiles[i] = p
			pm.startServer(p)
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
			pm.stopServer(id)
			pm.profiles = append(pm.profiles[:i], pm.profiles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("profile not found")
}

func (pm *ProfileManager) startServer(p profile.Profile) {
	if p.Port == 0 {
		return // Don't start server if port is 0
	}
	ps := NewProfileServer(p, pm.wasmDir)
	pm.servers[p.ID] = ps
	go func() {
		if err := ps.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Profile server '%s' failed: %v\n", p.ID, err)
		}
	}()
}

func (pm *ProfileManager) stopServer(id string) {
	if ps, ok := pm.servers[id]; ok {
		ps.Stop()
		delete(pm.servers, id)
	}
}
