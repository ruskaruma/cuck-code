package animation

import "math"

// The room is ray-cast, one ray per character cell, from a fixed camera: your
// eyes, sitting in the chair. World units are metres. The camera is the
// origin, +X is right, +Y is up and +Z points into the room. Nothing in this
// file ever moves the camera.

type vec3 struct{ X, Y, Z float64 }

func (a vec3) sub(b vec3) vec3      { return vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a vec3) scale(f float64) vec3 { return vec3{a.X * f, a.Y * f, a.Z * f} }
func (a vec3) dot(b vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a vec3) length() float64      { return math.Sqrt(a.dot(a)) }

// light is a per-channel intensity multiplier.
type light [3]float64

func (l *light) add(c light, k float64) {
	l[0] += c[0] * k
	l[1] += c[1] * k
	l[2] += c[2] * k
}

const (
	floorY = -1.1 // seated eye height is 1.1 m above the floor
	ceilY  = 1.4
	wallX  = 2.0
	backZ  = 6.0

	doorL, doorR = -1.75, -0.85
	doorTop      = 0.95
	doorFrame    = 0.08
	doorWidth    = doorR - doorL

	hallL, hallR = -3.6, 0.3
	hallBackZ    = 7.3
	hallCeil     = 1.25
)

// Surface identifiers, used for occlusion bookkeeping and outline glyphs.
const (
	sNone uint8 = iota
	sBack
	sFloor
	sCeil
	sLeft
	sRight
	sHall
	sDoor
	sBed
	sFurniture
	sLamp
	sWoman
	sPerson
	sChair
)

var (
	lampPos = vec3{-0.02, -0.22, 5.71}
	hallPos = vec3{-1.3, 1.1, 6.7}

	ambientLight = light{0.32, 0.31, 0.42} // moonlight blue
	lampLight    = light{1.00, 0.66, 0.36} // warm bedside lamp
	hallLight    = light{1.00, 0.90, 0.72} // bare hallway bulb
	fillLight    = light{0.62, 0.66, 0.85} // the glow of your own screen, from behind the chair
	fillPos      = vec3{0.3, 0.2, -0.5}
)

var (
	colWall      = RGB{150, 112, 132}
	colWallDark  = RGB{140, 103, 124}
	colBaseboard = RGB{70, 48, 40}
	colCeiling   = RGB{104, 98, 112}
	colFloor     = RGB{138, 92, 58}
	colFloorSeam = RGB{92, 58, 36}
	colRug       = RGB{44, 88, 98}
	colRugBorder = RGB{168, 124, 62}
	colDoorFrame = RGB{92, 62, 42}
	colDoor      = RGB{128, 84, 52}
	colDoorInset = RGB{100, 64, 40}
	colKnob      = RGB{215, 170, 70}
	colHallWall  = RGB{182, 162, 124}
	colGlass     = RGB{24, 34, 72}
	colCurtain   = RGB{128, 34, 62}
	colWood      = RGB{96, 58, 36}
	colSheet     = RGB{222, 216, 206}
	colBlanket   = RGB{150, 28, 48}
	colPillow    = RGB{240, 236, 228}
	colShade     = RGB{255, 206, 140}
	colBrass     = RGB{160, 124, 60}
	colSignFrame = RGB{58, 40, 26}
	colSign      = RGB{226, 204, 170}
	colSignText  = RGB{120, 40, 60}
	colArmrest   = RGB{104, 26, 30}
	colArmTop    = RGB{168, 60, 56}
)

type camera struct {
	w, h   int
	f      float64 // focal length in columns
	cx, cy float64
}

// newCamera frames the room for a w×h terminal. Cells are assumed to be
// roughly twice as tall as they are wide, so vertical focal length is f/2.
func newCamera(w, h int) camera {
	f := math.Min(2.4*float64(h), 0.9*float64(w))
	return camera{w: w, h: h, f: f, cx: float64(w) / 2, cy: float64(h) * 0.42}
}

// ray returns the direction (X, Y, 1) through the centre of a cell.
func (c camera) ray(col, row int) (X, Y float64) {
	X = (float64(col) + 0.5 - c.cx) / c.f
	Y = (c.cy - (float64(row) + 0.5)) / (c.f / 2)
	return
}

