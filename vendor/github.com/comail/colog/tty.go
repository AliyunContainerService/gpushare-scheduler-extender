// +build !windows

package colog

import (
	"syscall"
	"unsafe"
)

// Use variable indirection for test stubbing
var isTerminal = isTerminalFunc
var terminalWidth = terminalWidthFunc

// isTerminalFunc returns true if the given file descriptor is a terminal.
func isTerminalFunc(fd int) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}

// terminalWidthFunc returns the width in characters of the terminal.
func terminalWidthFunc(fd int) (width int) {
	var dimensions [4]uint16

	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&dimensions)), 0, 0, 0)
	if errno != 0 {
		return -1
	}

	return int(dimensions[1])
}
