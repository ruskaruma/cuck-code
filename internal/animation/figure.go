package animation

import "math"

// Eyes is the expression drawn in the figure's eyes.
type Eyes int

const (
	EyesDots Eyes = iota // unaware: small dots, looking somewhere else
	EyesOpen             // looking straight at you
	EyesWide             // caught you
	EyesWink
)

// Mouth is the expression drawn on the figure's mouth.
type Mouth int

const (
	MouthNone Mouth = iota
	MouthFlat
	MouthO
	MouthSmirk
)

// Person is the guy's pose for one frame. It is a billboard standing on
// the floor at (X, Z), so perspective scaling and occlusion by the door frame
// and bed fall out of the ray caster for free.
type Person struct {
	X, Z    float64
	Facing  float64 // radians: 0 faces the camera, +π/2 faces screen-right, π faces away
	Phase   float64 // walk cycle angle
	Stride  float64 // 0 standing still … 1 full walking stride
	Eyes    Eyes
	Mouth   Mouth
	Bubble  string // thought bubble above the head, e.g. "?" or "!"
	Sparkle bool   // the glint that accompanies the wink
	Tag     string // name tag floating above the head: the agent doing your work

	// Look turns just the head to face angle LookAt (e.g. a glance over the shoulder).
	Look   bool
	LookAt float64
	Lift   float64 // metres he stands above the floor, e.g. on the bed
	Kneel  float64 // 0 standing … 1 kneeling
	Lean   float64 // radians the upper body leans toward screen-right (over the bed)
}

type part uint8

const (
	pNone part = iota
	pHair
	pSkin
	pTank
	pPants
	pShoe
)

var (
	colHair   = RGB{40, 28, 22}
	colSkin   = RGB{230, 172, 124}
	colTank   = RGB{244, 244, 238}
	colPants  = RGB{56, 76, 118}
	colShoe   = RGB{44, 40, 40}
	colEyeW   = RGB{248, 248, 248}
	colPupil  = RGB{16, 16, 24}
	colLips   = RGB{128, 34, 40}
	colBubble = RGB{255, 226, 72}
	colTag    = RGB{217, 119, 87} // name tag: the agent he's playing
)

type vec2 struct{ u, v float64 }

// figure is a Person resolved into body-part geometry. Coordinates are in
// metres: u is horizontal offset from the person's position (screen-right),
// v is height above the floor.
type figure struct {
	p        *Person
	sa, ca   float64 // body facing
	hsa, hca float64 // head facing

	head     vec2
	hrx, hry float64
	shoulder float64
	hip      float64
	halfTop  float64
	halfHip  float64
	legs     [2][2]vec2 // hip, foot
	arms     [2][2]vec2 // shoulder, hand
	lsin     float64    // lean, rotating the upper body about the hips
	lcos     float64
	braced   bool // leaning: arms reach straight down to the mattress (screen space)
}

