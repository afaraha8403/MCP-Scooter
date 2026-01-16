package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mcp-scout/scooter/internal/domain/profile"
	"github.com/stretchr/testify/assert"
)

func TestProfileServerSSE(t *testing.T) {
	p := profile.Profile{ID: "test", Port: 9999}
	ps := NewProfileServer(p, ".")
	req := httptest.NewRequest("GET", "/sse", nil)
	w := httptest.NewRecorder()

	// Use a context with timeout to stop the SSE handler
	// However, the handler uses a ticker and context.Done(), so we can't easily test it with httptest.Recorder() for ongoing stream
	// but we can check headers and the first flush.

	// We'll wrap ps.mux.ServeHTTP logic or just call the handler
	go ps.mux.ServeHTTP(w, req)

	// Wait a bit for the first event
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
}

func TestControlServerCRUD(t *testing.T) {
	pm := NewProfileManager(nil, ".")
	srv := NewControlServer(nil, pm, false)

	// 1. Create Profile
	p := profile.Profile{ID: "work", Port: 6277}
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
	p.Port = 6278
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
