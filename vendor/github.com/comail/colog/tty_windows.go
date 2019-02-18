package colog

import (
	"syscall"
	"unsafe"
)

// Use variable indirection for test stubbing
var isTerminal = isTerminalFunc
var terminalWidth = terminalWidthFunc

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var procInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
var procMode = kernel32.NewProc("GetConsoleMode")

// Not applicable in windows
// define constant to avoid compilation error
const ioctlReadTermios = 0x0

// isTerminalFunc returns true if the given file descriptor is a terminal.
func isTerminalFunc(fd int) bool {
	var st uint32
	r, _, errno := syscall.Syscall(procMode.Addr(), 2, uintptr(fd), uintptr(unsafe.Pointer(&st)), 0)
	if errno != 0 {
		return false
	}

	return r != 0
}

type short int16
type word uint16

type coord struct {
	x short
	y short
}
type rectangle struct {
	left   short
	top    short
	right  short
	bottom short
}

type termInfo struct {
	size              coord
	cursorPosition    coord
	attributes        word
	window            rectangle
	maximumWindowSize coord
}

// terminalWidthFunc returns the width in characters of the terminal.
func terminalWidthFunc(fd int) (width int) {
	var info termInfo
	_, _, errno := syscall.Syscall(procInfo.Addr(), 2, uintptr(fd), uintptr(unsafe.Pointer(&info)), 0)
	if errno != 0 {
		return -1
	}

	return int(info.size.x)
}
