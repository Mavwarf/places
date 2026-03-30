package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Mavwarf/places/internal/app"
	"github.com/Mavwarf/places/internal/sessions"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// appTitle is the window title used by Wails and for FindWindowW lookups
// in topmost_windows.go and hotkey_windows.go.
const appTitle = "places dashboard"

// version and buildTime are set at build time via ldflags:
//
//	go build -ldflags "-X main.version=v0.3.8 -X 'main.buildTime=2026-03-05 09:30'"
var version = "dev"
var buildTime = ""

func main() {
	app.Version = version
	app.BuildTime = buildTime
	port := 8822
	if env := os.Getenv("PLACES_PORT"); env != "" {
		if p, err := strconv.Atoi(env); err == nil {
			port = p
		}
	}

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port", "-p":
			if i+1 < len(args) {
				if p, err := strconv.Atoi(args[i+1]); err == nil {
					port = p
				}
				i++
			}
		}
	}

	// If an instance is already running, ask it to show its window and exit.
	showURL := fmt.Sprintf("http://127.0.0.1:%d/api/show", port)
	resp, err := http.Post(showURL, "", nil)
	if err == nil {
		resp.Body.Close()
		os.Exit(0)
	}

	// Open session tracker for time tracking.
	tracker, err := sessions.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "places-app: warning: session tracking disabled: %v\n", err)
	}

	geom := loadGeometry()
	a := &App{port: port, ready: make(chan struct{}), geom: geom, tracker: tracker}

	// Start HTTP API server in a goroutine. The dashboard UI is served from
	// here (not Wails' asset server) so we get a plain HTTP page with full
	// control. Callbacks bridge HTTP endpoints to Wails window operations.
	go func() {
		if err := app.Serve(port, app.Callbacks{
			Show:     a.ShowWindow,
			Browse:     a.BrowseDir,
			BrowseFile: a.BrowseFile,
			Minimize: a.MinimizeWindow,
			Quit:     a.QuitApp,
			Topmost:        setAlwaysOnTop,
			PinAllDesktops:  pinAllDesktops,
			RunningSessions: func(names []string) []byte {
				rs := detectRunningSessions(names)
				// Update session tracker and enrich with timing.
				if tracker != nil {
					running := make([]struct{ Place, Action string }, len(rs))
					for i, s := range rs {
						running[i] = struct{ Place, Action string }{s.Name, s.Action}
					}
					tracker.Update(running)
					for i := range rs {
						if info := tracker.GetActiveInfo(rs[i].Name, rs[i].Action); info != nil {
							rs[i].Elapsed = info.Elapsed
							rs[i].Today = info.Today
						}
					}
				}
				data, _ := json.Marshal(rs)
				return data
			},
			SessionHistory: func(from, to int64) []byte {
				if tracker == nil {
					return []byte("[]")
				}
				data, _ := json.Marshal(tracker.QueryHistory(from, to))
				return data
			},
			LastDrop: a.LastDrop,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "places-app: %v\n", err)
			os.Exit(1)
		}
	}()

	if err := waitForServer(port, 3*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "places-app: %v\n", err)
		os.Exit(1)
	}

	go runTray(a)
	go runHotkey(a)

	// Wails requires an asset handler, but we redirect to the HTTP server in
	// startup(). This minimal loader just prevents a flash of white background.
	loader := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body style="background:#1a1b26"></body></html>`))
	})

	// BindingsAllowedOrigins: after startup() redirects the WebView to our
	// HTTP server, the page origin changes. Without this allowlist, Wails
	// blocks WebView2 IPC (postMessage) from the HTTP origin, which breaks
	// OnFileDrop and any other native Wails features.
	width, height := 1100, 600
	if geom != nil {
		width, height = geom.Width, geom.Height
	}

	origin := fmt.Sprintf("http://127.0.0.1:%d", port)
	err = wails.Run(&options.App{
		Title:             appTitle,
		Width:             width,
		Height:            height,
		MinWidth:          700,
		MinHeight:         400,
		AssetServer: &assetserver.Options{
			Handler: loader,
		},
		BackgroundColour:       &options.RGBA{R: 26, G: 27, B: 38, A: 255},
		BindingsAllowedOrigins: origin,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		OnStartup:        a.startup,
		OnBeforeClose:    a.beforeClose,
		OnShutdown:       a.shutdown,
		Bind:             []interface{}{a},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "places-app: %v\n", err)
		os.Exit(1)
	}
}

func waitForServer(port int, timeout time.Duration) error {
	addr := fmt.Sprintf("http://127.0.0.1:%d/", port)
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(addr)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server not ready after %s", timeout)
}
