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

// Windows modifier key bitmasks and virtual key codes for RegisterHotKey.
const (
	modAlt   = 0x0001
	modWin   = 0x0008
	vkP      = 0x50
	wmHotkey = 0x0312 // WM_HOTKEY message ID
	hotkeyID = 1      // unique ID for our RegisterHotKey call
)

var (
	registerHotKey = user32.NewProc("RegisterHotKey")
	getMessageW    = user32.NewProc("GetMessageW")
	findWindowW    = user32.NewProc("FindWindowW") // also used by topmost_windows.go
)

type point struct{ x, y int32 }

// msg mirrors the Windows MSG struct used by GetMessageW.
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
	// LockOSThread: RegisterHotKey and GetMessageW must run on the same OS
	// thread — Windows delivers hotkey messages to the thread that registered them.
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
		title, _ := syscall.UTF16PtrFromString(appTitle)
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
