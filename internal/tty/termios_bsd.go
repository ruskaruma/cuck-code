//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package tty

import "golang.org/x/sys/unix"

const (
	ioctlGetTermios = unix.TIOCGETA
	ioctlSetTermios = unix.TIOCSETA
	// ioctlSetTermiosFlush sets the modes and discards unread input.
	ioctlSetTermiosFlush = unix.TIOCSETAF
)
