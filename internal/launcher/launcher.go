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
// The tab title is set to "Claude Code - <name>". Uses Windows Terminal when
// available (--suppressApplicationTitle prevents Claude from overriding the title).
func Claude(path, name string) *exec.Cmd {
	title := "Claude Code - " + name
	psCmd := fmt.Sprintf("Set-Location '%s'; claude", psEscape(path))
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("wt.exe"); err == nil {
			// wt uses ; as command separator — escape with \;
			wtCmd := strings.ReplaceAll(psCmd, ";", "\\;")
			return exec.Command("wt", "new-tab", "--title", title,
				"--suppressApplicationTitle", "powershell", "-NoExit", "-Command", wtCmd)
		}
	}
	// Fallback: title may be overridden by Claude on startup.
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

// Detach starts a command and detaches from it (doesn't wait for exit).
func Detach(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
