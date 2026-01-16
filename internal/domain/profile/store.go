package profile

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Store handles persistence of profiles to a YAML file.
type Store struct {
	path string
}

// NewStore creates a new profile store.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Load reads profiles from the file.
func (s *Store) Load() ([]Profile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Profile{}, nil
		}
		return nil, err
	}

	var profiles struct {
		Profiles []Profile `yaml:"profiles"`
	}
	if err := yaml.Unmarshal(data, &profiles); err != nil {
		return nil, err
	}

	return profiles.Profiles, nil
}

// Save writes profiles to the file.
func (s *Store) Save(profiles []Profile) error {
	data := struct {
		Profiles []Profile `yaml:"profiles"`
	}{
		Profiles: profiles,
	}

	bytes, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, bytes, 0644)
}
