//go:build windows

package launcher

import (
	"os/exec"
	"syscall"
)

// StartDetached starts a process fully detached from the parent console.
// The child will not be killed when the parent terminal closes.
func StartDetached(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008, // DETACH_PROCESS
	}
	return cmd.Start()
}
