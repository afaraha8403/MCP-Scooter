package profile

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Store handles persistence of profiles and settings to a YAML file.
type Store struct {
	path string
}

// Config is the top-level structure for the YAML file.
type Config struct {
	Profiles []Profile `yaml:"profiles"`
	Settings Settings  `yaml:"settings"`
}

// NewStore creates a new profile store.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Load reads config from the file.
func (s *Store) Load() ([]Profile, Settings, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Profile{}, DefaultSettings(), nil
		}
		return nil, Settings{}, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, Settings{}, err
	}

	// Ensure defaults if not set
	if config.Settings.ControlPort == 0 {
		config.Settings.ControlPort = DefaultSettings().ControlPort
	}
	if config.Settings.McpPort == 0 {
		config.Settings.McpPort = DefaultSettings().McpPort
	}

	return config.Profiles, config.Settings, nil
}

// Save writes config to the file.
func (s *Store) Save(profiles []Profile, settings Settings) error {
	config := Config{
		Profiles: profiles,
		Settings: settings,
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, bytes, 0644)
}
