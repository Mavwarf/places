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

// PowerShell opens a new PowerShell window at the given directory.
func PowerShell(path string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", "start", "", "powershell", "-NoExit", "-Command",
			fmt.Sprintf("Set-Location '%s'", psEscape(path)))
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell)
	cmd.Dir = path
	return cmd
}

// Cmd opens a new cmd.exe window at the given directory (Windows only).
func Cmd(path string) *exec.Cmd {
	return exec.Command("cmd", "/c", "start", "", "cmd", "/k", "cd", "/d", path)
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
func Claude(path, name string) *exec.Cmd {
	title := "Claude Code - " + name
	psCmd := fmt.Sprintf("Set-Location '%s'; claude", psEscape(path))
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("wt.exe"); err == nil {
			wtCmd := strings.ReplaceAll(psCmd, ";", "\\;")
			return exec.Command("wt", "new-tab", "--title", title,
				"--suppressApplicationTitle", "powershell", "-NoExit", "-Command", wtCmd)
		}
	}
	return exec.Command("cmd", "/c", "start", title, "powershell", "-NoExit", "-Command", psCmd)
}

// Explorer opens the file explorer at the given directory.
func Explorer(path string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("explorer", path)
	}
	return exec.Command("xdg-open", path)
}

// Code opens VS Code at the given directory.
func Code(path string) *exec.Cmd {
	return exec.Command("code", path)
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
	go cmd.Wait()
	return nil
}
