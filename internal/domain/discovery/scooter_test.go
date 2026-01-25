package discovery_test

import (
	"context"
	"io"
	"testing"

	"github.com/mcp-scooter/scooter/internal/domain/discovery"
	"github.com/mcp-scooter/scooter/internal/domain/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorker for testing
type MockWorker struct {
	mock.Mock
}

func (m *MockWorker) Execute(stdin io.Reader, stdout io.Writer, env map[string]string) error {
	args := m.Called(stdin, stdout, env)
	return args.Error(0)
}

func (m *MockWorker) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestEngine_Find(t *testing.T) {
	engine := discovery.NewDiscoveryEngine(context.Background(), "", "")
	// DiscoveryEngine.Find currently returns all tools in registry
	tools := engine.Find("")
	assert.NotEmpty(t, tools)
}

func TestEngine_HandleBuiltinTool_ListActive(t *testing.T) {
	engine := discovery.NewDiscoveryEngine(context.Background(), "", "")
	
	// Initially empty
	res, err := engine.HandleBuiltinTool("scooter_list_active", nil)
	assert.NoError(t, err)
	
	activeInfo := res.(map[string]interface{})
	assert.Equal(t, 0, activeInfo["count"])
}

func TestEngine_HandleBuiltinTool_DeactivateAll(t *testing.T) {
	engine := discovery.NewDiscoveryEngine(context.Background(), "", "")
	
	// Test deactivating all (even if none are active)
	res, err := engine.HandleBuiltinTool("scooter_deactivate", map[string]interface{}{"all": true})
	assert.NoError(t, err)
	
	msg := res.(map[string]interface{})
	assert.Equal(t, "off", msg["status"])
}

func TestEngine_Settings_Propagation(t *testing.T) {
	engine := discovery.NewDiscoveryEngine(context.Background(), "", "")
	
	settings := profile.DefaultSettings()
	settings.AutoCleanupMinutes = 42
	engine.SetSettings(settings)
	
	// We can't easily check the private settings field, but we can verify SetSettings doesn't panic
}
