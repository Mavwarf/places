// selector.go — Interactive arrow-key menu for picking a place.
// Platform-independent: uses readKeyCode() from term_windows.go or term_unix.go.
// All output goes to stderr (os.Stderr) so stdout stays clean for the shell wrapper.

package main

import (
	"fmt"
	"io"
	"os"
)

// Key constants (256+) are above the ASCII range so they can't collide with
// printable characters. Both term_windows.go and term_unix.go map their
// platform-specific key events to these values.
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

	// ANSI: hide cursor to prevent flickering during re-renders.
	// \x1b[?25l = DECTCEM (DEC text cursor enable mode) — hide cursor.
	fmt.Fprint(out, "\x1b[?25l")

	render := func() {
		for i, item := range items {
			if i > 0 {
				// Move up is handled by clearing — we re-render from top each time.
			}
			// \x1b[2K = erase entire line, \r = carriage return to column 0.
			// \x1b[1m = bold, \x1b[32m = green, \x1b[36m = cyan, \x1b[2m = dim.
			// \x1b[0m = reset all attributes.
			if i == cursor {
				fmt.Fprintf(out, "\x1b[2K\r  \x1b[1m\x1b[32m> %-*s  \x1b[36m%s\x1b[0m", maxNameLen, item.Name, item.Path)
			} else {
				fmt.Fprintf(out, "\x1b[2K\r  \x1b[2m  %-*s  %s\x1b[0m", maxNameLen, item.Name, item.Path)
			}
			if item.Warning != "" {
				fmt.Fprintf(out, " \x1b[33m%s\x1b[0m", item.Warning) // \x1b[33m = yellow
			}
			fmt.Fprint(out, "\n")
		}
		fmt.Fprintf(out, "\x1b[2K\r  \x1b[2m↑/↓ navigate, Enter select, Esc cancel\x1b[0m")
	}

	// Move cursor up to re-render from top.
	moveToTop := func() {
		// \x1b[1A = cursor up one line. We move up len(items) lines
		// (the item lines; the hint line is on the current line).
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

// cleanup clears the menu lines and restores the cursor. This erases all
// visual output so the terminal looks clean after selection or cancellation.
func cleanup(out io.Writer, itemCount int) {
	fmt.Fprint(out, "\x1b[2K\r")           // clear hint line
	for i := 0; i < itemCount; i++ {
		fmt.Fprint(out, "\x1b[1A\x1b[2K")  // move up + clear each item line
	}
	fmt.Fprint(out, "\r")
	fmt.Fprint(out, "\x1b[?25h")           // \x1b[?25h = show cursor (DECTCEM)
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
