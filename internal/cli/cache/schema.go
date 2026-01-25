package cache

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mcp-scooter/scooter/internal/domain/registry"
)

type SchemaCache struct {
	dir string
}

func NewSchemaCache(dir string) *SchemaCache {
	return &SchemaCache{dir: dir}
}

func (c *SchemaCache) Get(server, tool string) (*registry.JSONSchema, bool) {
	path := filepath.Join(c.dir, server, tool+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var schema registry.JSONSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, false
	}
	return &schema, true
}

func (c *SchemaCache) Set(server, tool string, schema *registry.JSONSchema) error {
	dir := filepath.Join(c.dir, server)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, tool+".json"), data, 0644)
}