func newFigure(p *Person) *figure {
	f := &figure{p: p, sa: math.Sin(p.Facing), ca: math.Cos(p.Facing)}
	f.hsa, f.hca = f.sa, f.ca
	if p.Look {
		f.hsa, f.hca = math.Sin(p.LookAt), math.Cos(p.LookAt)
	}
	aca := math.Abs(f.ca)
	// Kneeling folds the shins away, lowering everything above the knees.
	bob := p.Stride*0.035*math.Abs(math.Cos(p.Phase)) - 0.6*p.Kneel

	// Slightly chibi proportions so the face stays readable at terminal resolution.
	f.head = vec2{f.hsa * 0.02, 1.55 + bob}
	f.hrx = 0.18 + 0.02*math.Abs(f.hsa)
	f.hry = 0.22
	if p.Look {
		// The look back is played for the camera: a slightly bigger head, so
		// the stare, smirk and wink read from across the room.
		f.hrx, f.hry = f.hrx*1.45, f.hry*1.35
		f.head.v += 0.05
	}
	f.shoulder = 1.30 + bob
	f.hip = 0.86 + bob
	f.halfTop = 0.13 + 0.13*aca // broad shoulders tapering to the waist
	f.halfHip = 0.12 + 0.07*aca

	for i, side := range [2]float64{-1, 1} {
		swing := p.Stride * side * math.Sin(p.Phase) * 0.24
		lift := p.Stride * math.Max(0, side*math.Cos(p.Phase)) * 0.08
		hip := vec2{side * 0.08 * aca, f.hip}
		foot := vec2{hip.u + f.sa*swing, 0.05 + lift*(0.5+0.5*aca)}
		f.legs[i] = [2]vec2{hip, foot}

		sh := vec2{side * math.Max((f.halfTop-0.02)*aca, 0.03), f.shoulder - 0.05}
		hand := vec2{sh.u + side*0.08*aca - f.sa*swing*0.8, 0.80 + bob + math.Abs(swing)*0.25}
		f.arms[i] = [2]vec2{sh, hand}
	}

	f.lsin, f.lcos = math.Sin(p.Lean), math.Cos(p.Lean)
	if p.Lean > 0.15 {
		// Leaning over the bed: hands planted on the mattress under the shoulders.
		f.braced = true
		for i := range f.arms {
			sh := f.toScreen(f.arms[i][0])
			f.arms[i] = [2]vec2{sh, {sh.u + 0.06, math.Max(sh.v-0.62, 0.22)}}
		}
	}
	return f
}

// toScreen maps a point on the upper body from body space into screen space,
// applying the lean about the hips.
func (f *figure) toScreen(p vec2) vec2 {
	dv := p.v - f.hip
	return vec2{p.u*f.lcos + dv*f.lsin, f.hip - p.u*f.lsin + dv*f.lcos}
}

// toBody is the inverse of toScreen.
func (f *figure) toBody(u, v float64) (float64, float64) {
	dv := v - f.hip
	return u*f.lcos - dv*f.lsin, f.hip + u*f.lsin + dv*f.lcos
}

// segHit reports whether (u, v) lies on a limb of the given thickness, always
// treating the limb as at least one cell wide so thin limbs never vanish.
func segHit(u, v float64, a, b vec2, thick, du float64) (bool, float64) {
	vx, vy := b.u-a.u, b.v-a.v
	l2 := vx*vx + vy*vy
	t := 0.0
	if l2 > 0 {
		t = math.Max(0, math.Min(1, ((u-a.u)*vx+(v-a.v)*vy)/l2))
	}
	dx, dy := u-(a.u+t*vx), v-(a.v+t*vy)
	if dx*dx+dy*dy <= thick*thick/4 {
		return true, t
	}
	// Cell-space test: columns are du wide and rows 2*du tall.
	cx, cy := dx/du, dy/(2*du)
	return cx*cx+cy*cy <= 0.25, t
}

