package main

import (
	"fmt"
	"os"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"

	"github.com/ruskaruma/cuck-code/internal/config"
)

// openPTY returns the terminal end of a new pseudo-terminal.
func openPTY(t *testing.T) *os.File {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skip("no pseudo-terminals here:", err)
	}
	t.Cleanup(func() { m.Close() })
	if err := unix.IoctlSetPointerInt(int(m.Fd()), unix.TIOCSPTLCK, 0); err != nil {
		t.Skip("can't unlock the pseudo-terminal:", err)
	}
	n, err := unix.IoctlGetInt(int(m.Fd()), unix.TIOCGPTN)
	if err != nil {
		t.Skip("can't name the pseudo-terminal:", err)
	}
	s, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skip("can't open the pseudo-terminal:", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestIntroWhenStdinIsATerminal(t *testing.T) {
	setStdin(t, openPTY(t))
	t.Setenv("CUCK_NO_ANIMATION", "")
	t.Setenv("TERM", "xterm-256color")
	if !shouldAnimate(false, config.Config{}) {
		t.Error("shouldAnimate = false with stdin on a terminal, want true")
	}
}
