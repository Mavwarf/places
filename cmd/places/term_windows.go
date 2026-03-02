//go:build windows

// Raw terminal input for Windows using kernel32 Console API.
// os.Stdin.Read() does NOT work for arrow keys on Windows — it only returns
// after Enter is pressed (line mode). We must use ReadConsoleInputW which
// delivers individual KEY_EVENT records with virtual key codes.

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode   = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode   = kernel32.NewProc("SetConsoleMode")
	procGetStdHandle     = kernel32.NewProc("GetStdHandle")
	procReadConsoleInput = kernel32.NewProc("ReadConsoleInputW")
)

// Console mode flags and standard handles.
const (
	stdInputHandle = uintptr(0xFFFFFFF6) // STD_INPUT_HANDLE (-10)
	stdErrorHandle = uintptr(0xFFFFFFF4) // STD_ERROR_HANDLE (-12)

	enableProcessedInput            = 0x0001 // handle Ctrl+C
	enableLineInput                 = 0x0002 // buffer input until Enter
	enableEchoInput                 = 0x0004 // echo typed characters
	enableVirtualTerminalProcessing = 0x0004 // for output handle: enable ANSI escapes

	keyEventFlag = 0x0001 // INPUT_RECORD.EventType for key events

	vkReturn = 0x0D // VK_RETURN
	vkEscape = 0x1B // VK_ESCAPE
	vkUp     = 0x26 // VK_UP
	vkDown   = 0x28 // VK_DOWN
)

// inputRecord mirrors the Windows INPUT_RECORD struct.
// Event is a union — we cast it to keyEventRecord when EventType == keyEventFlag.
type inputRecord struct {
	EventType uint16
	_         uint16
	Event     [16]byte
}

// keyEventRecord mirrors KEY_EVENT_RECORD.
type keyEventRecord struct {
	KeyDown         int32
	RepeatCount     uint16
	VirtualKeyCode  uint16
	VirtualScanCode uint16
	Char            uint16
	ControlKeyState uint32
}

var stdinHandle uintptr

// enableRawMode disables line buffering and echo on stdin so we receive
// individual keypresses. Also enables ANSI escape processing on stderr
// for the interactive menu renderer. Returns a restore function.
func enableRawMode() (restore func(), err error) {
	h, _, e := procGetStdHandle.Call(stdInputHandle)
	if h == 0 || h == ^uintptr(0) {
		return nil, e
	}
	stdinHandle = h

	var orig uint32
	r, _, e := procGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&orig)))
	if r == 0 {
		return nil, e
	}

	// Disable line buffering (ENABLE_LINE_INPUT) and echo (ENABLE_ECHO_INPUT).
	// Keep ENABLE_PROCESSED_INPUT so Ctrl+C still works.
	raw := orig &^ (enableLineInput | enableEchoInput)
	raw |= enableProcessedInput
	r, _, e = procSetConsoleMode.Call(h, uintptr(raw))
	if r == 0 {
		return nil, e
	}

	// Enable VT processing on stderr so ANSI escape output works.
	hErr, _, _ := procGetStdHandle.Call(stdErrorHandle)
	if hErr != 0 && hErr != ^uintptr(0) {
		var errMode uint32
		procGetConsoleMode.Call(hErr, uintptr(unsafe.Pointer(&errMode)))
		procSetConsoleMode.Call(hErr, uintptr(errMode|enableVirtualTerminalProcessing))
	}

	return func() {
		procSetConsoleMode.Call(h, uintptr(orig))
	}, nil
}

// readKeyCode reads one key press using ReadConsoleInput and returns
// a key constant (keyUp, keyDown, keyEnter, keyEscape) or the ASCII value.
func readKeyCode() int {
	for {
		var rec inputRecord
		var numRead uint32
		r, _, _ := procReadConsoleInput.Call(
			stdinHandle,
			uintptr(unsafe.Pointer(&rec)),
			1,
			uintptr(unsafe.Pointer(&numRead)),
		)
		if r == 0 {
			return -1
		}
		if rec.EventType != keyEventFlag {
			continue
		}
		ke := (*keyEventRecord)(unsafe.Pointer(&rec.Event))
		if ke.KeyDown == 0 {
			continue
		}
		switch ke.VirtualKeyCode {
		case vkUp:
			return keyUp
		case vkDown:
			return keyDown
		case vkReturn:
			return keyEnter
		case vkEscape:
			return keyEscape
		default:
			if ke.Char != 0 {
				return int(ke.Char)
			}
		}
	}
}
