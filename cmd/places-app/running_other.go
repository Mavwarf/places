//go:build !windows

package main

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/Mavwarf/places/internal/config"
)

// RunningSession represents a detected running action for a place.
type RunningSession struct {
	Name    string `json:"name"`
	Action  string `json:"action"`
	Desktop int    `json:"desktop,omitempty"`
	Elapsed int    `json:"elapsed,omitempty"` // seconds since session start
	Today   int    `json:"today,omitempty"`   // total seconds today
}

// detectRunningSessions detects running sessions by two methods:
//  1. Command line scanning — matches processes whose args contain a place path
//  2. Working directory scanning — finds claude/code/shell processes and checks
//     their cwd via lsof against known place paths
func detectRunningSessions(placeNames []string) []RunningSession {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}

	// Build path→name mapping for matching.
	pathToName := make(map[string]string, len(cfg.Places))
	for name, place := range cfg.Places {
		if place != nil {
			pathToName[strings.ToLower(place.Path)] = strings.ToLower(name)
		}
	}

	if len(pathToName) == 0 {
		return nil
	}

	// Build exe→action name map from custom actions.
	exeToAction := make(map[string]string)
	for actionName, act := range cfg.Actions {
		cmdLower := strings.ToLower(act.Cmd)
		for _, part := range strings.Fields(cmdLower) {
			part = strings.Trim(part, `"'`)
			if strings.Contains(part, "/") {
				idx := strings.LastIndex(part, "/")
				part = part[idx+1:]
			}
			if part != "" && part != "sh" && part != "bash" && part != "zsh" &&
				part != "-c" && part != "open" && part != "cd" &&
				part != "{path}" && part != "{name}" {
				exeToAction[part] = actionName
				break
			}
		}
	}

	var sessions []RunningSession

	// --- Method 1: Command line scanning ---
	// Catches processes launched with explicit paths in args (e.g. "code /path").
	out, err := exec.Command("ps", "-eo", "command").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if line == "" || strings.HasPrefix(line, "COMMAND") {
				continue
			}
			lineLower := strings.ToLower(line)
			for path, name := range pathToName {
				if !strings.Contains(lineLower, path) {
					continue
				}
				if strings.Contains(lineLower, "claude") &&
					!strings.Contains(lineLower, "places-app") {
					sessions = append(sessions, RunningSession{Name: name, Action: "claude"})
				} else if strings.Contains(lineLower, "/code") ||
					strings.HasPrefix(lineLower, "code ") {
					sessions = append(sessions, RunningSession{Name: name, Action: "code"})
				} else {
					for exe, actionName := range exeToAction {
						if strings.Contains(lineLower, exe) {
							sessions = append(sessions, RunningSession{Name: name, Action: actionName})
							break
						}
					}
				}
			}
		}
	}

	// --- Method 2: Working directory scanning ---
	// Finds claude/code processes by name, then checks their cwd via lsof.
	// This catches processes like "claude --continue" where the path isn't
	// in the command line but is the working directory.
	pidOut, err := exec.Command("ps", "-eo", "pid,command").Output()
	if err == nil {
		type candidate struct {
			pid    int
			action string
		}
		var candidates []candidate
		for _, line := range strings.Split(string(pidOut), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "PID") {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			if len(parts) < 2 {
				continue
			}
			pid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				continue
			}
			cmd := strings.ToLower(parts[1])

			// Skip places-app itself.
			if strings.Contains(cmd, "places-app") {
				continue
			}

			if strings.Contains(cmd, "claude") && !strings.Contains(cmd, "cloudd") {
				candidates = append(candidates, candidate{pid, "claude"})
			} else if strings.HasPrefix(cmd, "code ") || strings.Contains(cmd, "/code") {
				candidates = append(candidates, candidate{pid, "code"})
			}
		}

		// Batch lsof call for all candidate PIDs.
		if len(candidates) > 0 {
			pidStrs := make([]string, len(candidates))
			for i, c := range candidates {
				pidStrs[i] = strconv.Itoa(c.pid)
			}
			lsofOut, err := exec.Command("lsof", "-a", "-d", "cwd",
				"-p", strings.Join(pidStrs, ",")).Output()
			if err == nil {
				// Parse lsof output: "COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME"
				// The NAME column (last) is the cwd.
				pidToCwd := make(map[int]string)
				for _, line := range strings.Split(string(lsofOut), "\n") {
					fields := strings.Fields(line)
					if len(fields) < 9 || fields[0] == "COMMAND" {
						continue
					}
					pid, err := strconv.Atoi(fields[1])
					if err != nil {
						continue
					}
					// NAME is everything from field 8 onwards (path may have spaces).
					cwd := strings.Join(fields[8:], " ")
					pidToCwd[pid] = cwd
				}

				for _, c := range candidates {
					cwd, ok := pidToCwd[c.pid]
					if !ok {
						continue
					}
					cwdLower := strings.ToLower(cwd)
					if name, ok := pathToName[cwdLower]; ok {
						sessions = append(sessions, RunningSession{
							Name:   name,
							Action: c.action,
						})
					}
				}
			}
		}
	}

	return dedup(sessions)
}

func dedup(sessions []RunningSession) []RunningSession {
	seen := make(map[string]bool)
	var result []RunningSession
	for _, s := range sessions {
		key := s.Name + ":" + s.Action
		if !seen[key] {
			seen[key] = true
			result = append(result, s)
		}
	}
	return result
}
