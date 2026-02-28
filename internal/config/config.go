package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Place holds a bookmarked directory with usage statistics.
type Place struct {
	Path       string    `json:"path"`
	AddedAt    time.Time `json:"added_at"`
	UseCount   int       `json:"use_count"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
}

// Config holds the saved places.
type Config struct {
	Places map[string]*Place `json:"places"`
}

// configDir returns the directory for places config files.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "places"), nil
}

// ConfigPath returns the full path to places.json, creating the directory if needed.
func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return filepath.Join(dir, "places.json"), nil
}

// Load reads and parses the config file. Returns an empty config if the file
// does not exist. Handles migration from the old map[string]string format.
func Load() (Config, error) {
	p, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{Places: make(map[string]*Place)}, nil
		}
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	// Try new format first.
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	// Migrate from old format: if any place deserialized with an empty path,
	// the value was a plain string in the old format.
	if cfg.Places == nil {
		cfg.Places = make(map[string]*Place)
	}
	needsMigration := false
	for _, place := range cfg.Places {
		if place != nil && place.Path == "" {
			needsMigration = true
			break
		}
	}
	if needsMigration {
		var raw struct {
			Places map[string]json.RawMessage `json:"places"`
		}
		json.Unmarshal(data, &raw)
		migrated := make(map[string]*Place, len(raw.Places))
		for name, v := range raw.Places {
			// Try as string (old format).
			var s string
			if json.Unmarshal(v, &s) == nil {
				migrated[name] = &Place{
					Path:    s,
					AddedAt: time.Now(),
				}
				continue
			}
			// Otherwise keep the parsed Place.
			if p, ok := cfg.Places[name]; ok {
				migrated[name] = p
			}
		}
		cfg.Places = migrated
		// Save migrated format.
		Save(cfg)
	}

	return cfg, nil
}

// Save writes the config to disk as formatted JSON.
func Save(cfg Config) error {
	p, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// RecordUse increments the use count and updates the last-used timestamp.
func RecordUse(place *Place) {
	place.UseCount++
	place.LastUsedAt = time.Now()
}