// project maps a world point to fractional cell coordinates.
func (c camera) project(p vec3) (col, row float64) {
	return c.cx + c.f*p.X/p.Z, c.cy - c.f/2*p.Y/p.Z
}

type hit struct {
	t    float64
	surf uint8
	bg   RGB
	ch   byte // overlay glyph (0 = none)
	fg   RGB
	mono byte
}

type texFunc func(e *env, p, n vec3) (RGB, byte)

type box struct {
	min, max vec3
	surf     uint8
	col      RGB
	mono     byte
	emissive bool
	tex      texFunc
}

// intersect returns the entry distance and face normal of the ray (X, Y, 1).
func (b *box) intersect(X, Y float64) (float64, vec3, bool) {
	tmin, tmax := b.min.Z, b.max.Z
	n := vec3{0, 0, -1}
	slab := func(d, lo, hi float64, neg, pos vec3) bool {
		if d == 0 {
			return lo <= 0 && 0 <= hi
		}
		a, c, na := lo/d, hi/d, neg
		if a > c {
			a, c, na = c, a, pos
		}
		if a > tmin {
			tmin, n = a, na
		}
		if c < tmax {
			tmax = c
		}
		return true
	}
	if !slab(X, b.min.X, b.max.X, vec3{-1, 0, 0}, vec3{1, 0, 0}) ||
		!slab(Y, b.min.Y, b.max.Y, vec3{0, -1, 0}, vec3{0, 1, 0}) ||
		tmin > tmax || tmin <= 0 {
		return 0, n, false
	}
	return tmin, n, true
}

var furniture = []box{
	// Bed: frame, mattress, blanket, pillows, headboard, footboard.
	{min: vec3{0.30, -1.10, 4.00}, max: vec3{1.95, -0.80, 6.00}, surf: sBed, col: colWood, mono: '='},
	{min: vec3{0.34, -0.80, 4.04}, max: vec3{1.91, -0.58, 5.94}, surf: sBed + 1, col: colSheet, mono: '-'},
	{min: vec3{0.31, -0.90, 4.02}, max: vec3{1.94, -0.55, 5.42}, surf: sBed + 2, col: colBlanket, mono: '%', tex: texBlanket},
	{min: vec3{0.45, -0.58, 5.45}, max: vec3{1.05, -0.43, 5.88}, surf: sBed + 3, col: colPillow, mono: '@'},
	{min: vec3{1.20, -0.58, 5.45}, max: vec3{1.80, -0.43, 5.88}, surf: sBed + 4, col: colPillow, mono: '@'},
	{min: vec3{0.28, -1.10, 5.92}, max: vec3{1.97, -0.05, 6.00}, surf: sBed + 5, col: colWood, mono: 'H', tex: texHeadboard},
	{min: vec3{0.28, -1.10, 3.96}, max: vec3{1.97, -0.66, 4.03}, surf: sBed + 6, col: colWood, mono: '='},
	// Nightstand and lamp.
	{min: vec3{-0.30, -1.10, 5.45}, max: vec3{0.20, -0.56, 5.98}, surf: sFurniture, col: RGB{112, 72, 46}, mono: '[', tex: texNightstand},
	{min: vec3{-0.06, -0.56, 5.66}, max: vec3{0.02, -0.36, 5.76}, surf: sFurniture + 1, col: colBrass, mono: '|'},
	{min: vec3{-0.17, -0.36, 5.56}, max: vec3{0.13, -0.10, 5.86}, surf: sLamp, col: colShade, mono: 'A', emissive: true, tex: texShade},
}

// ellipsoid is a rounded shape: the woman in the bed is built from these.
type ellipsoid struct {
	c, r vec3
	surf uint8
	col  RGB
	mono byte
	tex  texFunc
}

func (el *ellipsoid) intersect(X, Y float64) (float64, vec3, bool) {
	d := vec3{X / el.r.X, Y / el.r.Y, 1 / el.r.Z}
	c := vec3{el.c.X / el.r.X, el.c.Y / el.r.Y, el.c.Z / el.r.Z}
	a, b, k := d.dot(d), -2*d.dot(c), c.dot(c)-1
	disc := b*b - 4*a*k
	if disc < 0 {
		return 0, vec3{}, false
	}
	t := (-b - math.Sqrt(disc)) / (2 * a)
	if t <= 0 {
		return 0, vec3{}, false
	}
	p := vec3{X * t, Y * t, t}.sub(el.c)
	n := vec3{p.X / (el.r.X * el.r.X), p.Y / (el.r.Y * el.r.Y), p.Z / (el.r.Z * el.r.Z)}
	return t, n.scale(1 / n.length()), true
}

