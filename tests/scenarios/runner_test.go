package scenarios

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mcp-scooter/scooter/tests/protocol"
	"github.com/stretchr/testify/require"
)

func TestScenario(t *testing.T) {
	scooterURL := os.Getenv("SCOOTER_URL")
	if scooterURL == "" {
		t.Skip("SCOOTER_URL not set")
	}

	definitionsDir := "definitions"
	entries, err := os.ReadDir(definitionsDir)
	require.NoError(t, err)

	client := protocol.NewClient(scooterURL, "test-brave", "")
	runner := &ScenarioRunner{Client: client}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			t.Run(entry.Name(), func(t *testing.T) {
				s, err := LoadScenario(filepath.Join(definitionsDir, entry.Name()))
				require.NoError(t, err)
				
				err = runner.Run(s)
				require.NoError(t, err)
			})
		}
	}
}
