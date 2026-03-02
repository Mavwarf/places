package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// mu serializes config read-modify-write cycles within a process.
// Callers that do Load→modify→Save should wrap the cycle with Lock/Unlock.
var mu sync.Mutex

// Lock acquires the config mutex. Use before a Load→modify→Save cycle.
func Lock() { mu.Lock() }

// Unlock releases the config mutex.
func Unlock() { mu.Unlock() }

// Place holds a bookmarked directory with usage statistics.
type Place struct {
	Path       string    `json:"path"`
	AddedAt    time.Time `json:"added_at"`
	UseCount   int       `json:"use_count"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Favorite   bool      `json:"favorite,omitempty"`
	Desktop    int       `json:"desktop,omitempty"`
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
		if err := Save(cfg); err != nil {
			return Config{}, fmt.Errorf("saving migrated config: %w", err)
		}
	}

	// Normalize path separators to OS convention.
	for _, place := range cfg.Places {
		place.Path = filepath.Clean(place.Path)
	}

	return cfg, nil
}

// Save writes the config to disk as formatted JSON.
// Uses atomic write (temp file + rename) to prevent corruption from
// concurrent reads or interrupted writes.
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

	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// RecordUse increments the use count and updates the last-used timestamp.
func RecordUse(place *Place) {
	place.UseCount++
	place.LastUsedAt = time.Now()
}

// SortedNames returns place names sorted alphabetically.
func SortedNames(cfg Config) []string {
	names := make([]string, 0, len(cfg.Places))
	for name := range cfg.Places {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// AddTag adds a tag to a place (lowercase, trimmed, deduplicated, sorted).
func AddTag(place *Place, tag string) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if tag == "" {
		return
	}
	for _, t := range place.Tags {
		if t == tag {
			return
		}
	}
	place.Tags = append(place.Tags, tag)
	sort.Strings(place.Tags)
}

// RemoveTag removes a tag from a place. Returns true if the tag was found.
func RemoveTag(place *Place, tag string) bool {
	tag = strings.ToLower(strings.TrimSpace(tag))
	for i, t := range place.Tags {
		if t == tag {
			place.Tags = append(place.Tags[:i], place.Tags[i+1:]...)
			if len(place.Tags) == 0 {
				place.Tags = nil
			}
			return true
		}
	}
	return false
}

// AllTags returns all unique tags across all places, sorted alphabetically.
func AllTags(cfg Config) []string {
	seen := make(map[string]bool)
	for _, p := range cfg.Places {
		for _, t := range p.Tags {
			seen[t] = true
		}
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}
