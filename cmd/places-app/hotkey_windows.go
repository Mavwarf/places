//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/Mavwarf/places/internal/desktop"
)

const (
	modAlt      = 0x0001
	modWin      = 0x0008
	vkP         = 0x50
	wmHotkey    = 0x0312
	hotkeyID    = 1
)

var (
	registerHotKey = user32.NewProc("RegisterHotKey")
	getMessageW    = user32.NewProc("GetMessageW")
	findWindowW    = user32.NewProc("FindWindowW")
)

type point struct{ x, y int32 }

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

// runHotkey registers Win+Alt+P as a global hotkey and listens for it.
// Must be called in a goroutine — blocks forever on the message loop.
func runHotkey(app *App) {
	runtime.LockOSThread()

	ret, _, err := registerHotKey.Call(0, hotkeyID, modAlt|modWin, vkP)
	if ret == 0 {
		fmt.Fprintf(os.Stderr, "places-app: hotkey: RegisterHotKey failed: %v\n", err)
		return
	}

	var m msg
	for {
		ret, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		if m.message == wmHotkey {
			onHotkeyPressed(app)
		}
	}
}

func onHotkeyPressed(app *App) {
	// Switch to the virtual desktop where the dashboard window lives.
	if desktop.Available() {
		title, _ := syscall.UTF16PtrFromString("places dashboard")
		hwnd, _, _ := findWindowW.Call(0, uintptr(unsafe.Pointer(title)))
		if hwnd != 0 {
			winDesk, err := desktop.WindowDesktop(hwnd)
			if err == nil && winDesk > 0 {
				desktop.SwitchTo(winDesk)
			}
		}
	}
	app.ShowWindow()
}
