//go:build windows

// Shell on Windows uses SysProcAttr.CmdLine to pass the command string
// directly to cmd.exe without Go's argument escaping. Go's exec.Command
// escapes embedded quotes with backslashes, but cmd.exe doesn't understand
// that convention — it would fail to find executables whose paths contain
// spaces or quotes.

package launcher

import (
	"os/exec"
	"syscall"
)

// Shell returns an exec.Cmd that runs a command string via cmd.exe.
func Shell(command string) *exec.Cmd {
	cmd := exec.Command("cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `/c ` + command}
	return cmd
}