// sample returns the body part covering (u, v); du is metres per column.
func (f *figure) sample(su, sv, du float64) part {
	sv -= f.p.Lift
	// The upper body is tested in body space, undoing the lean; legs are not.
	u, v := f.toBody(su, sv)

	// Head with short hair.
	dx, dy := (u-f.head.u)/f.hrx, (v-f.head.v)/f.hry
	if dx*dx+dy*dy <= 1 {
		switch {
		case f.hca < -0.25, dy > 0.5:
			return pHair
		case math.Abs(f.hsa) > 0.3 && dx*math.Copysign(1, f.hsa) < -0.25 && dy > -0.3:
			return pHair // back of the head when seen from the side
		case math.Abs(dx) > 0.86 && dy > 0.05:
			return pHair
		}
		return pSkin
	}
	if hy := (v - f.head.v - 0.02) / (f.hry * 1.04); v > f.head.v+0.06 && dx*dx/1.06+hy*hy <= 1 {
		return pHair
	}

	// Bare arms.
	au, av := u, v
	if f.braced {
		au, av = su, sv
	}
	for _, arm := range f.arms {
		if ok, _ := segHit(au, av, arm[0], arm[1], 0.11, du); ok {
			return pSkin
		}
	}

	// White tank top: bare shoulders outside the straps, scoop neck in front.
	if v >= f.hip && v <= f.shoulder {
		k := (v - f.hip) / (f.shoulder - f.hip)
		half := f.halfHip + (f.halfTop-f.halfHip)*k
		au := math.Abs(u)
		if au <= half && !(v > f.shoulder-0.05 && au > half*0.8) {
			switch {
			case v > f.shoulder-0.12 && au > half*0.62:
				return pSkin
			case f.ca > 0.3 && v > f.shoulder-0.14 && math.Abs(u-f.sa*0.05) < 0.09*f.ca:
				return pSkin
			}
			return pTank
		}
	}

	// Jeans and shoes.
	if v >= f.hip-0.1 && v < f.hip && math.Abs(u) <= f.halfHip {
		return pPants
	}
	if ok, _ := segHit(u, v, vec2{f.head.u * 0.5, f.head.v - f.hry}, vec2{0, f.shoulder}, 0.1, du); ok {
		return pSkin // neck
	}
	u, v = su, sv
	for _, leg := range f.legs {
		foot := leg[1]
		sx := (u - foot.u - f.sa*0.05) / (0.05 + 0.07*math.Abs(f.sa))
		sy := (v - foot.v + 0.01) / 0.05
		if sx*sx+sy*sy <= 1 {
			return pShoe
		}
	}
	for _, leg := range f.legs {
		if ok, _ := segHit(u, v, leg[0], leg[1], 0.13, du); ok {
			return pPants
		}
	}
	return pNone
}

// cell colours a body part, lit by the room but never so dark it disappears.
func (f *figure) cell(pt part, e *env) Cell {
	var base RGB
	var mono byte
	switch pt {
	case pHair:
		base, mono = colHair, '@'
	case pSkin:
		base, mono = colSkin, 0 // outlined later
	case pTank:
		base, mono = colTank, ':'
	case pPants:
		base, mono = colPants, '#'
	case pShoe:
		base, mono = colShoe, 'm'
	}
	pos := vec3{f.p.X, floorY + 1.1, f.p.Z}
	n := vec3{f.sa, 0, -f.ca}
	lit := e.shade(base, pos, n)
	floorC := base.Scale(0.82)
	c := RGB{max(lit.R, floorC.R), max(lit.G, floorC.G), max(lit.B, floorC.B)}
	return Cell{Ch: ' ', Mono: mono, BG: c}
}

type feature struct {
	col, row int
	text     string
	fg, bg   RGB
	keepBG   bool
	bold     bool
	onPerson bool  // only drawn where the figure is actually visible
	onSurf   uint8 // only drawn over this surface, when non-zero
}

// features returns the face and speech-bubble details, which are placed by
// projection rather than sampled so they survive at any size.
func (f *figure) features(cam camera) []feature {
	p := f.p
	du := p.Z / cam.f
	at := func(u, v float64) (int, int) {
		s := f.toScreen(vec2{u, v})
		c, r := cam.project(vec3{p.X + s.u, floorY + p.Lift + s.v, p.Z})
		return int(math.Floor(c)), int(math.Floor(r))
	}
	var out []feature
	switch {
	case f.hca > 0.85 && p.Eyes != EyesDots:
		out = append(out, f.face(at, du)...)
	case f.hca > -0.05:
		// Side-on or not looking at you: small dot eyes only.
		for _, side := range [2]float64{-1, 1} {
			if f.hca < 0.55 && side*f.hsa <= 0 {
				continue // the far eye is hidden when seen from the side
			}
			c, r := at(f.head.u+f.hsa*0.1+side*0.075*f.hca, f.head.v)
			out = append(out, feature{col: c, row: r, text: ".", fg: colPupil, keepBG: true, onPerson: true})
		}
	}
	if p.Tag != "" {
		tag := "[" + p.Tag + "]"
		c, r := at(f.head.u, f.head.v+f.hry+0.14)
		out = append(out, feature{col: c - len(tag)/2, row: r, text: tag, fg: RGB{20, 14, 12}, bg: colTag, bold: true})
	}
	if p.Bubble != "" {
		c, r := at(f.head.u+0.3, f.head.v+f.hry+0.02)
		out = append(out, feature{col: c, row: r, text: " " + p.Bubble + " ", fg: colPupil, bg: colBubble, bold: true})
	}
	return out
}

