// Package tty is the small amount of platform-specific terminal plumbing the
// animation needs: detecting a real terminal, reading its size, silencing
// keyboard echo while the scene plays, polling for a skip key, and putting
// everything back exactly as it was found.
package tty

import (
	"errors"
	"os"
	"sync"
)

// ErrNotTerminal is returned by Open when stdout is not an interactive terminal.
var ErrNotTerminal = errors.New("stdout is not a terminal")

// Terminal wraps the process's stdin/stdout console.
type Terminal struct {
	Out *os.File
	In  *os.File

	restoreOnce sync.Once
	state       platformState
}

// Open prepares the controlling terminal for full-screen drawing. On Windows it
// also switches the console into VT (ANSI escape) mode.
func Open() (*Terminal, error) {
	t := &Terminal{Out: os.Stdout, In: os.Stdin}
	if err := t.open(); err != nil {
		return nil, err
	}
	return t, nil
}

// Size reports the visible terminal size in cells.
func (t *Terminal) Size() (w, h int, err error) { return t.size() }

// Quiet stops typed keys from echoing over the scene and makes PollKey
// non-blocking. Ctrl+C still raises an interrupt.
func (t *Terminal) Quiet() error { return t.quiet() }

// PollKey reports whether a key was pressed since the last call. It never blocks.
func (t *Terminal) PollKey() (byte, bool) { return t.pollKey() }

// Restore undoes Quiet and any console mode changes, discarding keystrokes
// typed during the animation so they do not leak into the launched agent.
// It is safe to call more than once.
func (t *Terminal) Restore() {
	t.restoreOnce.Do(t.restore)
}

// IsTerminal reports whether f is attached to an interactive terminal.
func IsTerminal(f *os.File) bool { return isTerminal(f) }

// OpenConsole opens the controlling terminal directly, bypassing stdin and
// stdout. It lets `cuck setup` ask its question even when started by a
// package manager that has redirected the standard streams.
func OpenConsole() (in, out *os.File, err error) { return openConsole() }

// Width reports the width in columns of the terminal f is attached to, or 0
// if f is not a terminal.
func Width(f *os.File) int { return width(f) }
