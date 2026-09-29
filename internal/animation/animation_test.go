package animation

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestRenderAllSizes(t *testing.T) {
	sizes := [][2]int{{1, 1}, {20, 8}, {40, 14}, {80, 24}, {120, 40}, {300, 90}, {200, 20}, {50, 60}}
	for _, sz := range sizes {
		cv := NewCanvas(sz[0], sz[1])
		for ts := 0.0; ts <= storyLen; ts += 0.1 {
			st := StateAt(ts, "claude")
			Render(cv, &st)
		}
		out := NewCanvas(sz[0], sz[1])
		for ts := 0.0; ts <= fxLength; ts += 0.02 {
			RenderFX(out, cv, ts, int(ts*100), "-> claude")
		}
		renderTiny(cv, 3, "codex")
	}
}

func TestOutputIsASCII(t *testing.T) {
	cv := NewCanvas(100, 30)
	for _, ts := range []float64{0.2, 1.5, 3, 5.4, 7.6} {
		st := StateAt(ts, "claude")
		Render(cv, &st)
		for _, mode := range []ColorMode{ModeNone, Mode16, Mode256, ModeTrueColor} {
			for i, c := range cv.Cells {
				if g := c.Glyph(mode); g < 0x20 || g > 0x7e {
					t.Fatalf("t=%v mode=%v cell %d: non-printable glyph %q", ts, mode, i, g)
				}
			}
		}
	}
}

func TestStoryBeats(t *testing.T) {
	if st := StateAt(0.1, "claude"); st.Person != nil || st.Door != 0 {
		t.Error("room should start with only her in it and the door closed")
	}
	if st := StateAt(tDoorDone+0.1, "claude"); st.Person == nil || st.Door < 1 || st.Person.Tag != "claude" {
		t.Error("the agent should be in the open doorway wearing his name tag")
	}
	prevZ := 99.0
	for ts := tEnter; ts < tAtBed; ts += 0.05 {
		p := StateAt(ts, "claude").Person
		if p.Z > prevZ+1e-9 {
			t.Fatalf("walking backwards at t=%v", ts)
		}
		prevZ = p.Z
	}
	if p := StateAt(tKneeling+0.1, "claude").Person; math.Cos(p.Facing) > 0 || p.Lift < 0.2 || p.Lean < 0.5 || p.Look {
		t.Errorf("should be on the bed, leaning over her: %+v", p)
	}
	st := StateAt(storyLen, "codex")
	if p := st.Person; p.Eyes != EyesWink || !p.Look || p.LookAt != 0 || p.Lean < 0.5 {
		t.Errorf("should end on the bed, looking back at the camera, winking: %+v", p)
	}
	if !st.LampOff {
		t.Error("lights should go out at the end")
	}
	if !strings.Contains(StateAt(tEnter+0.5, "codex").Caption, "codex") {
		t.Error("captions should name the agent")
	}
}

func TestEncoderDiff(t *testing.T) {
	cv := NewCanvas(80, 24)
	st := StateAt(3, "claude")
	Render(cv, &st)
	for _, mode := range []ColorMode{ModeNone, Mode16, Mode256, ModeTrueColor} {
		enc := &Encoder{Mode: mode}
		full := append([]byte(nil), enc.Frame(cv)...)
		again := enc.Frame(cv)
		if len(again) != 0 {
			t.Errorf("mode %v: unchanged frame produced %d bytes", mode, len(again))
		}
		if mode == ModeNone && bytes.Contains(full, []byte("38;")) {
			t.Error("mono output contains colour escapes")
		}
		st2 := StateAt(3.1, "claude")
		Render(cv, &st2)
		if delta := enc.Frame(cv); len(delta) == 0 || len(delta) >= len(full) {
			t.Errorf("mode %v: delta frame %d bytes, full %d", mode, len(delta), len(full))
		}
		Render(cv, &st)
	}
}

func TestDetectColorMode(t *testing.T) {
	env := func(kv ...string) func(string) string {
		m := map[string]string{}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		return func(k string) string { return m[k] }
	}
	cases := []struct {
		get  func(string) string
		win  bool
		want ColorMode
	}{
		{env("COLORTERM", "truecolor", "TERM", "xterm-256color"), false, ModeTrueColor},
		{env("TERM", "xterm-256color"), false, Mode256},
		{env("TERM", "xterm-256color", "NO_COLOR", "1"), false, ModeNone},
		{env("TERM", "linux"), false, Mode16},
		{env("TERM", "xterm-kitty"), false, ModeTrueColor},
		{env(), true, ModeTrueColor},
		{env("TERM", "dumb"), false, ModeNone},
	}
	for i, c := range cases {
		if got := DetectColorMode(c.get, c.win); got != c.want {
			t.Errorf("case %d: got %v, want %v", i, got, c.want)
		}
	}
}

func TestQuantize(t *testing.T) {
	if got := to256(RGB{255, 0, 0}); got != 196 {
		t.Errorf("to256(red) = %d", got)
	}
	if got := to256(RGB{128, 128, 128}); got < 232 {
		t.Errorf("to256(grey) = %d, want grey ramp", got)
	}
	if got := to16(RGB{250, 250, 250}); got != 15 {
		t.Errorf("to16(white) = %d", got)
	}
}

func TestLogoRowsAligned(t *testing.T) {
	for _, r := range logo {
		if len(r) != len(logo[0]) {
			t.Fatalf("ragged logo:\n%s", strings.Join(logo, "\n"))
		}
	}
}

func TestSheMoves(t *testing.T) {
	a, b := herAt(0.1), herAt(tDoorDone)
	if b.Rise <= a.Rise || b.Look >= 0 {
		t.Errorf("she should prop herself up and look at the door: %+v -> %+v", a, b)
	}
	if herAt(tClimb+0.3).Wiggle == 0 {
		t.Error("she should squirm while he climbs on")
	}
	if c := herAt(tLookBack + 0.5); c.KneeL < 0.9 || c.KneeR < 0.9 {
		t.Errorf("her knees should be up by the end: %+v", c)
	}
	moved := 0.0
	for ts := 0.0; ts < storyLen; ts += 0.1 {
		a, b := herAt(ts), herAt(ts+0.1)
		moved += math.Abs(a.KneeL-b.KneeL) + math.Abs(a.KneeR-b.KneeR) + math.Abs(a.Sway-b.Sway)
	}
	if moved < 3 {
		t.Errorf("her legs barely move (total %.2f)", moved)
	}
}
