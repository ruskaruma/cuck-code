package animation

import "math"

// State is everything that varies between frames. The room and camera never change.
type State struct {
	Door    float64 // door angle in radians, 0 = closed
	Person  *Person // nil when nobody is in the scene
	Caption string  // dim caption along the bottom of the screen
	HerLine string  // what she says, in a speech bubble over the bed
	SFX     string  // sound effect text by the bed
	LampOff bool    // lights out
	Kiss    float64 // 0 none, then 0..1 through the kiss
	Her     Her     // what she's doing
}

// Render draws st into cv.
func Render(cv *Canvas, st *State) {
	cam := newCamera(cv.W, cv.H)
	e := newEnv(cam, st)
	var fig *figure
	if st.Person != nil {
		fig = newFigure(st.Person)
	}

	for row := 0; row < cv.H; row++ {
		for col := 0; col < cv.W; col++ {
			X, Y := cam.ray(col, row)
			h := e.cast(X, Y)
			cell := Cell{Ch: ' ', Mono: h.mono, BG: h.bg, FG: h.fg}
			if h.ch != 0 {
				cell.Ch = h.ch
				if cell.Mono == 0 || cell.Mono == ' ' {
					cell.Mono = h.ch
				}
			}
			surf, depth := h.surf, h.t
			if fig != nil && fig.p.Z < h.t {
				z := fig.p.Z
				if pt := fig.sample(X*z-fig.p.X, Y*z-floorY, z/cam.f); pt != pNone {
					cell = fig.cell(pt, e)
					surf, depth = sPerson, z
				}
			}
			i := row*cv.W + col
			cv.Cells[i], cv.surf[i], cv.depth[i] = cell, surf, depth
		}
	}

	outlineRoom(cv)
	if fig != nil {
		outlineSkin(cv)
	}
	drawChair(cv)
	vignette(cv)
	// Her eyes, peeking over the blanket.
	eyes := st.Her.Eyes
	if eyes == "" {
		eyes = "."
	}
	hh := e.herHead
	for _, side := range [2]float64{-1, 1} {
		c, r := cam.project(vec3{hh.X + side*0.06 + st.Her.Look*0.035, hh.Y + 0.02, hh.Z - 0.14})
		drawFeature(cv, feature{col: int(c), row: int(r), text: eyes, fg: colPupil, keepBG: true, bold: true, onSurf: sWoman + 2})
	}
	if fig != nil {
		for _, ft := range fig.features(cam) {
			drawFeature(cv, ft)
		}
	}
	if st.SFX != "" {
		c, r := cam.project(vec3{1.55, -0.2, 4.2})
		cv.Text(int(c)-len(st.SFX)/2, int(r), st.SFX, RGB{255, 214, 120}, true)
	}
	if st.Kiss > 0 {
		// A heart floats up from the kiss.
		hh := e.herHead
		c, r := cam.project(vec3{hh.X - 0.22, hh.Y + 0.2 + 0.45*st.Kiss, hh.Z})
		cv.Text(int(c), int(r), "<3", RGB{255, 90, 140}, true)
		if st.Kiss < 0.75 {
			c, r = cam.project(vec3{hh.X + 0.1, hh.Y + 0.3, hh.Z})
			cv.Text(int(c), int(r), "*smooch*", RGB{255, 170, 200}, true)
		}
	}
	if st.HerLine != "" {
		speechBubble(cv, cam, e.herHead, st.HerLine)
	}
	if st.Caption != "" && cv.H > 4 {
		x := (cv.W - len(st.Caption)) / 2
		for i := -1; i <= len(st.Caption); i++ {
			if c := cv.At(x+i, cv.H-2); c != nil {
				c.BG, c.Ch, c.Mono = c.BG.Scale(0.3), ' ', ' '
			}
		}
		cv.Text(x, cv.H-2, st.Caption, RGB{240, 226, 232}, false)
	}
}

// outlineRoom draws the edges where room surfaces meet, which gives the
// monochrome fallback its shape and adds a crisp retro look in colour.
func outlineRoom(cv *Canvas) {
	shell := func(s uint8) bool { return s >= sBack && s <= sRight }
	edge := func(a, b uint8, below bool) byte {
		switch {
		case a == sLeft && b == sBack, a == sBack && b == sRight:
			if !below {
				return '|'
			}
		case a == sCeil && b == sBack, a == sBack && b == sFloor:
			if below {
				return '_'
			}
		case a == sLeft && b == sFloor, a == sCeil && b == sRight:
			return '/'
		case a == sRight && b == sFloor, a == sCeil && b == sLeft,
			a == sFloor && b == sRight, a == sLeft && b == sCeil:
			return '\\'
		case a == sFloor && b == sLeft, a == sRight && b == sCeil:
			return '/'
		}
		return 0
	}
	for y := 0; y < cv.H; y++ {
		for x := 0; x < cv.W; x++ {
			i := y*cv.W + x
			a := cv.surf[i]
			if !shell(a) {
				continue
			}
			var g byte
			if x+1 < cv.W && shell(cv.surf[i+1]) && cv.surf[i+1] != a {
				g = edge(a, cv.surf[i+1], false)
			}
			if g == 0 && y+1 < cv.H && shell(cv.surf[i+cv.W]) && cv.surf[i+cv.W] != a {
				g = edge(a, cv.surf[i+cv.W], true)
			}
			if g != 0 {
				c := &cv.Cells[i]
				c.Ch, c.Mono, c.FG = g, g, c.BG.Scale(0.5)
			}
		}
	}
}

