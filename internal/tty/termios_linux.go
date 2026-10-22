package tty

import "golang.org/x/sys/unix"

const (
	ioctlGetTermios = unix.TCGETS
	ioctlSetTermios = unix.TCSETS
	// ioctlSetTermiosFlush sets the modes and discards unread input.
	ioctlSetTermiosFlush = unix.TCSETSF
)
