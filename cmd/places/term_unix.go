//go:build !windows

// Raw terminal input for Unix using termios ioctl.
// On Unix, os.Stdin.Read() works in raw mode, and arrow keys arrive as
// VT100 escape sequences (ESC [ A/B/C/D) that we parse byte-by-byte.
// See term_windows.go for the Windows equivalent using ReadConsoleInputW.

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// termios ioctl request codes (Linux values; also work on most BSDs).
const (
	ioctlGetTermios = 0x5401 // TCGETS — read current terminal attributes
	ioctlSetTermios = 0x5402 // TCSETS — write terminal attributes immediately
)

// termios mirrors the C struct termios used by the kernel to control
// terminal behavior. We only modify Lflag (local mode flags).
type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32   // local flags: ICANON, ECHO, etc.
	Cc     [20]byte // control characters (VMIN, VTIME, etc.)
	Ispeed uint32
	Ospeed uint32
}

// enableRawMode disables canonical (line-buffered) mode and echo so we receive
// individual keypresses without waiting for Enter. Returns a restore function
// that re-applies the original terminal attributes.
func enableRawMode() (restore func(), err error) {
	fd := int(os.Stdin.Fd())

	var orig termios
	if _, _, e := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
		uintptr(ioctlGetTermios), uintptr(unsafe.Pointer(&orig)),
		0, 0, 0); e != 0 {
		return nil, e
	}

	raw := orig
	// Clear ICANON (0x0002): deliver input byte-by-byte instead of line-by-line.
	// Clear ECHO (0x0008): don't echo typed characters back to the terminal.
	raw.Lflag &^= 0x0002 | 0x0008
	if _, _, e := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
		uintptr(ioctlSetTermios), uintptr(unsafe.Pointer(&raw)),
		0, 0, 0); e != 0 {
		return nil, e
	}

	return func() {
		syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
			uintptr(ioctlSetTermios), uintptr(unsafe.Pointer(&orig)),
			0, 0, 0)
	}, nil
}

// readKeyCode reads one key press, parsing VT100 escape sequences,
// and returns a key constant (keyUp, keyDown, keyEnter, keyEscape) or ASCII value.
//
// Arrow keys arrive as 3-byte escape sequences: ESC [ A (up), ESC [ B (down).
// A bare ESC (0x1B) with no following '[' is treated as the Escape key.
func readKeyCode() int {
	buf := make([]byte, 1)
	if _, err := os.Stdin.Read(buf); err != nil {
		return -1
	}

	switch buf[0] {
	case '\r', '\n':
		return keyEnter
	case 0x1B: // ESC byte — could be standalone Escape or start of a sequence.
		if _, err := os.Stdin.Read(buf); err != nil {
			return keyEscape
		}
		if buf[0] != '[' {
			return keyEscape // bare ESC, not a CSI sequence
		}
		if _, err := os.Stdin.Read(buf); err != nil {
			return keyEscape
		}
		switch buf[0] {
		case 'A':
			return keyUp // ESC [ A
		case 'B':
			return keyDown // ESC [ B
		}
		return keyEscape
	}
	return int(buf[0])
}
