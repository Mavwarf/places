//go:build !windows

package launcher

import "os/exec"

// StartDetached starts a process detached from the parent.
func StartDetached(cmd *exec.Cmd) error {
	return cmd.Start()
}
