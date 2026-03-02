package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/Mavwarf/places/internal/desktop"
)

// PowerShell opens a new PowerShell window at the given directory.
func PowerShell(path string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", "start", "", "powershell", "-NoExit", "-Command",
			fmt.Sprintf("Set-Location '%s'", path))
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
func Claude(path string) *exec.Cmd {
	return exec.Command("cmd", "/c", "start", "", "powershell", "-NoExit", "-Command",
		fmt.Sprintf("Set-Location '%s'; claude", path))
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
