package main

import (
	"context"
	"fmt"
	"os"

	"github.com/energye/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is a minimal struct for the Wails application lifecycle.
type App struct {
	ctx   context.Context
	port  int
	ready chan struct{} // closed when Wails startup completes
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	url := fmt.Sprintf("http://127.0.0.1:%d", a.port)
	wailsRuntime.WindowExecJS(ctx, fmt.Sprintf("window.location.href = '%s';", url))
	close(a.ready)
}

func (a *App) shutdown(ctx context.Context) {}

// beforeClose is called when the user clicks the window close button.
// Shift+close fully exits; normal close hides to tray.
func (a *App) beforeClose(ctx context.Context) bool {
	if isShiftHeld() {
		systray.Quit()
		os.Exit(0)
		return false
	}
	wailsRuntime.WindowHide(a.ctx)
	return true // prevent close → hide to tray
}

// ShowWindow makes the Wails window visible (used by the tray icon).
func (a *App) ShowWindow() {
	<-a.ready // wait for Wails to be initialized
	wailsRuntime.WindowShow(a.ctx)
}

// MinimizeWindow minimizes the Wails window.
func (a *App) MinimizeWindow() {
	<-a.ready
	wailsRuntime.WindowMinimise(a.ctx)
}

// QuitApp fully exits the application.
func (a *App) QuitApp() {
	systray.Quit()
	os.Exit(0)
}

// BrowseDir opens a native folder picker and returns the selected path.
func (a *App) BrowseDir() (string, error) {
	<-a.ready
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select directory",
	})
}