// outlineSkin gives bare skin a silhouette in monochrome, where the fill is blank.
func outlineSkin(cv *Canvas) {
	isP := func(x, y int) bool { return cv.in(x, y) && cv.surf[y*cv.W+x] == sPerson }
	for y := 0; y < cv.H; y++ {
		for x := 0; x < cv.W; x++ {
			i := y*cv.W + x
			c := &cv.Cells[i]
			if cv.surf[i] != sPerson || c.Mono != 0 {
				continue
			}
			l, r, d := !isP(x-1, y), !isP(x+1, y), !isP(x, y+1)
			switch {
			case l && r:
				c.Mono = '|'
			case l:
				c.Mono = '('
			case r:
				c.Mono = ')'
			case d:
				c.Mono = '_'
			default:
				c.Mono = ' '
			}
		}
	}
}

// drawChair overlays the chair's padded leather armrests in the bottom
// corners: the only part of "you" that is ever on screen.
func drawChair(cv *Canvas) {
	w, h := float64(cv.W), float64(cv.H)
	reach := math.Max(8, w*0.2)
	top := h * 0.7
	for y := 0; y < cv.H; y++ {
		fy := float64(y) + 0.5
		for x := 0; x < cv.W; x++ {
			for _, right := range [2]bool{false, true} {
				fx := float64(x) + 0.5
				if right {
					fx = w - fx
				}
				if fx > reach {
					continue
				}
				// The armrest's rounded top edge dips toward the centre of the screen.
				edge := top + (h-top)*math.Pow(fx/reach, 1.8)
				if fy < edge-0.5 {
					continue
				}
				c := &cv.Cells[y*cv.W+x]
				depth := (fy - edge) / (h - top + 1)
				*c = Cell{Ch: ' ', Mono: ':', BG: colArmrest.Scale(1 - 0.45*depth)}
				switch {
				case fy < edge+0.5:
					g := byte('\\')
					if right {
						g = '/'
					}
					*c = Cell{Ch: ' ', Mono: g, BG: colArmTop}
				case (x/4+y/2)%2 == 0 && x%4 == 2 && y%2 == 1:
					c.Ch, c.Mono, c.FG = '.', '.', colArmrest.Scale(0.45) // tufting
				}
				cv.surf[y*cv.W+x] = sChair
			}
		}
	}
}

func vignette(cv *Canvas) {
	for y := 0; y < cv.H; y++ {
		dy := (float64(y)+0.5)/float64(cv.H)*2 - 1
		for x := 0; x < cv.W; x++ {
			dx := (float64(x)+0.5)/float64(cv.W)*2 - 1
			k := 1 - 0.4*math.Pow(math.Min(1, (dx*dx*0.8+dy*dy)/1.8), 1.5)
			c := &cv.Cells[y*cv.W+x]
			c.BG, c.FG = c.BG.Scale(k), c.FG.Scale(k)
		}
	}
}

func drawFeature(cv *Canvas, ft feature) {
	for i := 0; i < len(ft.text); i++ {
		x, y := ft.col+i, ft.row
		if !cv.in(x, y) {
			continue
		}
		idx := y*cv.W + x
		if ft.onPerson && cv.surf[idx] != sPerson || ft.onSurf != 0 && cv.surf[idx] != ft.onSurf {
			continue
		}
		c := &cv.Cells[idx]
		c.Ch, c.Mono, c.FG, c.Bold = ft.text[i], ft.text[i], ft.fg, ft.bold
		if !ft.keepBG {
			c.BG = ft.bg
		}
	}
}

// speechBubble draws her line in a white bubble above the bed, with a tail
// pointing down at her.
func speechBubble(cv *Canvas, cam camera, head vec3, line string) {
	hc, hr := cam.project(vec3{head.X, head.Y + 0.16, head.Z})
	col, row := int(hc), int(hr)
	text := " " + line + " "
	if len(text) > cv.W {
		text = text[:cv.W]
	}
	x := min(max(col-2, 0), cv.W-len(text)) // extends right, away from him
	y := row - 2
	if y < 0 {
		return
	}
	paper, ink := RGB{250, 248, 240}, RGB{30, 20, 28}
	for i := 0; i < len(text); i++ {
		if c := cv.At(x+i, y); c != nil {
			*c = Cell{Ch: text[i], Mono: text[i], FG: ink, BG: paper, Bold: true}
		}
	}
	if c := cv.At(col, y+1); c != nil {
		c.Ch, c.Mono, c.FG, c.Bold = 'V', 'V', paper, true
	}
}
