// Command gen synthesises the intro's sound effects into internal/sound/clips.
// It is a development tool (run by `make sounds`); the clips it writes are
// committed and embedded into the cuck binary.
package main

import (
	"encoding/binary"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
)

const rate = 22050

type clip []float64

func seconds(s float64) clip { return make(clip, int(s*rate)) }

// add mixes src into c starting at time at.
func (c clip) add(at float64, src clip, gain float64) {
	o := int(at * rate)
	for i, v := range src {
		if o+i < len(c) {
			c[o+i] += v * gain
		}
	}
}

func env(t, attack, decay float64) float64 {
	if t < attack {
		return t / attack
	}
	return math.Exp(-(t - attack) / decay)
}

// knock: a dull thud on a wooden door.
func knock() clip {
	c := seconds(0.12)
	rng := rand.New(rand.NewSource(1))
	for i := range c {
		t := float64(i) / rate
		e := env(t, 0.002, 0.025)
		c[i] = e * (0.8*math.Sin(2*math.Pi*110*t) + 0.5*math.Sin(2*math.Pi*230*t) + 0.25*(rng.Float64()*2-1)*math.Exp(-t/0.008))
	}
	return c
}

func knocks() clip {
	c := seconds(0.75)
	for i, at := range []float64{0, 0.2, 0.4} {
		c.add(at, knock(), 1-0.1*float64(i))
	}
	return c
}

// creak: a stick-slip squeak, a buzzy tone that wobbles in pitch.
func creak(dur, base, spread float64, seed int64) clip {
	c := seconds(dur)
	rng := rand.New(rand.NewSource(seed))
	phase := 0.0
	for i := range c {
		t := float64(i) / rate
		f := base + spread*math.Sin(2*math.Pi*1.7*t/dur) + 18*math.Sin(2*math.Pi*9*t)
		phase += 2 * math.Pi * f / rate
		// Stick-slip: bursts of a harsh pulse train.
		pulse := math.Pow(math.Abs(math.Sin(phase/2)), 12)*2 - 0.3
		grain := 0.5 + 0.5*math.Sin(2*math.Pi*23*t+rng.Float64()*0.3)
		e := math.Sin(math.Pi*t/dur) * grain
		c[i] = e * pulse * 0.6
	}
	return c
}

// smooch: a wet pop followed by a rising "mwah".
func smooch() clip {
	c := seconds(0.45)
	rng := rand.New(rand.NewSource(3))
	phase := 0.0
	for i := range c {
		t := float64(i) / rate
		pop := (rng.Float64()*2 - 1) * math.Exp(-t/0.006)
		f := 280 + 700*math.Min(1, t/0.25)
		phase += 2 * math.Pi * f / rate
		mwah := math.Sin(phase) * math.Sin(math.Pi*math.Min(1, t/0.4)) * 0.45
		c[i] = pop*0.9 + mwah
	}
	return c
}

// ding: a small bell for the wink.
func ding() clip {
	c := seconds(0.9)
	for i := range c {
		t := float64(i) / rate
		c[i] = env(t, 0.003, 0.25) * (0.6*math.Sin(2*math.Pi*1760*t) + 0.3*math.Sin(2*math.Pi*2640*t) + 0.15*math.Sin(2*math.Pi*4400*t))
	}
	return c
}

// boom: a sub drop for the flash.
func boom() clip {
	c := seconds(1.1)
	rng := rand.New(rand.NewSource(4))
	phase := 0.0
	for i := range c {
		t := float64(i) / rate
		f := 38 + 90*math.Exp(-t/0.08)
		phase += 2 * math.Pi * f / rate
		c[i] = env(t, 0.004, 0.35)*math.Sin(phase)*0.95 + (rng.Float64()*2-1)*math.Exp(-t/0.03)*0.3
	}
	return c
}

func writeWAV(path string, c clip) error {
	peak := 0.0
	for _, v := range c {
		peak = math.Max(peak, math.Abs(v))
	}
	if peak == 0 {
		peak = 1
	}
	data := make([]byte, 2*len(c))
	for i, v := range c {
		binary.LittleEndian.PutUint16(data[2*i:], uint16(int16(v/peak*0.9*32767)))
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := []any{
		[4]byte{'R', 'I', 'F', 'F'}, uint32(36 + len(data)), [4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '}, uint32(16), uint16(1), uint16(1), uint32(rate), uint32(rate * 2), uint16(2), uint16(16),
		[4]byte{'d', 'a', 't', 'a'}, uint32(len(data)),
	}
	for _, v := range h {
		if err := binary.Write(f, binary.LittleEndian, v); err != nil {
			return err
		}
	}
	_, err = f.Write(data)
	return err
}

func main() {
	dir := "internal/sound/clips"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	clips := map[string]clip{
		"knock":  knocks(),
		"door":   creak(0.9, 150, 60, 5),
		"bed":    creak(0.35, 210, 40, 6),
		"smooch": smooch(),
		"ding":   ding(),
		"boom":   boom(),
	}
	for name, c := range clips {
		if err := writeWAV(filepath.Join(dir, name+".wav"), c); err != nil {
			log.Fatal(err)
		}
	}
}
