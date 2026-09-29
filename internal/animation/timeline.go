package animation

import (
	"math"

	"github.com/ruskaruma/cuck-code/internal/sound"
)

// Story beats in seconds at natural pace. Play rescales them to fit the
// requested duration; the transition after the wink always runs in real time.
//
// The premise: she is already in bed. The guy who walks in is your coding
// agent, here to do the work for you. You are the chair.
const (
	tKnock     = 0.45 // the door handle rattles
	tDoorOpen  = 0.70 // the door swings open; the agent is standing there
	tDoorDone  = 1.30
	tEnter     = 1.70 // he walks in
	tThrough   = 2.30
	tAtBed     = 3.60 // he arrives at the foot of the bed
	tClimb     = 4.00 // and climbs on
	tKneeling  = 4.90 // leaning over her
	tKiss      = 5.25 // he kisses her
	tKissEnd   = 5.75
	tLookBack  = 5.80 // pulls back; his head turns to look at you over his shoulder
	tPulled    = 6.05
	tLooking   = 6.15
	tSmirk     = 6.45
	tWink      = 6.85
	tLightsOut = 7.15
	storyLen   = 7.40
	skipTarget = tLookBack // where a key press jumps to
)

// soundCues are the sound effects, in story order.
var soundCues = []struct {
	t   float64
	cue string
}{
	{tKnock, sound.Knock},
	{tDoorOpen, sound.Door},
	{tClimb + 0.25, sound.Bed},
	{tClimb + 0.7, sound.Bed},
	{tKiss, sound.Smooch},
	{tWink, sound.Ding},
	{tWink + 0.05, sound.Voice}, // "who got the good d? who got the good d?"
}

type waypoint struct{ t, x, z float64 }

var entrance = []waypoint{{tEnter, -1.30, 6.45}, {tThrough, -1.22, 5.55}, {tAtBed, 0.0, 3.92}}

// bedPose is his body on the bed: where he is, how high, how far he leans.
type bedPose struct{ t, x, z, lift, kneel, lean float64 }

// onBed choreographs the climb: a knee up, lean over her, all the way in for
// the kiss, hold it, then pull back up to look at you.
var onBed = []bedPose{
	{tClimb, 0.00, 3.92, 0.00, 0.0, 0.0},
	{tKneeling, 0.40, 4.60, 0.33, 0.5, 0.85},
	{tKiss, 0.58, 4.90, 0.35, 1.0, 1.2},
	{tKissEnd, 0.58, 4.90, 0.35, 1.0, 1.2},
	{tPulled, 0.52, 4.80, 0.35, 0.55, 0.8},
}

func bedPoseAt(t float64) bedPose {
	last := onBed[len(onBed)-1]
	if t >= last.t {
		return last
	}
	for i := 1; i < len(onBed); i++ {
		a, b := onBed[i-1], onBed[i]
		if t < b.t {
			k := smoothstep(clamp01((t - a.t) / (b.t - a.t)))
			l := func(x, y float64) float64 { return x + (y-x)*k }
			return bedPose{t, l(a.x, b.x), l(a.z, b.z), l(a.lift, b.lift), l(a.kneel, b.kneel), l(a.lean, b.lean)}
		}
	}
	return last
}

const strideLen = 0.5 // metres per half walk cycle

// walk interpolates along a path, returning position, heading, distance
// covered and a 0..1 stride amplitude that eases in and out at the ends.
func walk(path []waypoint, t float64) (x, z, heading, dist, stride float64) {
	first, last := path[0], path[len(path)-1]
	for i := 1; i < len(path); i++ {
		a, b := path[i-1], path[i]
		dx, dz := b.x-a.x, b.z-a.z
		segLen := math.Hypot(dx, dz)
		h := math.Atan2(dx, -dz)
		if t < b.t || i == len(path)-1 {
			k := clamp01((t - a.t) / (b.t - a.t))
			x, z = a.x+dx*k, a.z+dz*k
			dist += segLen * k
			heading = h
			if i > 1 {
				// Round the corner instead of snapping to the new heading.
				p := path[i-2]
				prev := math.Atan2(a.x-p.x, -(a.z - p.z))
				heading = prev + (h-prev)*smoothstep(clamp01((t-a.t)/0.25))
			}
			break
		}
		dist += segLen
	}
	stride = math.Min(clamp01((t-first.t)/0.2), clamp01((last.t-t)/0.2))
	return
}

