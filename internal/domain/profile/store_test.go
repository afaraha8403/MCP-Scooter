package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mcp-scooter/scooter/internal/domain/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "profile-store-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pPath := filepath.Join(tmpDir, "profiles.yaml")
	sPath := filepath.Join(tmpDir, "settings.yaml")
	store := profile.NewStore(pPath, sPath)

	profiles := []profile.Profile{
		{ID: "test1"},
		{ID: "test2"},
	}

	settings := profile.DefaultSettings()
	settings.GatewayAPIKey = "test-key"
	settings.LastProfileID = "test2"

	// Test Save
	err = store.Save(profiles, settings)
	assert.NoError(t, err)

	// Test Load
	loadedProfiles, loadedSettings, err := store.Load()
	assert.NoError(t, err)
	assert.Len(t, loadedProfiles, 2)
	assert.Equal(t, "test1", loadedProfiles[0].ID)
	assert.Equal(t, "test2", loadedProfiles[1].ID)
	assert.Equal(t, "test-key", loadedSettings.GatewayAPIKey)
	assert.Equal(t, "test2", loadedSettings.LastProfileID)
}

func TestStore_LoadNonExistent(t *testing.T) {
	store := profile.NewStore("non-existent-profiles.yaml", "non-existent-settings.yaml")
	loadedProfiles, _, err := store.Load()
	assert.NoError(t, err)
	assert.Empty(t, loadedProfiles)
}
