package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/energye/systray"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is a minimal struct for the Wails application lifecycle.
type App struct {
	ctx      context.Context
	port     int
	ready    chan struct{} // closed when Wails startup completes
	geom     *WindowGeometry
	lastDrop string
	dropMu   sync.Mutex
}

// startup redirects the WebView to our HTTP server, bypassing Wails' built-in
// asset serving. This lets us serve a plain HTML page with vanilla JS (no Wails
// runtime needed). OnFileDrop is registered before the redirect because the
// native WebView2 drop handler + IPC still works after navigation.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	wailsRuntime.OnFileDrop(ctx, func(x, y int, paths []string) {
		if len(paths) > 0 {
			a.dropMu.Lock()
			a.lastDrop = paths[0]
			a.dropMu.Unlock()
		}
	})
	if a.geom != nil {
		wailsRuntime.WindowSetPosition(ctx, a.geom.X, a.geom.Y)
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", a.port)
	wailsRuntime.WindowExecJS(ctx, fmt.Sprintf("window.location.href = '%s';", url))
	close(a.ready)
}

func (a *App) shutdown(ctx context.Context) {}

// saveWindowGeometry persists the current window position and size.
func (a *App) saveWindowGeometry() {
	x, y := wailsRuntime.WindowGetPosition(a.ctx)
	w, h := wailsRuntime.WindowGetSize(a.ctx)
	saveGeometry(WindowGeometry{X: x, Y: y, Width: w, Height: h})
}

// beforeClose is called when the user clicks the window close button.
// Always fully exits the application.
func (a *App) beforeClose(ctx context.Context) bool {
	<-a.ready
	a.saveWindowGeometry()
	systray.Quit()
	os.Exit(0)
	return false
}

// ShowWindow makes the Wails window visible (used by the tray icon and /api/show).
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
	a.saveWindowGeometry()
	systray.Quit()
	os.Exit(0)
}

// LastDrop returns and clears the last file path received via drag-and-drop.
func (a *App) LastDrop() string {
	a.dropMu.Lock()
	defer a.dropMu.Unlock()
	p := a.lastDrop
	a.lastDrop = ""
	return p
}

// BrowseDir opens a native folder picker and returns the selected path.
func (a *App) BrowseDir() (string, error) {
	<-a.ready
	return wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select directory",
	})
}

// BrowseFile opens a native file picker and returns the selected path.
func (a *App) BrowseFile() (string, error) {
	<-a.ready
	return wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select executable",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Executables (*.exe)", Pattern: "*.exe"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}
