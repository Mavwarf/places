//go:build !windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	ioctlGetTermios = 0x5401 // TCGETS
	ioctlSetTermios = 0x5402 // TCSETS
)

type termios struct {
	Iflag  uint32
	Oflag  uint32
	Cflag  uint32
	Lflag  uint32
	Cc     [20]byte
	Ispeed uint32
	Ospeed uint32
}

func enableRawMode() (restore func(), err error) {
	fd := int(os.Stdin.Fd())

	var orig termios
	if _, _, e := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd),
		uintptr(ioctlGetTermios), uintptr(unsafe.Pointer(&orig)),
		0, 0, 0); e != 0 {
		return nil, e
	}

	raw := orig
	raw.Lflag &^= 0x0002 | 0x0008 // ICANON | ECHO
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
func readKeyCode() int {
	buf := make([]byte, 1)
	if _, err := os.Stdin.Read(buf); err != nil {
		return -1
	}

	switch buf[0] {
	case '\r', '\n':
		return keyEnter
	case 0x1B:
		// Try to read escape sequence.
		if _, err := os.Stdin.Read(buf); err != nil {
			return keyEscape
		}
		if buf[0] != '[' {
			return keyEscape
		}
		if _, err := os.Stdin.Read(buf); err != nil {
			return keyEscape
		}
		switch buf[0] {
		case 'A':
			return keyUp
		case 'B':
			return keyDown
		}
		return keyEscape
	}
	return int(buf[0])
}
