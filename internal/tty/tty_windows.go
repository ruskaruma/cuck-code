//go:build windows

package tty

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type platformState struct {
	out, in     windows.Handle
	outMode     uint32
	inMode      uint32
	outChanged  bool
	inChanged   bool
	inIsConsole bool
}

var procReadConsoleInputW = windows.NewLazySystemDLL("kernel32.dll").NewProc("ReadConsoleInputW")

// inputRecord mirrors INPUT_RECORD; only the key-down flag of KEY_EVENT_RECORD is read.
type inputRecord struct {
	EventType uint16
	_         uint16
	KeyDown   int32
	_         [12]byte
}

func isTerminal(f *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(f.Fd()), &mode) == nil
}

func (t *Terminal) open() error {
	s := &t.state
	s.out = windows.Handle(t.Out.Fd())
	if err := windows.GetConsoleMode(s.out, &s.outMode); err != nil {
		return ErrNotTerminal
	}
	want := s.outMode | windows.ENABLE_PROCESSED_OUTPUT | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if want != s.outMode {
		// Fails on pre-Windows 10 consoles, which cannot render the scene.
		if err := windows.SetConsoleMode(s.out, want); err != nil {
			return err
		}
		s.outChanged = true
	}
	s.in = windows.Handle(t.In.Fd())
	s.inIsConsole = windows.GetConsoleMode(s.in, &s.inMode) == nil
	return nil
}

func (t *Terminal) size() (int, int, error) {
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(t.state.out, &info); err != nil {
		return 0, 0, err
	}
	w := int(info.Window.Right-info.Window.Left) + 1
	h := int(info.Window.Bottom-info.Window.Top) + 1
	return w, h, nil
}

func (t *Terminal) quiet() error {
	s := &t.state
	if !s.inIsConsole {
		return nil
	}
	// Leave ENABLE_PROCESSED_INPUT on so Ctrl+C still interrupts.
	mode := s.inMode &^ (windows.ENABLE_ECHO_INPUT | windows.ENABLE_LINE_INPUT)
	if err := windows.SetConsoleMode(s.in, mode); err != nil {
		return err
	}
	s.inChanged = true
	return nil
}

func (t *Terminal) pollKey() (byte, bool) {
	s := &t.state
	if !s.inIsConsole {
		return 0, false
	}
	for {
		var n uint32
		if err := windows.GetNumberOfConsoleInputEvents(s.in, &n); err != nil || n == 0 {
			return 0, false
		}
		var rec inputRecord
		var read uint32
		r, _, _ := procReadConsoleInputW.Call(uintptr(s.in), uintptr(unsafe.Pointer(&rec)), 1, uintptr(unsafe.Pointer(&read)))
		if r == 0 || read == 0 {
			return 0, false
		}
		if rec.EventType == windows.KEY_EVENT && rec.KeyDown != 0 {
			return ' ', true
		}
		// Focus, mouse and key-up events are not "a key press"; keep looking.
	}
}

func (t *Terminal) restore() {
	s := &t.state
	if s.inIsConsole {
		_ = windows.FlushConsoleInputBuffer(s.in)
	}
	if s.inChanged {
		_ = windows.SetConsoleMode(s.in, s.inMode)
	}
	if s.outChanged {
		_ = windows.SetConsoleMode(s.out, s.outMode)
	}
}

func openConsole() (*os.File, *os.File, error) {
	in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	out, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		in.Close()
		return nil, nil, err
	}
	return in, out, nil
}

func width(f *os.File) int {
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(windows.Handle(f.Fd()), &info); err != nil {
		return 0
	}
	return int(info.Window.Right-info.Window.Left) + 1
}
