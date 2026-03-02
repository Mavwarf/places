//go:build windows

package main

import "syscall"

// user32 is loaded once here; other *_windows.go files in this package
// add their own procs (findWindowW, setWindowPos, registerHotKey, etc.).
var user32 = syscall.NewLazyDLL("user32.dll")
var getAsyncKeyState = user32.NewProc("GetAsyncKeyState")

// isShiftHeld returns true if Shift is currently held down.
// Used by beforeClose to distinguish "hide to tray" from "fully exit".
// GetAsyncKeyState returns a short with bit 15 set if the key is pressed.
func isShiftHeld() bool {
	ret, _, _ := getAsyncKeyState.Call(0x10) // VK_SHIFT
	return ret&0x8000 != 0
}
