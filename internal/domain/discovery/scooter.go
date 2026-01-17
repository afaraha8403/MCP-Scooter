package discovery

// Tool represents an MCP tool definition.
type Tool struct {
	Name        string
	Description string
	Source      string // "local" or "registry"
}

// Registry defines where to look for tools.
type Registry interface {
	Search(query string) ([]Tool, error)
}

// Engine handles tool discovery and management.
type Engine struct {
	registry    Registry
	activeTools map[string]bool
}

// NewEngine creates a new discovery engine.
func NewEngine(r Registry) *Engine {
	return &Engine{
		registry:    r,
		activeTools: make(map[string]bool),
	}
}

// Find searches for tools using the registry.
func (e *Engine) Find(query string) ([]Tool, error) {
	return e.registry.Search(query)
}

// Add enables a tool for the current session.
func (e *Engine) Add(toolName string) error {
	e.activeTools[toolName] = true
	return nil
}

// ListActive returns a list of currently loaded tools.
func (e *Engine) ListActive() []string {
	keys := make([]string, 0, len(e.activeTools))
	for k := range e.activeTools {
		keys = append(keys, k)
	}
	return keys
}
