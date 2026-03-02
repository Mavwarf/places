//go:build !windows

package launcher

import "os/exec"

// Shell returns an exec.Cmd that runs a command string via sh -c.
func Shell(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}
