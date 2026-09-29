package animation

import (
	"strconv"
	"strings"
)

// Cell is one character cell of a frame.
type Cell struct {
	Ch   byte // glyph drawn in colour modes
	Mono byte // glyph drawn when colour is unavailable; 0 means use Ch
	Bold bool
	FG   RGB
	BG   RGB
}

// Glyph returns the character to draw for the given mode.
func (c Cell) Glyph(mode ColorMode) byte {
	if (mode == ModeNone || mode == Mode16) && c.Mono != 0 {
		return c.Mono
	}
	if c.Ch == 0 {
		return ' '
	}
	return c.Ch
}

// Canvas is a frame buffer of cells plus the per-cell bookkeeping the
// renderer needs (depth and which surface a cell shows).
type Canvas struct {
	W, H  int
	Cells []Cell
	depth []float64
	surf  []uint8
}

// NewCanvas allocates a w by h canvas.
func NewCanvas(w, h int) *Canvas {
	n := w * h
	return &Canvas{W: w, H: h, Cells: make([]Cell, n), depth: make([]float64, n), surf: make([]uint8, n)}
}

func (c *Canvas) in(x, y int) bool { return x >= 0 && y >= 0 && x < c.W && y < c.H }

// At returns the cell at (x, y), or nil when out of bounds.
func (c *Canvas) At(x, y int) *Cell {
	if !c.in(x, y) {
		return nil
	}
	return &c.Cells[y*c.W+x]
}

// Fill paints every cell.
func (c *Canvas) Fill(cell Cell) {
	for i := range c.Cells {
		c.Cells[i] = cell
	}
}

// Text writes s at (x, y) in fg, keeping each cell's background.
func (c *Canvas) Text(x, y int, s string, fg RGB, bold bool) {
	for i := 0; i < len(s); i++ {
		if p := c.At(x+i, y); p != nil {
			p.Ch, p.Mono, p.FG, p.Bold = s[i], s[i], fg, bold
		}
	}
}

// Plain renders the canvas as ASCII without escapes (tests and debugging).
func (c *Canvas) Plain(mode ColorMode) string {
	var b strings.Builder
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			b.WriteByte(c.Cells[y*c.W+x].Glyph(mode))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// Encoder turns canvases into terminal output, emitting only the cells that
// changed since the previous frame. The room is static, so after the first
// frame only the figure's cells are redrawn, which keeps output small enough
// for slow consoles.
type Encoder struct {
	Mode ColorMode

	prev   []Cell
	pw, ph int
	buf    []byte

	sgrValid bool
	sgrFG    int
	sgrBG    int
	sgrBold  bool
}

// Invalidate forces the next frame to be drawn in full.
func (e *Encoder) Invalidate() { e.prev = nil }

func (e *Encoder) colorKey(c RGB) int {
	switch e.Mode {
	case ModeTrueColor:
		return int(c.R)<<16 | int(c.G)<<8 | int(c.B)
	case Mode256:
		return to256(c)
	case Mode16:
		return to16(c)
	}
	return 0
}

func (e *Encoder) appendColor(b []byte, key int, bg bool) []byte {
	switch e.Mode {
	case ModeTrueColor:
		if bg {
			b = append(b, "48;2;"...)
		} else {
			b = append(b, "38;2;"...)
		}
		b = strconv.AppendInt(b, int64(key>>16&0xff), 10)
		b = append(b, ';')
		b = strconv.AppendInt(b, int64(key>>8&0xff), 10)
		b = append(b, ';')
		b = strconv.AppendInt(b, int64(key&0xff), 10)
	case Mode256:
		if bg {
			b = append(b, "48;5;"...)
		} else {
			b = append(b, "38;5;"...)
		}
		b = strconv.AppendInt(b, int64(key), 10)
	case Mode16:
		base := 30
		if bg {
			base = 40
		}
		if key >= 8 {
			base += 60
			key -= 8
		}
		b = strconv.AppendInt(b, int64(base+key), 10)
	}
	return b
}

// Frame encodes c. The returned slice is reused by the next call.
func (e *Encoder) Frame(c *Canvas) []byte {
	b := e.buf[:0]
	full := e.prev == nil || e.pw != c.W || e.ph != c.H
	if full {
		b = append(b, "\x1b[0m\x1b[2J"...)
		e.sgrValid = false
	}
	curX, curY := -1, -1
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			// Never touch the bottom-right cell: on some consoles writing it
			// scrolls the screen even with autowrap disabled.
			if y == c.H-1 && x == c.W-1 {
				continue
			}
			i := y*c.W + x
			cell := c.Cells[i]
			if !full && e.prev[i] == cell {
				continue
			}
			if x != curX || y != curY {
				b = append(b, "\x1b["...)
				b = strconv.AppendInt(b, int64(y+1), 10)
				b = append(b, ';')
				b = strconv.AppendInt(b, int64(x+1), 10)
				b = append(b, 'H')
			}
			if e.Mode != ModeNone {
				fg, bg := e.colorKey(cell.FG), e.colorKey(cell.BG)
				if !e.sgrValid || fg != e.sgrFG || bg != e.sgrBG || cell.Bold != e.sgrBold {
					b = append(b, "\x1b["...)
					if cell.Bold {
						b = append(b, "1;"...)
					} else {
						b = append(b, "22;"...)
					}
					b = e.appendColor(b, fg, false)
					b = append(b, ';')
					b = e.appendColor(b, bg, true)
					b = append(b, 'm')
					e.sgrValid, e.sgrFG, e.sgrBG, e.sgrBold = true, fg, bg, cell.Bold
				}
			}
			b = append(b, cell.Glyph(e.Mode))
			curX, curY = x+1, y
		}
	}
	if len(e.prev) != len(c.Cells) {
		e.prev = make([]Cell, len(c.Cells))
	}
	copy(e.prev, c.Cells)
	e.pw, e.ph = c.W, c.H
	e.buf = b
	return b
}
