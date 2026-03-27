//go:build !windows

package main

func setAlwaysOnTop(on bool) {}

func pinAllDesktops(pin bool) bool { return false }
