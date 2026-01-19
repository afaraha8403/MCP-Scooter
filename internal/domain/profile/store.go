package profile

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Store handles persistence of profiles and settings to separate YAML files.
type Store struct {
	profilesPath string
	settingsPath string
}

// GetProfilesPath returns the path to the profiles file.
func (s *Store) GetProfilesPath() string {
	return s.profilesPath
}

// GetSettingsPath returns the path to the settings file.
func (s *Store) GetSettingsPath() string {
	return s.settingsPath
}

// ProfilesConfig is for the profiles.yaml file.
type ProfilesConfig struct {
	Profiles []Profile `yaml:"profiles"`
}

// SettingsConfig is for the settings.yaml file.
type SettingsConfig struct {
	Settings Settings `yaml:"settings"`
}

// NewStore creates a new profile store with separate paths for profiles and settings.
func NewStore(profilesPath, settingsPath string) *Store {
	return &Store{
		profilesPath: profilesPath,
		settingsPath: settingsPath,
	}
}

// Load reads both profiles and settings from their respective files.
func (s *Store) Load() ([]Profile, Settings, error) {
	// Load Profiles
	profiles := []Profile{}
	pData, err := os.ReadFile(s.profilesPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, Settings{}, err
		}
	} else {
		var pConfig ProfilesConfig
		if err := yaml.Unmarshal(pData, &pConfig); err != nil {
			// Backward compatibility: try loading the old combined format
			var oldConfig struct {
				Profiles []Profile `yaml:"profiles"`
				Settings Settings  `yaml:"settings"`
			}
			if err2 := yaml.Unmarshal(pData, &oldConfig); err2 == nil && len(oldConfig.Profiles) > 0 {
				profiles = oldConfig.Profiles
			} else {
				return nil, Settings{}, err
			}
		} else {
			profiles = pConfig.Profiles
		}
	}

	// Load Settings
	settings := DefaultSettings()
	sData, err := os.ReadFile(s.settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, Settings{}, err
		}
		// If settings.yaml doesn't exist, check if we can migrate from profiles.yaml (old format)
		if pData != nil {
			var oldConfig struct {
				Settings Settings `yaml:"settings"`
			}
			if err2 := yaml.Unmarshal(pData, &oldConfig); err2 == nil && oldConfig.Settings.ControlPort != 0 {
				settings = oldConfig.Settings
				// Save it to the new location immediately
				s.SaveSettings(settings)
			}
		}
	} else {
		var sConfig SettingsConfig
		if err := yaml.Unmarshal(sData, &sConfig); err != nil {
			return nil, Settings{}, err
		}
		settings = sConfig.Settings
	}

	// Ensure defaults if not set
	if settings.ControlPort == 0 {
		settings.ControlPort = DefaultSettings().ControlPort
	}
	if settings.McpPort == 0 {
		settings.McpPort = DefaultSettings().McpPort
	}

	return profiles, settings, nil
}

// SaveProfiles writes profiles to profiles.yaml.
func (s *Store) SaveProfiles(profiles []Profile) error {
	config := ProfilesConfig{
		Profiles: profiles,
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(s.profilesPath, bytes, 0644)
}

// SaveSettings writes settings to settings.yaml.
func (s *Store) SaveSettings(settings Settings) error {
	config := SettingsConfig{
		Settings: settings,
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(s.settingsPath, bytes, 0644)
}

// Save writes both for convenience (migrating old calls).
func (s *Store) Save(profiles []Profile, settings Settings) error {
	if err := s.SaveProfiles(profiles); err != nil {
		return err
	}
	return s.SaveSettings(settings)
}

// getToolParamsPath returns the path to the tool-params.json file.
func (s *Store) getToolParamsPath() string {
	dir := filepath.Dir(s.settingsPath)
	return filepath.Join(dir, "tool-params.json")
}

// LoadToolParams reads saved tool test parameters from tool-params.json.
func (s *Store) LoadToolParams() (map[string]map[string]interface{}, error) {
	data, err := os.ReadFile(s.getToolParamsPath())
	if err != nil {
		return nil, err
	}

	var params map[string]map[string]interface{}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, err
	}

	return params, nil
}

// SaveToolParams writes tool test parameters to tool-params.json.
func (s *Store) SaveToolParams(params map[string]map[string]interface{}) error {
	bytes, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.getToolParamsPath(), bytes, 0644)
}
