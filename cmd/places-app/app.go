package main

import (
	"context"
	"fmt"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is a minimal struct for the Wails application lifecycle.
type App struct {
	ctx  context.Context
	port int
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	url := fmt.Sprintf("http://127.0.0.1:%d", a.port)
	wailsRuntime.WindowExecJS(ctx, fmt.Sprintf("window.location.href = '%s';", url))
}

func (a *App) shutdown(ctx context.Context) {}

// ShowWindow makes the Wails window visible (used by the tray icon).
func (a *App) ShowWindow() {
	wailsRuntime.WindowShow(a.ctx)
}
