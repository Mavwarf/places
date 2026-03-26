// Package config handles loading, saving, and migrating the places.json file.
// The config file lives at ~/.config/places/places.json and stores bookmarked
// directories with usage statistics (use count, timestamps, tags, favorites).
//
// Concurrency: the exported Lock/Unlock functions serialize read-modify-write
// cycles within a process (the desktop app has multiple goroutines hitting the
// config). The Save function uses atomic write (temp file + rename) to prevent
// corruption from concurrent external reads.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// validName matches names that are safe for shell hooks and HTML:
// alphanumeric, hyphens, underscores, dots. Must not start with a dash.
var validName = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.\-]*$`)

// ValidateName returns an error if the name contains characters that could
// break shell hooks or HTML rendering.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("name too long (max 64 characters)")
	}
	if !validName.MatchString(name) {
		return fmt.Errorf("name %q contains invalid characters (use letters, numbers, hyphens, underscores, dots)", name)
	}
	return nil
}

// mu serializes config read-modify-write cycles within a process.
// Callers that do Load→modify→Save should wrap the cycle with Lock/Unlock.
var mu sync.Mutex

// Lock acquires the config mutex. Use before a Load→modify→Save cycle.
func Lock() { mu.Lock() }

// Unlock releases the config mutex.
func Unlock() { mu.Unlock() }

// Action defines a user-created command that can be assigned to places.
// The Cmd template supports {path} and {name} placeholders.
type Action struct {
	Label string `json:"label"` // short button text (e.g. "git", "JB")
	Cmd   string `json:"cmd"`   // shell command template with {path} and {name}
}

// Place holds a bookmarked directory with usage statistics.
type Place struct {
	Path       string    `json:"path"`
	AddedAt    time.Time `json:"added_at"`
	UseCount   int       `json:"use_count"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Favorite   bool      `json:"favorite,omitempty"`
	Desktop    int       `json:"desktop,omitempty"`
	Actions        []string  `json:"actions,omitempty"`         // names of assigned custom actions
	Note           string    `json:"note,omitempty"`
	HiddenDefaults []string  `json:"hidden_defaults,omitempty"` // hidden built-in actions
}

// Config holds the saved places and custom actions.
// RecentEntry records a recently launched place+action pair.
type RecentEntry struct {
	Name   string `json:"name"`
	Action string `json:"action"`
	Shift  bool   `json:"shift,omitempty"`
	Ctrl   bool   `json:"ctrl,omitempty"`
}

type Config struct {
	Actions       map[string]*Action `json:"actions,omitempty"`
	Places        map[string]*Place  `json:"places"`
	NotifyPath    string             `json:"notify_path,omitempty"`
	DefaultHidden  []string           `json:"default_hidden,omitempty"`
	DefaultActions []string           `json:"default_actions,omitempty"`
	Recent        []RecentEntry      `json:"recent,omitempty"`
	ClaudeShell   string             `json:"claude_shell,omitempty"`    // "cmd" (default) or "powershell"
	SuppressTitle bool               `json:"suppress_title,omitempty"` // suppress application title in Windows Terminal
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
			return Config{Places: make(map[string]*Place), Actions: make(map[string]*Action)}, nil
		}
		return Config{}, fmt.Errorf("reading config: %w", err)
	}

	// Try new format first ({"places": {"name": {Place object}}}).
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}

	// Migration from v1 format: the old format stored places as
	// {"places": {"name": "/path/to/dir"}} (string values instead of objects).
	// When json.Unmarshal sees a string where it expects a *Place, the Place
	// deserializes with all zero values — we detect this by checking for empty Path.
	if cfg.Places == nil {
		cfg.Places = make(map[string]*Place)
	}
	if cfg.Actions == nil {
		cfg.Actions = make(map[string]*Action)
	}
	needsMigration := false
	for _, place := range cfg.Places {
		if place != nil && place.Path == "" {
			needsMigration = true
			break
		}
	}
	if needsMigration {
		// Re-parse with RawMessage to handle the mixed types.
		var raw struct {
			Places map[string]json.RawMessage `json:"places"`
		}
		json.Unmarshal(data, &raw)
		migrated := make(map[string]*Place, len(raw.Places))
		for name, v := range raw.Places {
			// Try to unmarshal as a bare string (old format).
			var s string
			if json.Unmarshal(v, &s) == nil {
				migrated[name] = &Place{
					Path:    s,
					AddedAt: time.Now(),
				}
				continue
			}
			// Otherwise keep the already-parsed Place object.
			if p, ok := cfg.Places[name]; ok {
				migrated[name] = p
			}
		}
		cfg.Places = migrated
		// Persist the migrated format so this only happens once.
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
// Uses atomic write (write to .tmp file, then rename) to prevent corruption
// if the process is interrupted mid-write or another process reads concurrently.
// On Windows, os.Rename replaces the destination atomically.
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
		os.Remove(tmp) // clean up temp file on rename failure
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

// FilterNames returns names filtered by tag and/or favorite status.
func FilterNames(cfg Config, names []string, tagFilter string, favOnly bool) []string {
	if tagFilter == "" && !favOnly {
		return names
	}
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		p := cfg.Places[name]
		if tagFilter != "" {
			hasTag := false
			for _, t := range p.Tags {
				if t == tagFilter {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}
		if favOnly && !p.Favorite {
			continue
		}
		filtered = append(filtered, name)
	}
	return filtered
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

// AddAction adds an action name to a place's action list (deduplicated, sorted).
func AddAction(place *Place, actionName string) {
	for _, a := range place.Actions {
		if a == actionName {
			return
		}
	}
	place.Actions = append(place.Actions, actionName)
	sort.Strings(place.Actions)
}

// RemoveAction removes an action name from a place's action list.
// Returns true if the action was found and removed.
func RemoveAction(place *Place, actionName string) bool {
	for i, a := range place.Actions {
		if a == actionName {
			place.Actions = append(place.Actions[:i], place.Actions[i+1:]...)
			if len(place.Actions) == 0 {
				place.Actions = nil
			}
			return true
		}
	}
	return false
}

// SortedActionNames returns action names from cfg.Actions sorted alphabetically.
func SortedActionNames(cfg Config) []string {
	names := make([]string, 0, len(cfg.Actions))
	for name := range cfg.Actions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
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