var (
	colHerHair = RGB{196, 92, 44}
	herRest    = vec3{1.50, -0.31, 5.56} // her head resting on the right pillow
)

// Her is her pose for one frame.
type Her struct {
	Rise   float64 // 0 lying back … 1 propped up on her elbows
	Look   float64 // where her eyes point: -1 toward the door/him … 0 at you … +1 away
	Wiggle float64 // -1..1, squirming under the blanket
	KneeL  float64 // 0 legs flat … 1 knee drawn right up, tenting the blanket
	KneeR  float64
	Sway   float64 // metres her knees rock sideways
	Eyes   string  // eye glyph: "." looking, "-" blink, "o" wide, "^" delighted
}

// herShapes builds her from ellipsoids: tucked in up to the chin, hair spread
// over the pillow, head lifting off it as she props herself up.
func herShapes(h Her) (head vec3, shapes []ellipsoid) {
	r := h.Rise
	w := h.Wiggle * 0.035
	head = vec3{herRest.X - 0.06*r + w*0.5, herRest.Y + 0.2*r, herRest.Z - 0.16*r}
	knee := func(side, k float64) ellipsoid {
		// A knee under the blanket: the higher it's drawn up, the taller the tent.
		return ellipsoid{
			c:    vec3{herRest.X + side*0.14 + h.Sway + w, -0.52 + 0.15*k, 4.62},
			r:    vec3{0.13, 0.1 + 0.08*k, 0.3 - 0.08*k},
			surf: sWoman, col: colBlanket, mono: '%', tex: texBlanket,
		}
	}
	return head, []ellipsoid{
		knee(-1, h.KneeL),
		knee(1, h.KneeR),
		// Hips and feet under the blanket, shifting as she squirms.
		{c: vec3{herRest.X + w, -0.55, 4.72}, r: vec3{0.30, 0.19 + 0.02*math.Abs(h.Wiggle), 0.62}, surf: sWoman, col: colBlanket, mono: '%', tex: texBlanket},
		// Shoulders under the blanket, rising with her.
		{c: vec3{herRest.X + w*0.6, -0.5 + 0.13*r, 5.3 - 0.08*r}, r: vec3{0.27, 0.13 + 0.08*r, 0.24}, surf: sWoman, col: colBlanket, mono: '%', tex: texBlanket},
		// Hair fanned over the pillow, trailing behind her head as she lifts it.
		{c: vec3{head.X, -0.42 + 0.15*r, 5.70 - 0.06*r}, r: vec3{0.30 - 0.08*r, 0.07 + 0.1*r, 0.19}, surf: sWoman + 1, col: colHerHair, mono: '@'},
		{c: head, r: vec3{0.16, 0.16, 0.15}, surf: sWoman + 2, col: colSkin, mono: 'O', tex: texHerHead},
	}
}

func texHerHead(e *env, p, n vec3) (RGB, byte) {
	if p.Y > e.herHead.Y+0.08 || n.Z > 0.2 || math.Abs(p.X-e.herHead.X) > 0.12 {
		return colHerHair, 0
	}
	return colSkin, 0
}

func texBlanket(_ *env, p, n vec3) (RGB, byte) {
	// A folded-back band near the pillows.
	if n.Y > 0 && p.Z > 5.22 {
		return colSheet, 0
	}
	return colBlanket, 0
}

func texHeadboard(_ *env, p, n vec3) (RGB, byte) {
	if n.Z < 0 && math.Mod(p.X-0.28, 0.24) < 0.05 {
		return colWood.Scale(0.72), 0
	}
	return colWood.Scale(0.9), 0
}

func texNightstand(_ *env, p, n vec3) (RGB, byte) {
	c := RGB{112, 72, 46}
	if n.Z < 0 && math.Abs(p.Y+0.83) < 0.02 {
		return c.Scale(0.6), 0
	}
	if n.Z < 0 && math.Abs(p.Y+0.72) < 0.03 && math.Abs(p.X+0.05) < 0.04 {
		return colBrass, 0
	}
	return c, 0
}

