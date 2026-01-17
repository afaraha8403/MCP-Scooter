package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mcp-scooter/scooter/internal/domain/profile"
	"github.com/stretchr/testify/assert"
)

func TestMcpGatewaySSE(t *testing.T) {
	pm := NewProfileManager(nil, ".", ".", ".")
	p := profile.Profile{ID: "test"}
	pm.AddProfile(p)
	settings := profile.DefaultSettings()
	gw := NewMcpGateway(pm, settings)
	
	req := httptest.NewRequest("GET", "/profiles/test/sse", nil)
	w := httptest.NewRecorder()

	// Use a context with timeout to stop the SSE handler
	ctx, cancel := context.WithTimeout(req.Context(), 100*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	go gw.ServeHTTP(w, req)

	// Wait a bit for the first event
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
}

func TestControlServerCRUD(t *testing.T) {
	pm := NewProfileManager(nil, ".", ".", ".")
	settings := profile.DefaultSettings()
	srv := NewControlServer(nil, pm, settings, false)

	// 1. Create Profile
	p := profile.Profile{ID: "work"}
	pJSON, _ := json.Marshal(p)
	req := httptest.NewRequest("POST", "/api/profiles", strings.NewReader(string(pJSON)))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 2. Get Profiles
	req = httptest.NewRequest("GET", "/api/profiles", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Profiles []struct {
			profile.Profile
			Running bool `json:"running"`
		} `json:"profiles"`
		OnboardingRequired bool `json:"onboarding_required"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Len(t, resp.Profiles, 1)
	assert.Equal(t, "work", resp.Profiles[0].ID)

	// 3. Update Profile
	p.RemoteServerURL = "http://remote"
	pJSON, _ = json.Marshal(p)
	req = httptest.NewRequest("PUT", "/api/profiles", strings.NewReader(string(pJSON)))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Delete Profile
	req = httptest.NewRequest("DELETE", "/api/profiles?id=work", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify empty
	req = httptest.NewRequest("GET", "/api/profiles", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Empty(t, resp.Profiles)
}
