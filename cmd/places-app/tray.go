package main

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"

	"github.com/Mavwarf/places/internal/config"
	"github.com/energye/systray"
)

//go:embed appicon.png
var trayIcon []byte

// runTray starts the system tray icon. Must be called in a goroutine —
// systray.Run blocks until Quit is called.
func runTray(app *App) {
	// Lock this goroutine to an OS thread so that the hidden window created
	// by systray and the GetMessage loop share the same thread.
	runtime.LockOSThread()
	systray.Run(func() { onTrayReady(app) }, func() {})
}

// pngToICO wraps raw PNG bytes in a minimal ICO container.
// Windows LoadImage(IMAGE_ICON) requires ICO format; since Vista,
// ICO supports embedded PNG data directly.
func pngToICO(png []byte) []byte {
	buf := new(bytes.Buffer)
	// ICONDIR header
	binary.Write(buf, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(buf, binary.LittleEndian, uint16(1)) // type: 1 = ICO
	binary.Write(buf, binary.LittleEndian, uint16(1)) // count: 1 image

	// ICONDIRENTRY
	buf.WriteByte(0)  // width (0 = 256)
	buf.WriteByte(0)  // height (0 = 256)
	buf.WriteByte(0)  // color count
	buf.WriteByte(0)  // reserved
	binary.Write(buf, binary.LittleEndian, uint16(1))          // color planes
	binary.Write(buf, binary.LittleEndian, uint16(32))         // bits per pixel
	binary.Write(buf, binary.LittleEndian, uint32(len(png)))   // image data size
	binary.Write(buf, binary.LittleEndian, uint32(6+1*16))     // offset to image data (header + 1 entry)

	// PNG data
	buf.Write(png)
	return buf.Bytes()
}

func onTrayReady(app *App) {
	systray.SetIcon(pngToICO(trayIcon))
	systray.SetTooltip("places")
	systray.SetOnClick(func(menu systray.IMenu) { menu.ShowMenu() })
	systray.SetOnDClick(func(menu systray.IMenu) { app.ShowWindow() })

	mDashboard := systray.AddMenuItem("Open Dashboard", "Show the places dashboard window")
	mDashboard.Click(func() { app.ShowWindow() })

	systray.AddSeparator()
	addPlaceMenus()
	systray.AddSeparator()

	mRefresh := systray.AddMenuItem("Refresh", "Reload places from config")
	mRefresh.Click(func() {
		systray.ResetMenu()
		onTrayReady(app)
	})

	mQuit := systray.AddMenuItem("Quit", "Exit places-app")
	mQuit.Click(func() {
		systray.Quit()
		os.Exit(0)
	})
}

func addPlaceMenus() {
	cfg, err := config.Load()
	if err != nil {
		item := systray.AddMenuItem("(failed to load places)", "")
		item.Disable()
		return
	}

	names := make([]string, 0, len(cfg.Places))
	for name := range cfg.Places {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		place := cfg.Places[name]
		path := place.Path
		parent := systray.AddMenuItem(name, path)

		mPS := parent.AddSubMenuItem("PowerShell", "Open PowerShell here")
		mPS.Click(func() { openTerminal(path, "powershell") })

		mClaude := parent.AddSubMenuItem("Claude", "Open PowerShell + Claude here")
		mClaude.Click(func() { openTerminal(path, "claude") })

		mCmd := parent.AddSubMenuItem("cmd", "Open cmd.exe here")
		mCmd.Click(func() { openTerminal(path, "cmd") })

		mExplorer := parent.AddSubMenuItem("Explorer", "Open Explorer here")
		mExplorer.Click(func() { openTerminal(path, "explorer") })
	}
}

func openTerminal(path, action string) {
	var cmd *exec.Cmd
	switch action {
	case "powershell":
		cmd = exec.Command("cmd", "/c", "start", "", "powershell", "-NoExit", "-Command",
			fmt.Sprintf("Set-Location '%s'", path))
	case "cmd":
		cmd = exec.Command("cmd", "/c", "start", "", "cmd", "/k",
			fmt.Sprintf("cd /d \"%s\"", path))
	case "claude":
		cmd = exec.Command("cmd", "/c", "start", "", "powershell", "-NoExit", "-Command",
			fmt.Sprintf("Set-Location '%s'; claude", path))
	case "explorer":
		cmd = exec.Command("explorer", path)
	}
	if cmd != nil {
		if err := cmd.Start(); err == nil {
			go cmd.Wait()
		}
	}
}
