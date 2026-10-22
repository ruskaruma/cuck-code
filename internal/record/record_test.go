package record

import (
	"bytes"
	"encoding/binary"
	"image/gif"
	"math"
	"testing"

	"github.com/ruskaruma/cuck-code/internal/animation"
)

func TestGIF(t *testing.T) {
	var b bytes.Buffer
	o := Options{Cols: 40, Rows: 14, FPS: 5, Agent: "codex", Mode: animation.ModeTrueColor}
	if err := GIF(&b, o); err != nil {
		t.Fatal(err)
	}
	g, err := gif.DecodeAll(&b)
	if err != nil {
		t.Fatal(err)
	}
	want := int(math.Ceil((animation.StoryLength() + animation.FXLength()) * 5))
	if len(g.Image) != want || g.Image[0].Bounds().Dx() != 40*CellW || g.Image[0].Bounds().Dy() != 14*CellH {
		t.Errorf("got %d frames of %v, want %d of %dx%d", len(g.Image), g.Image[0].Bounds(), want, 40*CellW, 14*CellH)
	}
}

// loudness is the soundtrack's RMS from a to b seconds.
func loudness(track []byte, a, b float64) float64 {
	pcm := track[44:]
	sum, n := 0.0, 0
	for i := int(a * Rate); i < int(b*Rate) && 2*i+1 < len(pcm); i++ {
		v := float64(int16(binary.LittleEndian.Uint16(pcm[2*i:]))) / 32768
		sum += v * v
		n++
	}
	return math.Sqrt(sum / float64(max(n, 1)))
}

func TestSoundtrack(t *testing.T) {
	o := Options{Agent: "claude"}
	track := Soundtrack(o, 9)
	if len(track) != 44+2*9*Rate || string(track[:4]) != "RIFF" {
		t.Fatalf("soundtrack is %d bytes", len(track))
	}
	if got := samples(track); len(got) != 9*Rate {
		t.Fatalf("the soundtrack doesn't read back as a WAV: %d samples", len(got))
	}
	var bed, wink float64
	for _, c := range animation.SoundCues() {
		switch c.Name {
		case "bed":
			bed = c.T
		case "voice":
			wink = c.T
		}
	}
	if loudness(track, wink+0.2, wink+1.2) < 0.01 {
		t.Error("the wink line is missing")
	}
	if loudness(track, bed, bed+0.15) < 0.005 {
		t.Error("the bed is missing")
	}
	o.SFW = true
	if quiet := Soundtrack(o, 9); loudness(quiet, bed, bed+0.15) > 0.0001 {
		t.Error("safe for work, the bed should be silent")
	}
}