func texShade(e *env, p, n vec3) (RGB, byte) {
	if e.lampGain == 0 {
		return colShade.Scale(0.22), 0
	}
	if n.Y > 0 {
		return colShade.Scale(0.8), 0
	}
	return colShade, 0
}

// env is the per-frame lighting and scenery state shared by every ray.
type env struct {
	door     float64  // door angle in radians, 0 = closed
	doorLit  float64  // how much hallway light spills into the room, 0..1
	cellW    float64  // metres per column at the back wall
	cellH    float64  // metres per row at the back wall
	sign     []string // lines of text on the framed sign above the bed
	herHead  vec3
	her      []ellipsoid
	lampGain float64
}

func newEnv(cam camera, st *State) *env {
	e := &env{
		door:     st.Door,
		doorLit:  math.Min(1, math.Sin(math.Min(st.Door, math.Pi/2))*1.3),
		cellW:    backZ / cam.f,
		cellH:    backZ / (cam.f / 2),
		lampGain: 1,
	}
	if st.LampOff {
		e.lampGain = 0
	}
	e.herHead, e.her = herShapes(st.Her)
	cols := (signR - signL - 2*signBorder) / e.cellW
	rows := (signT - signB - 2*signBorder) / e.cellH
	for _, lines := range signTexts {
		width := 0
		for _, l := range lines {
			width = max(width, len(l))
		}
		if float64(width)+1 <= cols && float64(len(lines)) <= rows {
			e.sign = lines
			break
		}
	}
	return e
}

// shade lights a base colour at point p with surface normal n.
func (e *env) shade(base RGB, p, n vec3) RGB {
	var l light
	l.add(ambientLight, 0.8+0.25*n.Y)

	if p.Z <= backZ+1e-6 {
		d := lampPos.sub(p)
		dist := d.length()
		if ndl := n.dot(d.scale(1 / dist)); ndl > 0 {
			l.add(lampLight, e.lampGain*2.2*ndl/(1+0.42*dist*dist))
		}
	}

	if p.Z <= backZ+1e-6 {
		d := fillPos.sub(p)
		dist := d.length()
		if ndl := n.dot(d.scale(1 / dist)); ndl > 0 {
			l.add(fillLight, 0.75*ndl/(1+0.025*dist*dist))
		}
	}

	if p.Z > backZ+1e-6 {
		// Hallway surfaces are lit directly by the bulb.
		d := hallPos.sub(p)
		dist := d.length()
		ndl := math.Max(0.15, n.dot(d.scale(1/dist)))
		l.add(hallLight, 1.9*ndl/(1+0.35*dist*dist))
	} else if e.doorLit > 0 {
		// Bedroom surfaces only see the bulb through the open doorway.
		d := hallPos.sub(p)
		s := (backZ - p.Z) / d.Z
		ix, iy := p.X+s*d.X, p.Y+s*d.Y
		if ix > doorL && ix < doorR && iy > floorY && iy < doorTop {
			dist := d.length()
			if ndl := n.dot(d.scale(1 / dist)); ndl > 0 {
				l.add(hallLight, e.doorLit*3.2*ndl/(1+0.16*dist*dist))
			}
		}
	}
	return base.Lit(l)
}

