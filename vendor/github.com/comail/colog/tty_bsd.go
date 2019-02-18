// +build darwin dragonfly freebsd netbsd openbsd

package colog

import "syscall"

const ioctlReadTermios = syscall.TIOCGETA
