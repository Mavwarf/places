//go:build windows

package main

import (
	"bytes"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Mavwarf/places/internal/config"
	"github.com/Mavwarf/places/internal/desktop"
)

var (
	enumWindows     = user32.NewProc("EnumWindows")
	getWindowTextW  = user32.NewProc("GetWindowTextW")
	isWindowVisible = user32.NewProc("IsWindowVisible")
)

// RunningSession represents a detected running action for a place.
type RunningSession struct {
	Name    string `json:"name"`
	Action  string `json:"action"`
	Desktop int    `json:"desktop,omitempty"` // 1-indexed, 0 = unknown
	Elapsed int    `json:"elapsed,omitempty"` // seconds since session start
	Today   int    `json:"today,omitempty"`   // total seconds today
}

// detectRunningSessions detects running Claude and VS Code sessions.
// Uses window title scanning for VS Code and custom Claude titles,
// plus process parent command line scanning for Claude (which overrides titles).
func detectRunningSessions(placeNames []string) []RunningSession {
	var sessions []RunningSession
	nameSet := make(map[string]bool, len(placeNames))
	for _, n := range placeNames {
		nameSet[strings.ToLower(n)] = true
	}

	// Build path→name mapping from config for process scanning.
	cfg, err := config.Load()
	pathToName := make(map[string]string)
	if err == nil {
		for name, place := range cfg.Places {
			if place != nil {
				// Normalize path to lowercase with backslashes for matching.
				p := strings.ToLower(strings.ReplaceAll(place.Path, "/", "\\"))
				pathToName[p] = strings.ToLower(name)
			}
		}
	}

	// --- Window title scanning ---
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		vis, _, _ := isWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1
		}
		buf := make([]uint16, 512)
		getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 512)
		title := syscall.UTF16ToString(buf)
		if title == "" {
			return 1
		}
		titleLower := strings.ToLower(title)

		getDesktop := func(h uintptr) int {
			if !desktop.Available() {
				return 0
			}
			d, _ := desktop.WindowDesktop(h)
			if d < 0 {
				return 0
			}
			return d
		}

		// Match our custom Claude title (when suppressTitle is on).
		if strings.HasPrefix(titleLower, "claude ") {
			rest := titleLower[7:]
			rest = strings.TrimSuffix(rest, " yolo")
			rest = strings.TrimSpace(rest)
			if nameSet[rest] {
				sessions = append(sessions, RunningSession{Name: rest, Action: "claude", Desktop: getDesktop(hwnd)})
				return 1
			}
		}

		// Match VS Code windows.
		if strings.Contains(titleLower, "visual studio code") {
			for _, n := range placeNames {
				if strings.Contains(titleLower, strings.ToLower(n)) {
					sessions = append(sessions, RunningSession{Name: strings.ToLower(n), Action: "code", Desktop: getDesktop(hwnd)})
					return 1
				}
			}
		}

		// Match Explorer windows (title like "places - File Explorer" or path-based).
		if strings.Contains(titleLower, "file explorer") || strings.Contains(titleLower, "explorer") {
			for path, name := range pathToName {
				if strings.Contains(titleLower, path) || strings.Contains(titleLower, name) {
					sessions = append(sessions, RunningSession{Name: name, Action: "explorer", Desktop: getDesktop(hwnd)})
					return 1
				}
			}
		}

		return 1
	})
	enumWindows.Call(cb, 0)

	// --- Process scanning for Claude and custom actions ---
	// Use wmic to find processes whose command line contains place paths.
	// Matches Claude (via cmd.exe/powershell.exe parents) and custom action
	// executables (via their exe name in the command line).

	// Build exe→action name map from custom actions.
	exeToAction := make(map[string]string)
	if err == nil {
		for actionName, act := range cfg.Actions {
			// Extract exe name from command template (e.g. "start "" "C:\...\webstorm64.exe" "{path}"")
			cmdLower := strings.ToLower(act.Cmd)
			for _, part := range strings.Split(cmdLower, "\"") {
				part = strings.TrimSpace(part)
				if strings.HasSuffix(part, ".exe") {
					// Get just the filename.
					idx := strings.LastIndex(part, "\\")
					if idx >= 0 {
						part = part[idx+1:]
					}
					exeToAction[part] = actionName
					break
				}
			}
		}
	}

	// Build map of place path→assigned actions for matching.
	placeActions := make(map[string][]string)
	if err == nil {
		for name, place := range cfg.Places {
			if place != nil {
				p := strings.ToLower(strings.ReplaceAll(place.Path, "/", "\\"))
				placeActions[p] = place.Actions
				_ = name
			}
		}
	}

	wmicCmd := exec.Command("wmic", "process",
		"get", "Name,CommandLine", "/format:list")
	wmicCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	out, wmicErr := wmicCmd.Output()
	if wmicErr == nil {
		// Parse wmic output into records (Name= and CommandLine= pairs).
		var currentName, currentCmd string
		for _, line := range bytes.Split(out, []byte("\r\n")) {
			s := strings.TrimSpace(string(line))
			if strings.HasPrefix(s, "CommandLine=") {
				currentCmd = strings.ToLower(s[12:])
			} else if strings.HasPrefix(s, "Name=") {
				currentName = strings.ToLower(s[5:])
			} else if s == "" && currentName != "" && currentCmd != "" {
				// Process the record.

				// Shell processes with a place path in the command line.
				if currentName == "cmd.exe" || currentName == "powershell.exe" {
					for path, name := range pathToName {
						if strings.Contains(currentCmd, path) {
							if strings.Contains(currentCmd, "claude") {
								// Claude parent shell.
								sessions = append(sessions, RunningSession{Name: name, Action: "claude"})
							} else if currentName == "powershell.exe" {
								sessions = append(sessions, RunningSession{Name: name, Action: "powershell"})
							} else {
								sessions = append(sessions, RunningSession{Name: name, Action: "cmd"})
							}
							break
						}
					}
				}

				// VS Code detection via process.
				if currentName == "code.exe" {
					for path, name := range pathToName {
						if strings.Contains(currentCmd, path) {
							sessions = append(sessions, RunningSession{Name: name, Action: "code"})
							break
						}
					}
				}

				// Custom action detection: exe matches a known action and command line contains a place path.
				if actionName, ok := exeToAction[currentName]; ok {
					for path, placeName := range pathToName {
						if strings.Contains(currentCmd, path) {
							sessions = append(sessions, RunningSession{Name: placeName, Action: actionName})
							break
						}
					}
				}

				currentName = ""
				currentCmd = ""
			}
		}
	}

	return dedup(sessions)
}

func dedup(sessions []RunningSession) []RunningSession {
	seen := make(map[string]bool)
	result := sessions[:0]
	for _, s := range sessions {
		key := s.Name + ":" + s.Action
		if !seen[key] {
			seen[key] = true
			result = append(result, s)
		}
	}
	return result
}
