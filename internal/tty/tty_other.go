//go:build !windows && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package tty

import "os"

type platformState struct{}

func isTerminal(*os.File) bool { return false }

func (t *Terminal) open() error             { return ErrNotTerminal }
func (t *Terminal) size() (int, int, error) { return 0, 0, ErrNotTerminal }
func (t *Terminal) quiet() error            { return nil }
func (t *Terminal) pollKey() (byte, bool)   { return 0, false }
func (t *Terminal) restore()                {}

func openConsole() (*os.File, *os.File, error) { return nil, nil, ErrNotTerminal }

func width(*os.File) int { return 0 }
