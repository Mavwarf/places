//go:build windows

// Running session detection on Windows.
//
// Two complementary strategies detect which places have active sessions:
//
//  1. Window title scanning (EnumWindows) — fast, catches VS Code (title
//     contains "Visual Studio Code" + folder name), Explorer (title contains
//     path or place name), and Claude when custom tab titles are enabled
//     (title starts with "claude <name>").
//
//  2. Process command line scanning (wmic) — catches Claude even when it
//     overrides the tab title, plus cmd/PowerShell terminals and custom
//     action executables. Works by finding shell processes whose command
//     line contains both "claude" and a known place path, or matching
//     custom action exe names against running processes.
//
// The wmic call runs with CREATE_NO_WINDOW (0x08000000) to prevent
// visible console flashes. Output is parsed as Name=/CommandLine= pairs
// separated by blank lines.

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

// user32 procs for window enumeration. The user32 DLL itself is loaded
// lazily in shift_windows.go — these just register additional procedures.
var (
	enumWindows     = user32.NewProc("EnumWindows")
	getWindowTextW  = user32.NewProc("GetWindowTextW")
	isWindowVisible = user32.NewProc("IsWindowVisible")
)

// RunningSession represents a detected running action for a place.
// Elapsed and Today are populated by the session tracker in main.go,
// not by the detection logic here.
type RunningSession struct {
	Name    string `json:"name"`
	Action  string `json:"action"`
	Desktop int    `json:"desktop,omitempty"` // 1-indexed virtual desktop, 0 = unknown
	Elapsed int    `json:"elapsed,omitempty"` // seconds since session start (from tracker)
	Today   int    `json:"today,omitempty"`   // total seconds today (from tracker)
}

// detectRunningSessions scans for running sessions across all action types.
// Called every 5 seconds when "Detect running sessions" is enabled.
// Returns deduplicated sessions matched against the given place names.
func detectRunningSessions(placeNames []string) []RunningSession {
	var sessions []RunningSession
	nameSet := make(map[string]bool, len(placeNames))
	for _, n := range placeNames {
		nameSet[strings.ToLower(n)] = true
	}

	// Build path→name lookup from config. Paths are normalized to lowercase
	// backslashes for case-insensitive matching against wmic output.
	cfg, err := config.Load()
	pathToName := make(map[string]string)
	if err == nil {
		for name, place := range cfg.Places {
			if place != nil {
				p := strings.ToLower(strings.ReplaceAll(place.Path, "/", "\\"))
				pathToName[p] = strings.ToLower(name)
			}
		}
	}

	// ── Strategy 1: Window title scanning ──
	// EnumWindows iterates all top-level windows. The callback checks each
	// visible window's title against known patterns.
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		// Skip invisible windows (background processes, minimized-to-tray, etc.).
		vis, _, _ := isWindowVisible.Call(hwnd)
		if vis == 0 {
			return 1 // continue enumeration
		}

		buf := make([]uint16, 512)
		getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 512)
		title := syscall.UTF16ToString(buf)
		if title == "" {
			return 1
		}
		titleLower := strings.ToLower(title)

		// getDesktop resolves which virtual desktop a window lives on.
		// Returns 0 if the DLL is unavailable or the window is on an unknown desktop.
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

		// Match our custom Claude title: "claude <name>" or "claude <name> YOLO".
		// Only works when "Set tab title" preference is on (suppressTitle=true),
		// because Claude Code overrides the title when suppression is off.
		if strings.HasPrefix(titleLower, "claude ") {
			rest := titleLower[7:]
			rest = strings.TrimSuffix(rest, " yolo")
			rest = strings.TrimSpace(rest)
			if nameSet[rest] {
				sessions = append(sessions, RunningSession{Name: rest, Action: "claude", Desktop: getDesktop(hwnd)})
				return 1
			}
		}

		// Match VS Code: title is typically "<folder> - <file> - Visual Studio Code".
		if strings.Contains(titleLower, "visual studio code") {
			for _, n := range placeNames {
				if strings.Contains(titleLower, strings.ToLower(n)) {
					sessions = append(sessions, RunningSession{Name: strings.ToLower(n), Action: "code", Desktop: getDesktop(hwnd)})
					return 1
				}
			}
		}

		// Match Explorer: title is "<folder name> - File Explorer" or shows the full path.
		if strings.Contains(titleLower, "file explorer") || strings.Contains(titleLower, "explorer") {
			for path, name := range pathToName {
				if strings.Contains(titleLower, path) || strings.Contains(titleLower, name) {
					sessions = append(sessions, RunningSession{Name: name, Action: "explorer", Desktop: getDesktop(hwnd)})
					return 1
				}
			}
		}

		return 1 // continue enumeration
	})
	enumWindows.Call(cb, 0)

	// ── Strategy 2: Process command line scanning via wmic ──
	// Fetches Name and CommandLine for all processes. Matches against:
	//   - cmd.exe/powershell.exe with place path + "claude" → Claude session
	//   - cmd.exe/powershell.exe with place path (no claude) → terminal
	//   - code.exe with place path → VS Code
	//   - Custom action exe names with place path → custom action

	// Build exe→action name map from custom action command templates.
	// Parses the .exe filename from the Cmd string (e.g. "start "" "...\webstorm64.exe" "{path}").
	exeToAction := make(map[string]string)
	if err == nil {
		for actionName, act := range cfg.Actions {
			cmdLower := strings.ToLower(act.Cmd)
			for _, part := range strings.Split(cmdLower, "\"") {
				part = strings.TrimSpace(part)
				if strings.HasSuffix(part, ".exe") {
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

	// Run wmic with CREATE_NO_WINDOW to avoid visible console flash.
	// Output format (list): alternating "CommandLine=..." and "Name=..." lines
	// separated by blank lines between records.
	wmicCmd := exec.Command("wmic", "process",
		"get", "Name,CommandLine", "/format:list")
	wmicCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	out, wmicErr := wmicCmd.Output()
	if wmicErr == nil {
		var currentName, currentCmd string
		for _, line := range bytes.Split(out, []byte("\r\n")) {
			s := strings.TrimSpace(string(line))
			if strings.HasPrefix(s, "CommandLine=") {
				currentCmd = strings.ToLower(s[12:])
			} else if strings.HasPrefix(s, "Name=") {
				currentName = strings.ToLower(s[5:])
			} else if s == "" && currentName != "" && currentCmd != "" {
				// End of record — match against known patterns.

				// Shell with a place path: distinguish Claude parent vs plain terminal.
				if currentName == "cmd.exe" || currentName == "powershell.exe" {
					for path, name := range pathToName {
						if strings.Contains(currentCmd, path) {
							if strings.Contains(currentCmd, "claude") {
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

				// VS Code process (fallback if window title scan missed it).
				if currentName == "code.exe" {
					for path, name := range pathToName {
						if strings.Contains(currentCmd, path) {
							sessions = append(sessions, RunningSession{Name: name, Action: "code"})
							break
						}
					}
				}

				// Custom action: exe name matches a defined action's command template.
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

// dedup removes duplicate place:action pairs, keeping the first occurrence
// (which may have Desktop info from window scanning).
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
