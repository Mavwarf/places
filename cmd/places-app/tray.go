package main

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"os"
	"runtime"

	"github.com/Mavwarf/places/internal/config"
	"github.com/Mavwarf/places/internal/launcher"
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
// Windows LoadImage(IMAGE_ICON) requires ICO format, but since Vista, ICO
// supports embedded PNG data directly — no need to decode/re-encode the image.
//
// ICO file structure:
//   ICONDIR header (6 bytes): reserved=0, type=1(icon), count=1
//   ICONDIRENTRY (16 bytes): dimensions=0(use PNG header), planes=1, bpp=32,
//     data_size=len(png), data_offset=6+16=22
//   Raw PNG data follows immediately.
func pngToICO(png []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	// ICONDIR header: {reserved=0, type=1(icon), count=1}
	for _, v := range []interface{}{uint16(0), uint16(1), uint16(1)} {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			return nil, err
		}
	}

	// ICONDIRENTRY: width=0, height=0, colors=0, reserved=0 (0 means "read from PNG")
	buf.Write([]byte{0, 0, 0, 0})
	// planes=1, bpp=32, data_size, data_offset (header=6 + 1 entry*16 = 22)
	for _, v := range []interface{}{uint16(1), uint16(32), uint32(len(png)), uint32(6 + 1*16)} {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			return nil, err
		}
	}

	buf.Write(png)
	return buf.Bytes(), nil
}

func onTrayReady(app *App) {
	ico, err := pngToICO(trayIcon)
	if err != nil {
		systray.SetIcon(trayIcon) // fallback to raw PNG
	} else {
		systray.SetIcon(ico)
	}
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

	names := config.SortedNames(cfg)

	for _, name := range names {
		place := cfg.Places[name]
		path := place.Path
		desk := place.Desktop
		placeName := name
		parent := systray.AddMenuItem(name, path)

		mPS := parent.AddSubMenuItem("PowerShell", "Open PowerShell here")
		mPS.Click(func() { recordTrayUse(placeName); launcher.SwitchDesktop(desk); launcher.Detach(launcher.PowerShell(path)) })

		mClaude := parent.AddSubMenuItem("Claude", "Open PowerShell + Claude here")
		mClaude.Click(func() { recordTrayUse(placeName); launcher.SwitchDesktop(desk); launcher.Detach(launcher.Claude(path, placeName)) })

		mCmd := parent.AddSubMenuItem("cmd", "Open cmd.exe here")
		mCmd.Click(func() { recordTrayUse(placeName); launcher.SwitchDesktop(desk); launcher.Detach(launcher.Cmd(path)) })

		mExplorer := parent.AddSubMenuItem("Explorer", "Open Explorer here")
		mExplorer.Click(func() { recordTrayUse(placeName); launcher.SwitchDesktop(desk); launcher.Detach(launcher.Explorer(path)) })
	}
}

// recordTrayUse increments the use count for a place when launched from the tray.
// Errors are silently ignored — tray actions shouldn't fail visibly.
func recordTrayUse(name string) {
	config.Lock()
	defer config.Unlock()
	cfg, err := config.Load()
	if err != nil {
		return
	}
	place, ok := cfg.Places[name]
	if !ok {
		return
	}
	config.RecordUse(place)
	config.Save(cfg)
}