// StateAt returns the scene at story time t (seconds). agent is the name of
// the agent being launched; it goes on his name tag and in the captions.
func StateAt(t float64, agent string) State {
	if agent == "" {
		agent = "your agent"
	}
	var st State
	switch {
	case t < 1.6:
		st.Caption = "pov: you are the chair"
	case t >= tEnter && t < tAtBed:
		st.Caption = agent + " is here to do your work for you"
	case t >= tAtBed && t < tLookBack:
		st.Caption = "she's better off without you."
	case t >= tLooking:
		st.Caption = `"relax. i'll take it from here."`
	}

	// She has been waiting for someone competent.
	switch {
	case t >= tEnter+0.2 && t < tAtBed+0.2:
		st.HerLine = "finally. someone who does it right."
	case t >= tClimb+0.1 && t < tKneeling+0.2:
		st.HerLine = "take me to heaven."
	}
	// The bed has opinions too.
	if (t >= tClimb+0.25 && t < tClimb+0.55) || (t >= tClimb+0.7 && t < tKneeling) {
		st.SFX = "*creak*"
	}
	if t >= tKiss-0.05 && t < tKissEnd+0.25 {
		st.Kiss = clamp01((t - tKiss + 0.05) / (tKissEnd + 0.25 - tKiss + 0.05))
	}
	st.LampOff = t >= tLightsOut
	st.Her = herAt(t)

	switch {
	case t < tKnock:
	case t < tDoorOpen:
		st.Door = 0.05 * math.Abs(math.Sin((t-tKnock)*42))
	default:
		st.Door = 1.45 * easeOutCubic(clamp01((t-tDoorOpen)/(tDoorDone-tDoorOpen)))
	}
	if t < tDoorOpen {
		return st
	}

	p := &Person{Eyes: EyesDots, Tag: agent}
	st.Person = p

	here := entrance[len(entrance)-1]
	const faceHer = 2.15 // three-quarters away from you, toward her
	switch {
	case t < tEnter:
		// Silhouetted in the doorway, sizing up the room.
		p.X, p.Z, p.Facing = entrance[0].x, entrance[0].z, 0.35
	case t < tAtBed:
		var h, dist float64
		p.X, p.Z, h, dist, p.Stride = walk(entrance, t)
		p.Phase = dist / strideLen * math.Pi
		p.Facing = h
		if t < tEnter+0.25 {
			p.Facing = 0.35 + (h-0.35)*smoothstep((t-tEnter)/0.25)
		}
	case t < tClimb:
		_, _, h, _, _ := walk(entrance, tAtBed)
		p.X, p.Z = here.x, here.z
		p.Facing = h + (faceHer-h)*smoothstep(clamp01((t-tAtBed)/(tClimb-tAtBed)))
	default:
		// A knee on the bed, lean over her, kiss her, pull back.
		bp := bedPoseAt(t)
		p.X, p.Z, p.Lift, p.Kneel, p.Lean = bp.x, bp.z, bp.lift, bp.kneel, bp.lean
		p.Facing = faceHer
		if t < tKneeling {
			k := clamp01((t - tClimb) / (tKneeling - tClimb))
			p.Stride, p.Phase = 0.6*math.Sin(k*math.Pi), k*3*math.Pi
		}
		if t >= tLookBack {
			// Just the head turns: a look back at you over his shoulder.
			p.Look = true
			p.LookAt = faceHer * (1 - smoothstep(clamp01((t-tLookBack)/(tLooking-tLookBack))))
			if t >= tLooking-0.1 {
				p.Eyes, p.Mouth = EyesOpen, MouthFlat
			}
			if t >= tSmirk {
				p.Mouth = MouthSmirk
			}
			if t >= tWink {
				p.Eyes, p.Sparkle = EyesWink, true
			}
		}
	}
	return st
}

