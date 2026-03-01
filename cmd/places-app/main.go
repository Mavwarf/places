package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Mavwarf/places/internal/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	port := 8822

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port", "-p":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &port)
				i++
			}
		}
	}

	go func() {
		if err := app.Serve(port); err != nil {
			fmt.Fprintf(os.Stderr, "places-app: %v\n", err)
			os.Exit(1)
		}
	}()

	if err := waitForServer(port, 3*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "places-app: %v\n", err)
		os.Exit(1)
	}

	a := &App{port: port}

	loader := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body style="background:#1a1b26"></body></html>`))
	})

	err := wails.Run(&options.App{
		Title:     "places dashboard",
		Width:     900,
		Height:    600,
		MinWidth:  700,
		MinHeight: 400,
		AssetServer: &assetserver.Options{
			Handler: loader,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 27, B: 38, A: 255},
		OnStartup:        a.startup,
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
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(addr)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server not ready after %s", timeout)
}