// cast traces one ray through the bedroom.
func (e *env) cast(X, Y float64) hit {
	t, surf := backZ, sBack
	if X < 0 {
		if tt := -wallX / X; tt < t {
			t, surf = tt, sLeft
		}
	} else if X > 0 {
		if tt := wallX / X; tt < t {
			t, surf = tt, sRight
		}
	}
	if Y < 0 {
		if tt := floorY / Y; tt < t {
			t, surf = tt, sFloor
		}
	} else if Y > 0 {
		if tt := ceilY / Y; tt < t {
			t, surf = tt, sCeil
		}
	}
	p := vec3{X * t, Y * t, t}
	h := hit{t: t, surf: surf}

	switch surf {
	case sBack:
		if p.X > doorL && p.X < doorR && p.Y < doorTop {
			h = e.doorway(X, Y)
			break
		}
		h.bg, h.ch, h.mono = e.backWall(p)
		h.bg = e.shade(h.bg, p, vec3{0, 0, -1})
	case sLeft, sRight:
		var n vec3
		if surf == sLeft {
			n = vec3{1, 0, 0}
		} else {
			n = vec3{-1, 0, 0}
		}
		base, ch, emissive := e.sideWall(p, surf == sLeft)
		if emissive {
			h.bg = base
		} else {
			h.bg = e.shade(base, p, n)
		}
		h.ch, h.mono = ch, ch
		if ch != 0 {
			h.fg = RGB{230, 230, 255}
		}
	case sFloor:
		base, mono := e.floor(p)
		h.bg, h.mono = e.shade(base, p, vec3{0, 1, 0}), mono
	case sCeil:
		h.bg = e.shade(colCeiling, p, vec3{0, -1, 0})
	}

	for i := range furniture {
		b := &furniture[i]
		bt, n, ok := b.intersect(X, Y)
		if !ok || bt >= h.t {
			continue
		}
		bp := vec3{X * bt, Y * bt, bt}
		base, ch := b.col, byte(0)
		if b.tex != nil {
			base, ch = b.tex(e, bp, n)
		}
		h = hit{t: bt, surf: b.surf, mono: b.mono, ch: ch}
		if b.emissive {
			h.bg = base
		} else {
			// Faces pointing at the camera read better slightly brightened.
			h.bg = e.shade(base, bp, n).Scale(1 - 0.12*math.Abs(n.X))
		}
	}
	for i := range e.her {
		el := &e.her[i]
		et, n, ok := el.intersect(X, Y)
		if !ok || et >= h.t {
			continue
		}
		ep := vec3{X * et, Y * et, et}
		base := el.col
		if el.tex != nil {
			base, _ = el.tex(e, ep, n)
		}
		h = hit{t: et, surf: el.surf, mono: el.mono, bg: e.shade(base, ep, n)}
	}
	if h.ch != 0 && h.fg == (RGB{}) {
		h.fg = h.bg.Scale(0.55)
	}
	return h
}

// The framed sign above the bed, with the moral of the story.
const signL, signR, signB, signT, signBorder = 0.25, 1.95, 0.12, 0.98, 0.05

// signTexts are tried in order until one fits the terminal.
var signTexts = [][]string{
	{"TAKE THE BACK SEAT.", "LET IT DO THE WORK."},
	{"TAKE THE BACK SEAT", "LET IT WORK"},
	{"BACK SEAT,", "LET IT WORK"},
	{"LET IT", "WORK"},
}

func (e *env) backWall(p vec3) (RGB, byte, byte) {
	if p.X > signL && p.X < signR && p.Y > signB && p.Y < signT {
		if p.X < signL+signBorder || p.X > signR-signBorder || p.Y < signB+signBorder || p.Y > signT-signBorder {
			return colSignFrame, 0, '+'
		}
		mid := (signB + signT) / 2
		for i, line := range e.sign {
			ly := mid + (float64(len(e.sign)-1)/2-float64(i))*e.cellH
			if math.Abs(p.Y-ly) >= e.cellH/2 {
				continue
			}
			x0 := (signL+signR)/2 - float64(len(line))*e.cellW/2
			if j := int(math.Floor((p.X - x0) / e.cellW)); j >= 0 && j < len(line) && line[j] != ' ' {
				return colSign, line[j], line[j]
			}
		}
		return colSign, 0, ' '
	}
	// Door frame.
	if p.X > doorL-doorFrame && p.X < doorR+doorFrame && p.Y < doorTop+doorFrame {
		return colDoorFrame, 0, '|'
	}
	if p.Y < floorY+0.1 {
		return colBaseboard, 0, '_'
	}
	return wallpaper(p.X), 0, ' '
}

func wallpaper(a float64) RGB {
	if frac(a/0.36) < 0.14 {
		return colWallDark
	}
	return colWall
}

