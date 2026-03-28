//go:build !windows

package main

// RunningSession represents a detected running action for a place.
type RunningSession struct {
	Name    string `json:"name"`
	Action  string `json:"action"`
	Desktop int    `json:"desktop,omitempty"`
	Elapsed int    `json:"elapsed,omitempty"` // seconds since session start
	Today   int    `json:"today,omitempty"`   // total seconds today
}

// detectRunningSessions is a no-op on non-Windows platforms.
func detectRunningSessions(placeNames []string) []RunningSession { return nil }
