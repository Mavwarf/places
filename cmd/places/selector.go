package main

import (
	"fmt"
	"io"
	"os"
)

const (
	keyUp     = 256
	keyDown   = 257
	keyEnter  = 258
	keyEscape = 259
)

type selectItem struct {
	Name    string
	Path    string
	Warning string // e.g. "[missing!]"
}

// readKey reads a single keystroke using platform-specific readKeyCode().
func readKey() int {
	return readKeyCode()
}

// interactiveSelect presents an arrow-key navigable menu on out (stderr).
// Returns the selected index and true, or -1 and false if cancelled.
func interactiveSelect(items []selectItem, out io.Writer) (int, bool) {
	cursor := 0
	maxNameLen := 0
	for _, item := range items {
		if len(item.Name) > maxNameLen {
			maxNameLen = len(item.Name)
		}
	}

	// Hide cursor.
	fmt.Fprint(out, "\x1b[?25l")

	render := func() {
		for i, item := range items {
			if i > 0 {
				// Move up is handled by clearing — we re-render from top each time.
			}
			if i == cursor {
				fmt.Fprintf(out, "\x1b[2K\r  \x1b[1m\x1b[32m> %-*s  \x1b[36m%s\x1b[0m", maxNameLen, item.Name, item.Path)
			} else {
				fmt.Fprintf(out, "\x1b[2K\r  \x1b[2m  %-*s  %s\x1b[0m", maxNameLen, item.Name, item.Path)
			}
			if item.Warning != "" {
				fmt.Fprintf(out, " \x1b[33m%s\x1b[0m", item.Warning)
			}
			fmt.Fprint(out, "\n")
		}
		fmt.Fprintf(out, "\x1b[2K\r  \x1b[2m↑/↓ navigate, Enter select, Esc cancel\x1b[0m")
	}

	// Move cursor up to re-render from top.
	moveToTop := func() {
		// Move up len(items) lines (items + hint line already printed).
		for i := 0; i < len(items); i++ {
			fmt.Fprint(out, "\x1b[1A")
		}
		fmt.Fprint(out, "\r")
	}

	render()

	for {
		key := readKey()
		switch key {
		case keyUp:
			if cursor > 0 {
				cursor--
			}
		case keyDown:
			if cursor < len(items)-1 {
				cursor++
			}
		case keyEnter:
			cleanup(out, len(items))
			return cursor, true
		case keyEscape, 'q':
			cleanup(out, len(items))
			return -1, false
		default:
			continue
		}
		moveToTop()
		render()
	}
}

// cleanup clears the menu lines and shows cursor again.
func cleanup(out io.Writer, itemCount int) {
	// Clear the hint line.
	fmt.Fprint(out, "\x1b[2K\r")
	// Move up and clear each item line.
	for i := 0; i < itemCount; i++ {
		fmt.Fprint(out, "\x1b[1A\x1b[2K")
	}
	fmt.Fprint(out, "\r")
	// Show cursor.
	fmt.Fprint(out, "\x1b[?25h")
}

// runInteractiveSelect sets up raw mode and runs the interactive selector.
// On failure to enable raw mode, it returns an error so the caller can fall back.
func runInteractiveSelect(items []selectItem) (int, bool, error) {
	restore, err := enableRawMode()
	if err != nil {
		return 0, false, err
	}
	defer restore()

	idx, ok := interactiveSelect(items, os.Stderr)
	return idx, ok, nil
}