// herAt animates her: a blink, a glance at the door, propping herself up to
// watch him walk in, delight, squirming as he climbs on, and a last look at you.
func herAt(t float64) Her {
	h := Her{Eyes: ".", KneeL: 0.35, KneeR: 0.2}
	// Idle: one foot bouncing, knees rocking lazily.
	h.KneeR += 0.12 * math.Max(0, math.Sin(t*7))
	h.Sway = 0.03 * math.Sin(t*2.2)
	switch {
	case t < tKnock:
		if t > 0.25 && t < 0.35 {
			h.Eyes = "-"
		}
	case t < tDoorOpen:
		h.Look = -1 // what was that?
	case t < tAtBed:
		k := easeOutCubic(clamp01((t - tDoorOpen) / 0.6))
		h.Rise = k
		h.KneeL = 0.35 + 0.45*k // draws a knee up as she sits up to look
		h.Look = -1
		h.Eyes = "o"
		if t > tEnter+0.5 {
			h.Eyes = "."
			// Her eyes follow him across the room, knees rocking side to side.
			h.Look = -1 + 1.2*clamp01((t-tThrough)/(tAtBed-tThrough))
			h.Sway = 0.09 * math.Sin((t-tEnter)*5)
			if t > 2.9 && t < 3.0 {
				h.Eyes = "-"
			}
		}
	case t < tClimb:
		k := smoothstep(clamp01((t - tAtBed) / (tClimb - tAtBed)))
		h.Rise, h.Look, h.Eyes = 1, -0.4, "^"
		h.KneeL, h.KneeR = 0.8+0.2*k, 0.3+0.7*k // both knees up
		h.Sway = 0.09 * math.Sin((t-tEnter)*5) * (1 - k)
	case t >= tKiss-0.1 && t < tLookBack:
		// The kiss: eyes closed, melting into the pillow.
		h.Rise, h.Look, h.Eyes = 0.4, -0.6, "-"
		h.KneeL, h.KneeR = 1, 0.85
	case t < tLookBack:
		// Sinking back into the pillow as he climbs over her, squirming.
		k := clamp01((t - tClimb) / (tKiss - tClimb))
		h.Rise = 1 - 0.6*smoothstep(clamp01((t-tClimb)/(tKneeling-tClimb)))
		h.Look, h.Eyes = -0.4, "^"
		env := math.Sin(math.Pi * k)
		h.Wiggle = math.Sin((t-tClimb)*17) * env
		h.KneeL = 1 - 0.25*math.Max(0, math.Sin((t-tClimb)*11))*env
		h.KneeR = 1 - 0.25*math.Max(0, -math.Sin((t-tClimb)*11))*env
		h.Sway = 0.05 * math.Sin((t-tClimb)*11) * env
	default:
		// She looks at you too, knees still up.
		h.Rise, h.Look, h.Eyes = 0.4, 0, "."
		h.KneeL, h.KneeR, h.Sway = 1, 1, 0
		if t >= tWink {
			h.Eyes = "^"
		}
	}
	return h
}

func clamp01(x float64) float64 { return math.Max(0, math.Min(1, x)) }

func smoothstep(x float64) float64 { return x * x * (3 - 2*x) }

func easeOutCubic(x float64) float64 { return 1 - math.Pow(1-x, 3) }

// StoryLength is the natural-pace length of the story in seconds, up to the wink.
func StoryLength() float64 { return storyLen }
