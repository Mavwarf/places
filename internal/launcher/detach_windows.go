//go:build windows

package launcher

import (
	"os/exec"
	"syscall"
)

// StartDetached starts a process fully detached from the parent console.
// DETACH_PROCESS (0x08) tells CreateProcess to not inherit the parent's
// console — the child gets no console at all. This prevents the child from
// being killed when the parent terminal window closes, which is essential
// for launching places-app from the CLI.
func StartDetached(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008, // DETACH_PROCESS
	}
	return cmd.Start()
}
