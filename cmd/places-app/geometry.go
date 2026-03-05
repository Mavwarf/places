package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WindowGeometry stores the last known window position and size.
type WindowGeometry struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// geometryPath returns ~/.config/places/window.json.
func geometryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "places", "window.json"), nil
}

// loadGeometry reads saved window geometry. Returns nil if the file is
// missing, unreadable, or contains invalid data.
func loadGeometry() *WindowGeometry {
	p, err := geometryPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var g WindowGeometry
	if err := json.Unmarshal(data, &g); err != nil {
		fmt.Fprintf(os.Stderr, "places-app: failed to parse window geometry: %v\n", err)
		return nil
	}
	if g.Width < 100 || g.Height < 100 {
		return nil
	}
	return &g
}

// saveGeometry writes window geometry to disk.
func saveGeometry(g WindowGeometry) {
	// Skip saving if the window is minimized (Windows reports ~-32000).
	if g.X < -10000 || g.Y < -10000 {
		return
	}
	p, err := geometryPath()
	if err != nil {
		return
	}
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "places-app: failed to marshal window geometry: %v\n", err)
		return
	}
	data = append(data, '\n')
	if err := os.WriteFile(p, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "places-app: failed to save window geometry: %v\n", err)
	}
}
