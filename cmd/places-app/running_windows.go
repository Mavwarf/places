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

		return 1
	})
	enumWindows.Call(cb, 0)

	// --- Process scanning for Claude ---
	// Use wmic to find cmd.exe/powershell.exe processes whose command line
	// contains "claude" and a place path. This catches Claude sessions where
	// Claude overrides the window title.
	wmicCmd := exec.Command("wmic", "process", "where",
		"(Name='cmd.exe' OR Name='powershell.exe')",
		"get", "CommandLine", "/format:list")
	wmicCmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	out, err := wmicCmd.Output()
	if err == nil {
		for _, line := range bytes.Split(out, []byte("\r\n")) {
			s := strings.TrimSpace(string(line))
			if !strings.HasPrefix(s, "CommandLine=") {
				continue
			}
			cmdLine := strings.ToLower(s[12:])
			if !strings.Contains(cmdLine, "claude") {
				continue
			}
			// Match against known place paths.
			for path, name := range pathToName {
				if strings.Contains(cmdLine, path) {
					sessions = append(sessions, RunningSession{Name: name, Action: "claude"})
					break
				}
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
