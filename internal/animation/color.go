package animation

import (
	"fmt"
	"strings"
)

// RGB is a 24-bit colour. Scenes are always composed in RGB and quantised at
// encode time to whatever the terminal can display.
type RGB struct{ R, G, B uint8 }

func clamp8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// Scale multiplies every channel by f.
func (c RGB) Scale(f float64) RGB {
	return RGB{clamp8(float64(c.R) * f), clamp8(float64(c.G) * f), clamp8(float64(c.B) * f)}
}

// Lit multiplies the colour by a per-channel light intensity.
func (c RGB) Lit(l light) RGB {
	return RGB{clamp8(float64(c.R) * l[0]), clamp8(float64(c.G) * l[1]), clamp8(float64(c.B) * l[2])}
}

// Invert returns the photographic negative.
func (c RGB) Invert() RGB { return RGB{255 - c.R, 255 - c.G, 255 - c.B} }

// Mix linearly interpolates from a to b.
func Mix(a, b RGB, t float64) RGB {
	return RGB{
		clamp8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		clamp8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		clamp8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
	}
}

func (c RGB) luma() float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
}

// ColorMode is the colour depth used when encoding frames.
type ColorMode int

const (
	// ModeNone emits plain ASCII with no colour escapes (NO_COLOR, dumb-ish terminals).
	ModeNone ColorMode = iota
	// Mode16 uses the 16 standard ANSI colours.
	Mode16
	// Mode256 uses the xterm 256-colour palette.
	Mode256
	// ModeTrueColor uses 24-bit colour escapes.
	ModeTrueColor
)

func (m ColorMode) String() string {
	switch m {
	case ModeNone:
		return "none"
	case Mode16:
		return "16"
	case Mode256:
		return "256"
	case ModeTrueColor:
		return "truecolor"
	}
	return "unknown"
}

// ParseColorMode parses a --color value. auto is true for "auto" (or ""),
// meaning the caller should use DetectColorMode.
func ParseColorMode(s string) (mode ColorMode, auto bool, err error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return 0, true, nil
	case "truecolor", "24bit", "true":
		return ModeTrueColor, false, nil
	case "256", "8bit":
		return Mode256, false, nil
	case "16", "ansi":
		return Mode16, false, nil
	case "none", "mono", "no", "off", "ascii":
		return ModeNone, false, nil
	}
	return 0, false, fmt.Errorf("unknown color mode %q (want auto, truecolor, 256, 16 or none)", s)
}

// DetectColorMode guesses the terminal's colour depth from the environment.
// VTEnabledWindows should be true when running on a Windows console that
// accepted VT mode, all of which support 24-bit colour.
func DetectColorMode(getenv func(string) string, vtEnabledWindows bool) ColorMode {
	if getenv("NO_COLOR") != "" {
		return ModeNone
	}
	switch strings.ToLower(getenv("COLORTERM")) {
	case "truecolor", "24bit":
		return ModeTrueColor
	}
	if vtEnabledWindows || getenv("WT_SESSION") != "" {
		return ModeTrueColor
	}
	switch getenv("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "vscode", "ghostty", "Hyper", "Tabby", "rio":
		return ModeTrueColor
	}
	term := strings.ToLower(getenv("TERM"))
	for _, s := range []string{"truecolor", "24bit", "direct", "kitty", "alacritty", "foot", "wezterm", "ghostty"} {
		if strings.Contains(term, s) {
			return ModeTrueColor
		}
	}
	switch {
	case strings.Contains(term, "256"):
		return Mode256
	case term == "" || term == "dumb" || strings.HasPrefix(term, "vt"):
		return ModeNone
	}
	return Mode16
}

var cubeLevels = [6]int{0, 95, 135, 175, 215, 255}

func sqDist(c RGB, r, g, b int) int {
	dr, dg, db := int(c.R)-r, int(c.G)-g, int(c.B)-b
	return 2*dr*dr + 4*dg*dg + 3*db*db
}

// to256 maps c to the nearest xterm-256 index (colour cube or grey ramp).
func to256(c RGB) int {
	idx := func(v uint8) int {
		switch {
		case v < 48:
			return 0
		case v < 115:
			return 1
		}
		return (int(v) - 35) / 40
	}
	r, g, b := idx(c.R), idx(c.G), idx(c.B)
	cube := 16 + 36*r + 6*g + b
	cubeDist := sqDist(c, cubeLevels[r], cubeLevels[g], cubeLevels[b])

	avg := (int(c.R) + int(c.G) + int(c.B)) / 3
	gi := 0
	if avg > 238 {
		gi = 23
	} else if avg > 8 {
		gi = (avg - 8) / 10
	}
	gv := 8 + 10*gi
	if sqDist(c, gv, gv, gv) < cubeDist {
		return 232 + gi
	}
	return cube
}

var ansi16 = [16]RGB{
	{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0},
	{0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229},
	{127, 127, 127}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
	{92, 92, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

// to16 maps c to the nearest of the 16 standard ANSI colours.
func to16(c RGB) int {
	best, bestD := 0, 1<<62
	for i, p := range ansi16 {
		if d := sqDist(c, int(p.R), int(p.G), int(p.B)); d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// Paint returns the escape sequence selecting c as the foreground (or
// background) colour in mode m; it is empty in ModeNone.
func (m ColorMode) Paint(c RGB, background bool) string {
	e := Encoder{Mode: m}
	if m == ModeNone {
		return ""
	}
	return "\x1b[" + string(e.appendColor(nil, e.colorKey(c), background)) + "m"
}
