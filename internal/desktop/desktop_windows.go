//go:build windows

// Virtual desktop control via VirtualDesktopAccessor.dll (third-party DLL).
// https://github.com/Ciantic/VirtualDesktopAccessor
//
// The DLL wraps undocumented Windows COM interfaces for virtual desktop management.
// It must be placed next to the executable. If the DLL is missing, all functions
// degrade gracefully (Available() returns false, actions become no-ops).
//
// IMPORTANT: All DLL functions use 0-indexed desktop numbers internally,
// but this package exposes a 1-indexed API (desktop 1 = first desktop) to
// match user expectations. The conversion happens at each call site.

package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

var dll *syscall.LazyDLL

func init() {
	// Look for DLL next to the running executable (not on PATH).
	exe, err := os.Executable()
	if err == nil {
		dll = syscall.NewLazyDLL(filepath.Join(filepath.Dir(exe), "VirtualDesktopAccessor.dll"))
	}
}

// Available reports whether the VirtualDesktopAccessor DLL is loadable.
func Available() bool {
	if dll == nil {
		return false
	}
	return dll.Load() == nil
}

// SwitchTo switches to the given virtual desktop (1-indexed).
func SwitchTo(n int) error {
	if dll == nil {
		return fmt.Errorf("VirtualDesktopAccessor.dll not found")
	}
	proc := dll.NewProc("GoToDesktopNumber")
	if err := proc.Find(); err != nil {
		return fmt.Errorf("GoToDesktopNumber not found in DLL: %w", err)
	}
	// DLL is 0-indexed; callers use 1-indexed.
	ret, _, _ := proc.Call(uintptr(n - 1))
	if ret != 0 {
		return fmt.Errorf("GoToDesktopNumber(%d) returned %d", n-1, ret)
	}
	return nil
}

// Current returns the current virtual desktop number (1-indexed).
func Current() (int, error) {
	if dll == nil {
		return 0, fmt.Errorf("VirtualDesktopAccessor.dll not found")
	}
	proc := dll.NewProc("GetCurrentDesktopNumber")
	if err := proc.Find(); err != nil {
		return 0, fmt.Errorf("GetCurrentDesktopNumber not found in DLL: %w", err)
	}
	ret, _, _ := proc.Call()
	return int(ret) + 1, nil // convert 0-indexed to 1-indexed
}

// HideConsole detaches this process from its console window using FreeConsole().
// This is called early in places-app startup so that when the GUI app launches
// from a terminal, Windows doesn't refocus the console window after launch.
// Without this, clicking the tray icon would flash the parent console.
func HideConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("FreeConsole")
	proc.Call()
}

// MoveWindowToDesktop moves a window to the given virtual desktop (1-indexed).
func MoveWindowToDesktop(hwnd uintptr, n int) error {
	if dll == nil {
		return fmt.Errorf("VirtualDesktopAccessor.dll not found")
	}
	proc := dll.NewProc("MoveWindowToDesktopNumber")
	if err := proc.Find(); err != nil {
		return fmt.Errorf("MoveWindowToDesktopNumber not found in DLL: %w", err)
	}
	// DLL is 0-indexed; callers use 1-indexed.
	proc.Call(hwnd, uintptr(n-1))
	return nil
}

// WindowDesktop returns the virtual desktop number (1-indexed) for the given window.
// Returns -1 if the window's desktop cannot be determined (e.g. hidden to tray,
// or the window has been moved to "all desktops"). The int32 cast is needed
// because the DLL returns -1 as a uint32 (0xFFFFFFFF), which Go would interpret
// as a large positive number without the signed cast.
func WindowDesktop(hwnd uintptr) (int, error) {
	if dll == nil {
		return -1, fmt.Errorf("VirtualDesktopAccessor.dll not found")
	}
	proc := dll.NewProc("GetWindowDesktopNumber")
	if err := proc.Find(); err != nil {
		return -1, fmt.Errorf("GetWindowDesktopNumber not found in DLL: %w", err)
	}
	ret, _, _ := proc.Call(hwnd)
	n := int(int32(ret)) // signed cast: DLL returns -1 for unknown windows
	if n < 0 {
		return -1, nil
	}
	return n + 1, nil // convert 0-indexed to 1-indexed
}

// Count returns the number of virtual desktops.
func Count() (int, error) {
	if dll == nil {
		return 0, fmt.Errorf("VirtualDesktopAccessor.dll not found")
	}
	proc := dll.NewProc("GetDesktopCount")
	if err := proc.Find(); err != nil {
		return 0, fmt.Errorf("GetDesktopCount not found in DLL: %w", err)
	}
	ret, _, _ := proc.Call()
	return int(ret), nil
}
