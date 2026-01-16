package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mcp-scout/scooter/internal/domain/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "profile-store-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "profiles.yaml")
	store := profile.NewStore(path)

	profiles := []profile.Profile{
		{ID: "test1", Port: 6277},
		{ID: "test2", Port: 6278},
	}

	// Test Save
	err = store.Save(profiles)
	assert.NoError(t, err)

	// Test Load
	loaded, err := store.Load()
	assert.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Equal(t, "test1", loaded[0].ID)
	assert.Equal(t, "test2", loaded[1].ID)
}

func TestStore_LoadNonExistent(t *testing.T) {
	store := profile.NewStore("non-existent.yaml")
	loaded, err := store.Load()
	assert.NoError(t, err)
	assert.Empty(t, loaded)
}
