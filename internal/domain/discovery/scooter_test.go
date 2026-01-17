package discovery_test

import (
	"testing"

	"github.com/mcp-scooter/scooter/internal/domain/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRegistry for testing
type MockRegistry struct {
	mock.Mock
}

func (m *MockRegistry) Search(query string) ([]discovery.Tool, error) {
	args := m.Called(query)
	return args.Get(0).([]discovery.Tool), args.Error(1)
}

func TestEngine_Find(t *testing.T) {
	mockReg := new(MockRegistry)
	engine := discovery.NewEngine(mockReg)

	expectedTools := []discovery.Tool{
		{Name: "jira-mcp", Description: "Jira Integration", Source: "registry"},
	}

	mockReg.On("Search", "jira").Return(expectedTools, nil)

	tools, err := engine.Find("jira")
	assert.NoError(t, err)
	assert.Equal(t, expectedTools, tools)
	mockReg.AssertExpectations(t)
}

func TestEngine_Add(t *testing.T) {
	mockReg := new(MockRegistry)
	engine := discovery.NewEngine(mockReg)

	// Simulate adding a tool
	err := engine.Add("jira-mcp")
	assert.NoError(t, err)

	// For now, check if it's "installed" (active).
	// We might need to query active tools to verify.
	active := engine.ListActive()
	assert.Contains(t, active, "jira-mcp")
}
