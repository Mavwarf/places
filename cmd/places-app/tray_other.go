//go:build !windows

package main

// runTray is a no-op on non-Windows. The energye/systray library conflicts
// with Wails' NSApplication run loop on macOS, causing SIGILL. The system tray
// is skipped; the dashboard window is the primary interface.
func runTray(app *App) {}

// quitTray is a no-op on non-Windows.
func quitTray() {}