func (e *env) sideWall(p vec3, left bool) (RGB, byte, bool) {
	if p.Y < floorY+0.1 {
		return colBaseboard, 0, false
	}
	if left {
		// Curtained window with a night sky.
		if p.Y > -0.5 && p.Y < 0.95 && ((p.Z > 3.92 && p.Z < 4.16) || (p.Z > 5.24 && p.Z < 5.48)) {
			if frac(p.Z/0.06) < 0.45 {
				return colCurtain.Scale(0.75), 0, false
			}
			return colCurtain, 0, false
		}
		if p.Z > 4.16 && p.Z < 5.24 && p.Y > -0.2 && p.Y < 0.8 {
			if p.Z < 4.22 || p.Z > 5.18 || p.Y < -0.14 || p.Y > 0.74 ||
				math.Abs(p.Z-4.70) < 0.025 || math.Abs(p.Y-0.30) < 0.025 {
				return RGB{210, 200, 190}, 0, false
			}
			sky := Mix(colGlass, RGB{52, 60, 104}, (0.74-p.Y)/0.9)
			if hash2(math.Floor(p.Z*28), math.Floor(p.Y*14)) < 0.07 {
				return sky, '.', true
			}
			return sky, 0, true
		}
	}
	return wallpaper(p.Z), 0, false
}

func (e *env) floor(p vec3) (RGB, byte) {
	// Oval rug in front of the bed.
	rx, rz := (p.X+0.45)/0.95, (p.Z-4.45)/0.85
	if r := rx*rx + rz*rz; r < 1 {
		if r > 0.72 {
			return colRugBorder, '+'
		}
		return colRug, '~'
	}
	if frac(p.X/0.24) < 0.1 {
		return colFloorSeam, '.'
	}
	plank := math.Floor(p.X / 0.24)
	if frac(p.Z/1.6+hash2(plank, 7)) < 0.025 {
		return colFloorSeam, 0
	}
	return colFloor, 0
}

// doorway handles rays passing through the door opening: the swinging door
// panel first, then the lit hallway beyond it.
func (e *env) doorway(X, Y float64) hit {
	// The door is hinged on its right edge and swings away into the hallway.
	sin, cos := math.Sin(e.door), math.Cos(e.door)
	if den := X*sin + cos; den != 0 {
		s := (doorR - X*backZ) / den
		t := backZ + s*sin
		y := Y * t
		if s >= 0 && s <= doorWidth && y >= floorY && y <= doorTop && t >= backZ {
			base := colDoor
			// Two inset panels and a knob, in door-local coordinates.
			inU, inV := s > 0.12 && s < doorWidth-0.12, (y > -0.92 && y < -0.2) || (y > 0.02 && y < 0.8)
			if inU && inV {
				edge := math.Min(math.Min(s-0.12, doorWidth-0.12-s), math.Min(math.Abs(y+0.92), math.Abs(y+0.2)))
				if y > 0 {
					edge = math.Min(math.Min(s-0.12, doorWidth-0.12-s), math.Min(math.Abs(y-0.02), math.Abs(y-0.8)))
				}
				if edge < 0.035 {
					base = colDoorInset
				}
			}
			mono := byte('#')
			if s > doorWidth-0.1 && s < doorWidth-0.04 && math.Abs(y+0.1) < 0.045 {
				base, mono = colKnob, 'o'
			}
			// The panel darkens as it turns edge-on to us.
			n := vec3{sin, 0, -cos}
			c := e.shade(base, vec3{X * t, y, backZ - 0.01}, n).Scale(0.55 + 0.45*cos)
			return hit{t: t, surf: sDoor, bg: c, mono: mono}
		}
	}

	t, n := hallBackZ, vec3{0, 0, -1}
	if X < 0 {
		if tt := hallL / X; tt < t {
			t, n = tt, vec3{1, 0, 0}
		}
	} else if X > 0 {
		if tt := hallR / X; tt < t {
			t, n = tt, vec3{-1, 0, 0}
		}
	}
	if Y < 0 {
		if tt := floorY / Y; tt < t {
			t, n = tt, vec3{0, 1, 0}
		}
	} else if Y > 0 {
		if tt := hallCeil / Y; tt < t {
			t, n = tt, vec3{0, -1, 0}
		}
	}
	p := vec3{X * t, Y * t, t}
	base, mono := colHallWall, byte(':')
	if n.Y > 0 {
		base, mono = e.floor(p)
		if mono == 0 {
			mono = ' '
		}
	}
	return hit{t: t, surf: sHall, bg: e.shade(base, p, n), mono: mono}
}

func frac(x float64) float64 { return x - math.Floor(x) }

func hash2(a, b float64) float64 {
	return frac(math.Sin(a*127.1+b*311.7) * 43758.5453)
}
