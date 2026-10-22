//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package tty

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

var stopSignals = []os.Signal{syscall.SIGTSTP}

type platformState struct {
	inFd      int
	saved     *unix.Termios
	quietened bool
}

func isTerminal(f *os.File) bool {
	_, err := unix.IoctlGetTermios(int(f.Fd()), ioctlGetTermios)
	return err == nil
}

func (t *Terminal) open() error {
	if !isTerminal(t.Out) {
		return ErrNotTerminal
	}
	t.state.inFd = int(t.In.Fd())
	return nil
}

func (t *Terminal) size() (int, int, error) {
	ws, err := unix.IoctlGetWinsize(int(t.Out.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}

func (t *Terminal) quiet() error {
	saved, err := unix.IoctlGetTermios(t.state.inFd, ioctlGetTermios)
	if err != nil {
		return nil // stdin is not a terminal (e.g. piped); nothing to silence
	}
	raw := *saved
	// Keep ISIG so Ctrl+C still arrives as SIGINT; drop echo and line
	// buffering, and make reads return immediately when nothing is queued.
	raw.Lflag &^= unix.ECHO | unix.ICANON
	raw.Cc[unix.VMIN] = 0
	raw.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(t.state.inFd, ioctlSetTermios, &raw); err != nil {
		return err
	}
	t.state.saved = saved
	t.state.quietened = true
	return nil
}

func (t *Terminal) pollKey() (byte, bool) {
	if !t.state.quietened {
		return 0, false
	}
	// Check first: if something (a shell, after Ctrl+Z and fg) put the
	// terminal back in line mode, a plain read would wait for Enter.
	fds := []unix.PollFd{{Fd: int32(t.state.inFd), Events: unix.POLLIN}}
	if n, err := unix.Poll(fds, 0); err != nil || n <= 0 {
		return 0, false
	}
	var buf [16]byte
	n, err := unix.Read(t.state.inFd, buf[:])
	if err != nil || n <= 0 {
		return 0, false
	}
	return buf[0], true
}

func (t *Terminal) restore() {
	if !t.state.quietened {
		return
	}
	// Discard anything typed during the show along with setting the modes
	// back; unlike reading it away, this can't wait for a whole line.
	_ = unix.IoctlSetTermios(t.state.inFd, ioctlSetTermiosFlush, t.state.saved)
	t.state.quietened = false
}

func (t *Terminal) suspend() {
	quiet := t.state.quietened
	if quiet {
		_ = unix.IoctlSetTermios(t.state.inFd, ioctlSetTermios, t.state.saved)
		t.state.quietened = false
	}
	// Stop as Ctrl+Z would have; this returns once the shell continues us
	// (fg). SIGSTOP, because Go keeps handling SIGTSTP itself (dropping it)
	// even after signal.Reset.
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGSTOP)
	if quiet {
		_ = t.quiet()
	}
}

func openConsole() (*os.File, *os.File, error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	return f, f, nil
}

func width(f *os.File) int {
	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0
	}
	return int(ws.Col)
}