// face draws the expression when he is looking straight at you. It is laid
// out in cell space (brows, eyes, blush, mouth) rather than sampled, so the
// stare, the smirk and the wink stay crisp at any terminal size.
func (f *figure) face(at func(u, v float64) (int, int), du float64) []feature {
	p := f.p
	// Laid out upright from the centre of the head, even when his body leans,
	// so the features always land on the face.
	cx, er := at(f.head.u+f.hsa*0.1, f.head.v-0.02)
	mr := er + max(1, int(math.Round(0.1/(2*du))))
	fw := 2 * f.hrx / du // face width in columns
	k := max(1, int(math.Round(0.075/du)))
	ew := min(max(1, int(math.Round(fw/6))), 3)
	eyeCol := [2]int{cx - k - ew + 1, cx + k} // screen-left, screen-right
	winking := p.Eyes == EyesWink
	sly := winking || p.Mouth == MouthSmirk

	var out []feature
	put := func(col, row int, s string, fg RGB, bg *RGB, bold bool) {
		ft := feature{col: col, row: row, text: s, fg: fg, keepBG: bg == nil, bold: bold, onPerson: true}
		if bg != nil {
			ft.bg = *bg
		}
		out = append(out, ft)
	}
	brow := colHair.Scale(0.5)
	white, blush := colEyeW, RGB{236, 120, 120}

	// Brows: flat for the stare; the left one arches for the smirk and wink.
	for i, c := range eyeCol {
		b := repeat('_', ew)
		if sly && i == 0 {
			b = [...]string{"^", "/\\", "/^\\"}[ew-1]
		}
		put(c, er-1, b, brow, nil, true)
	}

	// Eyes: whites with pupils turned in toward you; the right one winks shut.
	for i, c := range eyeCol {
		if winking && i == 1 {
			put(c, er, repeat('-', ew), colPupil, nil, true)
			continue
		}
		s := "o"
		switch ew {
		case 2:
			s = "o "
			if i == 0 {
				s = " o"
			}
		case 3:
			s = " O "
		}
		put(c, er, s, colPupil, &white, true)
	}

	// Cheeks flush during the wink.
	if winking && mr > er+1 {
		put(eyeCol[0]-1, er+1, " ", colPupil, &blush, false)
		put(eyeCol[1]+ew, er+1, " ", colPupil, &blush, false)
	}

	mw := min(max(2, int(math.Round(0.12/du))), max(2, int(fw)-3))
	switch {
	case winking:
		// A big, bold grin, teeth and all.
		mw = min(max(3, int(math.Round(0.17/du))), max(3, int(fw)-1))
		if mw < 3 {
			put(cx-1, mr, "\\/", colLips, nil, true)
			break
		}
		start := cx - mw/2
		put(start, mr, "\\", colLips, nil, true)
		put(start+1, mr, repeat('_', mw-2), colLips, &white, true)
		put(start+mw-1, mr, "/", colLips, nil, true)
	case p.Mouth == MouthSmirk:
		// Lopsided: flat on the left, curling up on the right.
		put(cx-mw/2+1, mr, repeat('_', mw-1)+"/", colLips, nil, true)
	case p.Mouth == MouthFlat:
		put(cx-mw/2, mr, repeat('-', mw), colLips, nil, true)
	}

	if p.Sparkle {
		out = append(out,
			feature{col: eyeCol[1] + ew + 1, row: er - 2, text: "*", fg: RGB{255, 255, 255}, keepBG: true, bold: true},
			feature{col: eyeCol[1] + ew + 2, row: er - 1, text: "+", fg: RGB{255, 240, 180}, keepBG: true, bold: true})
	}
	return out
}

func repeat(b byte, n int) string {
	s := make([]byte, n)
	for i := range s {
		s[i] = b
	}
	return string(s)
}

// pad centres s in a field of width n.
func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	l := (n - len(s)) / 2
	return repeat(' ', l) + s + repeat(' ', n-len(s)-l)
}
