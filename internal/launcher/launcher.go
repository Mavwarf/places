// Package launcher provides functions to launch external applications
// (PowerShell, cmd, Claude, VS Code, Explorer) at a given directory.
// Each function returns an *exec.Cmd — the caller is responsible for
// starting it (typically via Detach for fire-and-forget).
//
// On Windows, most launchers use "cmd /c start" to open a new console window
// detached from the caller. On Unix, they run in the current terminal.

package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/Mavwarf/places/internal/desktop"
)

// psEscape escapes a string for embedding in a PowerShell single-quoted string.
// PowerShell's only escape inside '...' is '' for a literal single quote.
func psEscape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// shellQuote wraps a string in single quotes for sh/bash, escaping embedded
// single quotes with the '\'' idiom.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// PowerShell opens a new PowerShell window at the given directory.
// On macOS this opens a new Terminal.app window; on Linux a new shell process.
func PowerShell(path string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", "start", "", "powershell", "-NoExit", "-Command",
			fmt.Sprintf("Set-Location '%s'", psEscape(path)))
	}
	return Terminal(path)
}

// hasITerm2 checks if iTerm2 is installed on macOS.
func hasITerm2() bool {
	_, err := os.Stat("/Applications/iTerm.app")
	return err == nil
}

// darwinTerminalScript returns a command that opens a new terminal window and
// runs shCmd. Prefers iTerm2 if installed, falls back to Terminal.app.
// For iTerm2, writes a temp shell script and opens it, avoiding AppleScript
// automation permission issues with .app bundles.
func darwinTerminalScript(shCmd string) *exec.Cmd {
	if hasITerm2() {
		// Write a temp script that runs the command in an interactive shell.
		// The script deletes itself after sourcing, so it doesn't linger.
		f, err := os.CreateTemp("", "places-launch-*.sh")
		if err != nil {
			// Fall back to Terminal.app on error.
			return darwinTerminalAppScript(shCmd)
		}
		script := fmt.Sprintf("#!/bin/zsh\nrm -f %q\n%s\n", f.Name(), shCmd)
		f.WriteString(script)
		f.Close()
		os.Chmod(f.Name(), 0755)
		return exec.Command("open", "-a", "iTerm", f.Name())
	}
	return darwinTerminalAppScript(shCmd)
}

// darwinTerminalAppScript opens Terminal.app and runs shCmd via AppleScript.
func darwinTerminalAppScript(shCmd string) *exec.Cmd {
	escaped := strings.ReplaceAll(shCmd, `"`, `\"`)
	script := fmt.Sprintf(`tell application "Terminal"
activate
do script "%s"
end tell`, escaped)
	return exec.Command("osascript", "-e", script)
}

// Terminal opens a new terminal window at the given directory.
// On macOS, prefers iTerm2 if installed, falls back to Terminal.app.
// On Linux, starts the user's default shell.
func Terminal(path string) *exec.Cmd {
	if runtime.GOOS == "darwin" {
		return darwinTerminalScript("cd " + shellQuote(path))
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell)
	cmd.Dir = path
	return cmd
}

// cmdEscape quotes a string for safe use as a cmd.exe argument.
// Double quotes neutralize all cmd.exe metacharacters (&, |, <, >, ^, etc.)
// which is sufficient for cd /d paths. Embedded quotes are escaped.
func cmdEscape(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// Cmd opens a new cmd.exe window at the given directory (Windows only).
func Cmd(path string) *exec.Cmd {
	return exec.Command("cmd", "/c", "start", "", "cmd", "/k", "cd", "/d", cmdEscape(path))
}

// Claude opens a new PowerShell window at the given directory and starts Claude.
// The tab title is set to "Claude Code - <name>".
//
// Windows Terminal detection: if wt.exe is on PATH, we use it for better tab
// title support. --suppressApplicationTitle prevents Claude's own title-setting
// from overriding our custom title. wt.exe uses `;` as a command separator,
// so we must escape the PowerShell `;` in our command string with `\;`.
//
// Fallback (plain conhost): "cmd /c start <title>" sets the window title, but
// Claude may override it on startup since conhost doesn't support title pinning.
func Claude(path, name string, cont, yolo bool, shell string, suppressTitle bool, effort string) *exec.Cmd {
	title := "claude " + name
	claudeCmd := "claude --continue"
	if cont {
		claudeCmd = "claude"
	}
	if yolo {
		title += " YOLO"
		claudeCmd += " --dangerously-skip-permissions"
	}
	if effort != "" {
		claudeCmd += " --effort " + effort
	}

	// Non-Windows: open a new terminal window with claude.
	if runtime.GOOS != "windows" {
		shCmd := fmt.Sprintf("cd %s && %s", shellQuote(path), claudeCmd)
		if runtime.GOOS == "darwin" {
			return darwinTerminalScript(shCmd)
		}
		// Linux / other: just run in a new shell process.
		return Shell(shCmd)
	}

	if shell == "powershell" {
		psCmd := fmt.Sprintf("Set-Location '%s'; %s", psEscape(path), claudeCmd)
		if _, err := exec.LookPath("wt.exe"); err == nil {
			wtCmd := strings.ReplaceAll(psCmd, ";", "\\;")
			args := []string{"new-tab", "--title", title}
			if suppressTitle {
				args = append(args, "--suppressApplicationTitle")
			}
			args = append(args, "powershell", "-NoExit", "-Command", wtCmd)
			return exec.Command("wt", args...)
		}
		return exec.Command("cmd", "/c", "start", title, "powershell", "-NoExit", "-Command", psCmd)
	}

	// Default: cmd (Windows)
	cmdStr := fmt.Sprintf("cd /d %s && %s", cmdEscape(path), claudeCmd)
	if _, err := exec.LookPath("wt.exe"); err == nil {
		args := []string{"new-tab", "--title", title}
		if suppressTitle {
			args = append(args, "--suppressApplicationTitle")
		}
		args = append(args, "cmd", "/k", cmdStr)
		return exec.Command("wt", args...)
	}
	return exec.Command("cmd", "/c", "start", title, "cmd", "/k", cmdStr)
}

// Explorer opens the file explorer at the given directory.
func Explorer(path string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", path)
	case "darwin":
		return exec.Command("open", path)
	default:
		return exec.Command("xdg-open", path)
	}
}

// Code opens VS Code at the given directory.
func Code(path string) *exec.Cmd {
	return exec.Command("code", path)
}

// ExpandAction replaces {path} and {name} placeholders in a command template.
// Placeholders are substituted verbatim — the template must include any
// quoting needed for paths with spaces or special characters, e.g.:
//
//	cd /d "{path}" && my-tool
//	echo {name}
func ExpandAction(cmdTpl, path, name string) string {
	s := strings.ReplaceAll(cmdTpl, "{path}", path)
	return strings.ReplaceAll(s, "{name}", name)
}

// SwitchDesktop switches to the given virtual desktop before launching.
// Does nothing if n <= 0 or if the DLL is unavailable.
func SwitchDesktop(n int) {
	if n > 0 && desktop.Available() {
		desktop.SwitchTo(n)
	}
}

// Detach starts a command and detaches from it (fire-and-forget).
// The goroutine calling cmd.Wait() is necessary to reap the child process
// and prevent zombie processes. We don't use the exit status.
func Detach(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "places-app: launch error: %v\n", err)
		}
	}()
	return nil
}
