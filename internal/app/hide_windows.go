//go:build windows

package app

import (
	"os/exec"
	"syscall"
)

// hideWindow prevents an exec.Cmd from opening a visible console window.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
