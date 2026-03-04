//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var setWindowPos = user32.NewProc("SetWindowPos")

// SetWindowPos z-order constants and flags.
// HWND_TOPMOST (-1) places the window above all non-topmost windows.
// HWND_NOTOPMOST (-2) restores normal z-order.
// SWP flags tell SetWindowPos to only change z-order, leaving position/size/focus alone.
const (
	hwndTopmost   = ^uintptr(0) // -1
	hwndNotopmost = ^uintptr(1) // -2
	swpNomove     = 0x0002
	swpNosize     = 0x0001
	swpNoactivate = 0x0010
)

// setAlwaysOnTop finds the Wails window by title and toggles its z-order.
func setAlwaysOnTop(on bool) {
	title, _ := syscall.UTF16PtrFromString(appTitle)
	hwnd, _, _ := findWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return
	}
	insertAfter := hwndNotopmost
	if on {
		insertAfter = hwndTopmost
	}
	setWindowPos.Call(hwnd, insertAfter, 0, 0, 0, 0, swpNomove|swpNosize|swpNoactivate)
}
