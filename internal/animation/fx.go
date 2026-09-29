package animation

import (
	"math/rand"
	"strings"
)

// The wink triggers this transition: a white flash, a negative frame, a
// ramping glitch that reveals the logo, then a CRT power-off.
const (
	fxFlash  = 0.05
	fxInvert = 0.10
	fxGlitch = 0.52
	fxCRT    = 0.72
	fxLength = 0.72
)

var logoLetters = map[byte][5]string{
	'C': {"  ____ ", " / ___|", "| |    ", "| |___ ", ` \____|`},
	'U': {" _   _ ", "| | | |", "| | | |", "| |_| |", ` \___/ `},
	'K': {" _  __", "| |/ /", "| ' / ", `| . \ `, `|_|\_\`},
	'O': {"  ___  ", ` / _ \ `, "| | | |", "| |_| |", ` \___/ `},
	'D': {" ____  ", `|  _ \ `, "| | | |", "| |_| |", "|____/ "},
	'E': {" _____ ", "| ____|", "|  _|  ", "| |___ ", "|_____|"},
}

func bigText(s string) []string {
	var rows [5]strings.Builder
	for i := 0; i < len(s); i++ {
		g, ok := logoLetters[s[i]]
		for r := range rows {
			if ok {
				rows[r].WriteString(g[r])
			} else {
				rows[r].WriteString("   ")
			}
		}
	}
	out := make([]string, len(rows))
	for i := range rows {
		out[i] = rows[i].String()
	}
	return out
}

var logo = bigText("CUCK CODE")

var (
	black = RGB{0, 0, 0}
	white = RGB{255, 255, 255}
	neon  = []RGB{{255, 40, 180}, {40, 255, 230}, {255, 240, 60}, {120, 90, 255}, {255, 255, 255}}
)

const noiseGlyphs = `#%$&@*+=?/\|<>~^`

// RenderFX draws the transition at time t (seconds since the wink) on top of
// base, the final story frame. frame seeds the glitch noise.
func RenderFX(cv, base *Canvas, t float64, frame int, tagline string) {
	switch {
	case t < fxFlash:
		cv.Fill(Cell{Ch: ' ', Mono: '#', BG: white})
	case t < fxInvert:
		copy(cv.Cells, base.Cells)
		for i := range cv.Cells {
			c := &cv.Cells[i]
			c.BG, c.FG = c.BG.Invert(), c.FG.Invert()
			if c.Glyph(ModeNone) == ' ' {
				c.Mono = '#'
			} else {
				c.Mono = ' '
			}
		}
	case t < fxGlitch:
		glitch(cv, base, (t-fxInvert)/(fxGlitch-fxInvert), frame, tagline)
	case t < fxCRT:
		crtOff(cv, (t-fxGlitch)/(fxCRT-fxGlitch))
	default:
		cv.Fill(Cell{Ch: ' ', BG: black})
	}
}

func glitch(cv, base *Canvas, k float64, frame int, tagline string) {
	rng := rand.New(rand.NewSource(int64(frame)*7919 + 1))
	w, h := cv.W, cv.H
	for y := 0; y < h; y++ {
		off := 0
		if rng.Float64() < 0.2+0.6*k {
			span := 2 + int(k*float64(w)/5)
			off = rng.Intn(2*span+1) - span
		}
		tint := rng.Float64() < 0.15+0.35*k
		fade := 1 - 0.75*k // darken the room so the logo pops
		for x := 0; x < w; x++ {
			sx := (x + off + w) % w
			c := base.Cells[y*w+sx]
			if tint {
				c.BG = RGB{c.BG.B, c.BG.R / 2, c.BG.G}.Scale(1.4)
			}
			c.BG, c.FG = c.BG.Scale(fade), c.FG.Scale(fade)
			cv.Cells[y*w+x] = c
		}
	}
	for n := int(k * k * float64(w*h) * 0.12); n > 0; n-- {
		c := &cv.Cells[rng.Intn(len(cv.Cells))]
		g := noiseGlyphs[rng.Intn(len(noiseGlyphs))]
		c.Ch, c.Mono, c.FG, c.Bold = g, g, neon[rng.Intn(len(neon))], rng.Intn(2) == 0
	}
	if k < 0.3 {
		return
	}
	lines := logo
	if len(logo[0])+2 > w || h < 9 {
		lines = []string{"C U C K   C O D E"}
	}
	top := (h - len(lines)) / 2
	if tagline != "" && top+len(lines)+1 < h {
		lines = append(append([]string(nil), lines...), "", tagline)
		top = (h - len(lines)) / 2
	}
	for i, s := range lines {
		if rng.Float64() > 0.55+k*0.5 {
			continue // the logo flickers in
		}
		jit := 0
		if rng.Float64() < 0.3 {
			jit = rng.Intn(5) - 2
		}
		x := (w-len(s))/2 + jit
		fg := white
		if i >= len(logo) {
			fg = RGB{255, 120, 200}
		}
		for j := 0; j < len(s); j++ {
			if c := cv.At(x+j, top+i); c != nil {
				c.BG = c.BG.Scale(0.25)
				if s[j] != ' ' {
					c.Ch, c.Mono, c.FG, c.Bold = s[j], s[j], fg, true
				} else {
					c.Ch, c.Mono = ' ', ' '
				}
			}
		}
	}
}

// crtOff collapses the picture to a line, then a dot, like an old tube TV.
func crtOff(cv *Canvas, k float64) {
	cv.Fill(Cell{Ch: ' ', BG: black})
	w, h := cv.W, cv.H
	mid := h / 2
	glow := RGB{225, 235, 255}
	if k < 0.45 {
		half := int(float64(h) / 2 * (1 - k/0.45))
		for y := mid - half; y <= mid+half; y++ {
			for x := 0; x < w; x++ {
				if c := cv.At(x, y); c != nil {
					*c = Cell{Ch: ' ', Mono: '=', BG: glow}
				}
			}
		}
		return
	}
	half := int(float64(w) / 2 * (1 - (k-0.45)/0.55))
	for x := w/2 - half; x <= w/2+half; x++ {
		if c := cv.At(x, mid); c != nil {
			*c = Cell{Ch: '-', Mono: '-', FG: white, BG: glow, Bold: true}
		}
	}
	if half <= 1 {
		if c := cv.At(w/2, mid); c != nil {
			*c = Cell{Ch: '*', Mono: '*', FG: white, BG: black, Bold: true}
		}
	}
}

// renderTiny is the fallback for terminals too small for the room: the same
// story, told as stage directions.
func renderTiny(cv *Canvas, t float64, agent string) {
	if agent == "" {
		agent = "your agent"
	}
	cv.Fill(Cell{Ch: ' ', BG: RGB{20, 14, 22}})
	line := "(she is in bed. you are the chair.)"
	switch {
	case t >= tWink:
		line = ";)"
	case t >= tLookBack:
		line = "(" + agent + " looks back at you)"
	case t >= tClimb:
		line = `she: "take me to heaven."`
	case t >= tDoorOpen:
		line = "(" + agent + " walks in)"
	}
	if len(line) > cv.W {
		line = line[:cv.W]
	}
	cv.Text((cv.W-len(line))/2, cv.H/2, line, RGB{230, 200, 220}, true)
}

// FXLength is the length of the post-wink transition in seconds.
func FXLength() float64 { return fxLength }
