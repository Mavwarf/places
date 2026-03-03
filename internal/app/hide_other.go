//go:build !windows

package app

import "os/exec"

func hideWindow(_ *exec.Cmd) {}
