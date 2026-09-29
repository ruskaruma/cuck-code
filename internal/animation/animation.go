// Package animation renders the Cuck Code intro: a fixed-camera ANSI scene
// seen from the chair. She is already in bed. Your coding agent walks in to
// do the work for you, looks at her, looks at you and winks; the wink
// triggers a glitch transition, after which the caller launches the agent.
package animation

import (
	"errors"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/ruskaruma/cuck-code/internal/tty"
)

// ErrInterrupted is returned when the user interrupts the animation (Ctrl+C).
var ErrInterrupted = errors.New("animation interrupted")

// ErrNotTerminal is returned when there is no interactive terminal to draw on.
var ErrNotTerminal = tty.ErrNotTerminal

const (
	// DefaultDuration is the total runtime of the intro, including the transition.
	DefaultDuration = 7500 * time.Millisecond
	MinDuration     = 3 * time.Second
	MaxDuration     = 30 * time.Second
	DefaultFPS      = 30

	minW, minH = 40, 14 // below this the room is illegible; show stage directions instead
)

// Options configures Play.
type Options struct {
	Duration  time.Duration
	FPS       int
	Color     ColorMode
	ColorAuto bool   // detect the colour mode from the environment instead of using Color
	Agent     string // the agent's name: he wears it on his name tag
	Tagline   string // shown under the logo during the transition, e.g. "-> claude"
}

const (
	enterScreen = "\x1b[?1049h\x1b[?25l\x1b[?7l\x1b[0m\x1b[2J\x1b[H"
	leaveScreen = "\x1b[0m\x1b[?7h\x1b[?25h\x1b[?1049l"
	// clearMain wipes the normal screen after a finished show, so the old
	// shell output doesn't flash back while the agent boots.
	clearMain = "\x1b[H\x1b[2J"
)

// Play runs the intro on the controlling terminal and always leaves the
// terminal as it found it, including on Ctrl+C, SIGTERM or a panic.
//
// Any key skips ahead to the eye contact; a second key (or q/Esc) ends the
// intro immediately.
func Play(o Options) (err error) {
	term, err := tty.Open()
	if err != nil {
		return err
	}
	mode := o.Color
	if o.ColorAuto {
		mode = DetectColorMode(os.Getenv, runtime.GOOS == "windows")
	}
	fps := o.FPS
	if fps <= 0 {
		fps = DefaultFPS
	}
	fps = min(max(fps, 5), 60)
	dur := o.Duration
	if dur <= 0 {
		dur = DefaultDuration
	}
	dur = min(max(dur, MinDuration), MaxDuration)
	scale := (dur.Seconds() - fxLength) / storyLen

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sig)

	var once sync.Once
	restore := func() {
		once.Do(func() {
			out := leaveScreen
			if err == nil {
				out += clearMain
			}
			_, _ = term.Out.WriteString(out)
			term.Restore()
		})
	}
	defer restore()
	defer func() {
		if r := recover(); r != nil {
			restore()
			panic(r)
		}
	}()

	if _, err := term.Out.WriteString(enterScreen); err != nil {
		return err
	}
	_ = term.Quiet()

	enc := &Encoder{Mode: mode}
	var cv, final *Canvas
	storyT, fxT := 0.0, 0.0
	ticker := time.NewTicker(time.Second / time.Duration(fps))
	defer ticker.Stop()
	last := time.Now()

	for frame := 0; ; frame++ {
		w, h, err := term.Size()
		if err != nil || w <= 0 || h <= 0 {
			w, h = 80, 24
		}
		if cv == nil || cv.W != w || cv.H != h {
			cv = NewCanvas(w, h)
			enc.Invalidate()
			final = nil
		}

		if storyT < storyLen {
			if w < minW || h < minH {
				renderTiny(cv, storyT, o.Agent)
			} else {
				st := StateAt(storyT, o.Agent)
				Render(cv, &st)
			}
		} else {
			if final == nil {
				// Freeze the wink frame as the source for the glitch.
				final = NewCanvas(w, h)
				if w < minW || h < minH {
					renderTiny(final, tWink, o.Agent)
				} else {
					st := StateAt(storyLen, o.Agent)
					Render(final, &st)
				}
			}
			if fxT >= fxLength {
				break
			}
			RenderFX(cv, final, fxT, frame, o.Tagline)
		}
		if _, err := term.Out.Write(enc.Frame(cv)); err != nil {
			return err
		}

		select {
		case <-sig:
			return ErrInterrupted
		case <-ticker.C:
		}
		now := time.Now()
		dt := now.Sub(last).Seconds()
		last = now
		if storyT < storyLen {
			storyT += dt / scale
		} else {
			fxT += dt
		}

		if key, ok := term.PollKey(); ok {
			switch {
			case key == 'q' || key == 0x1b || storyT >= skipTarget:
				return nil
			default:
				storyT = skipTarget
			}
		}
	}
	return nil
}
