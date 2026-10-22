package tty

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// quietPTY returns a terminal on a new pseudo-terminal, quietened as during
// the show, and the pseudo-terminal's other end, which plays the keyboard.
func quietPTY(t *testing.T) (*Terminal, *os.File) {
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
	term := &Terminal{In: s, Out: s}
	if err := term.open(); err != nil {
		t.Fatal(err)
	}
	if err := term.Quiet(); err != nil {
		t.Fatal(err)
	}
	return term, m
}

// lineMode does what a shell does to the terminal while the show is stopped
// with Ctrl+Z: it puts back line-at-a-time input with echo.
func lineMode(t *testing.T, term *Terminal) {
	tio, err := unix.IoctlGetTermios(term.state.inFd, ioctlGetTermios)
	if err != nil {
		t.Fatal(err)
	}
	tio.Lflag |= unix.ICANON | unix.ECHO
	tio.Cc[unix.VMIN], tio.Cc[unix.VTIME] = 1, 0
	if err := unix.IoctlSetTermios(term.state.inFd, ioctlSetTermios, tio); err != nil {
		t.Fatal(err)
	}
}

func within(t *testing.T, what string, f func()) {
	done := make(chan struct{})
	go func() { f(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("%s still blocked after 1s", what)
	}
}

// After `fg`, the terminal may be back in line mode with half a line typed:
// checking for a key and restoring the terminal must still return at once.
func TestPollKeyNeverBlocks(t *testing.T) {
	term, kbd := quietPTY(t)
	lineMode(t, term)
	kbd.WriteString("ab")
	within(t, "PollKey", func() { term.PollKey() })
}

func TestRestoreNeverBlocks(t *testing.T) {
	term, kbd := quietPTY(t)
	lineMode(t, term)
	kbd.WriteString("ab")
	within(t, "Restore", term.Restore)
}

// Keys typed during the show are dropped, not handed to the agent.
func TestRestoreDropsTypedKeys(t *testing.T) {
	term, kbd := quietPTY(t)
	kbd.WriteString("typed during the show\n")
	time.Sleep(50 * time.Millisecond)
	term.Restore()
	n, err := unix.IoctlGetInt(term.state.inFd, unix.TIOCINQ)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("%d bytes typed during the show are still waiting for the agent", n)
	}
}
