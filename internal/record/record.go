// Package record renders the intro to a file to share: an animated GIF, or
// with ffmpeg, an MP4 with its sound.
package record

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/ruskaruma/cuck-code/internal/animation"
	"github.com/ruskaruma/cuck-code/internal/sound"
)

// Options says what to record.
type Options struct {
	Cols, Rows int // the terminal it plays in
	FPS        int
	Agent      string // on his name tag, in the captions, and his wink line
	SFW        bool   // safe for work: cleaner lines and sounds
	Mode       animation.ColorMode
}

// CellW and CellH are a character cell's size in pixels.
const CellW, CellH = 7, 14

// Frame renders the scene at story time story (seconds at natural pace) or,
// with fx >= 0, the transition fx seconds after the wink.
func Frame(o Options, story, fx float64) *animation.Canvas {
	cv := animation.NewCanvas(o.Cols, o.Rows)
	st := animation.StateAt(story, o.Agent)
	if o.SFW {
		st.Clean()
	}
	animation.Render(cv, &st)
	if fx >= 0 {
		base := cv
		cv = animation.NewCanvas(o.Cols, o.Rows)
		animation.RenderFX(cv, base, fx, int(fx*60), "-> "+o.Agent)
	}
	return cv
}

// Each calls fn with every frame of the intro, at natural pace, in order.
func Each(o Options, fn func(img *image.RGBA) error) error {
	story, fxLen := animation.StoryLength(), animation.FXLength()
	for i := 0; ; i++ {
		ts := float64(i) / float64(o.FPS)
		var cv *animation.Canvas
		switch {
		case ts < story:
			cv = Frame(o, ts, -1)
		case ts < story+fxLen:
			cv = Frame(o, story, ts-story)
		default:
			return nil
		}
		if err := fn(Rasterize(cv, o.Mode)); err != nil {
			return err
		}
	}
}

// Rasterize draws a frame's cells as pixels, CellW by CellH each.
func Rasterize(cv *animation.Canvas, mode animation.ColorMode) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, cv.W*CellW, cv.H*CellH))
	d := &font.Drawer{Dst: img, Face: basicfont.Face7x13}
	for y := 0; y < cv.H; y++ {
		for x := 0; x < cv.W; x++ {
			c := cv.Cells[y*cv.W+x]
			bg, fg := c.BG, c.FG
			if mode == animation.ModeNone {
				bg, fg = animation.RGB{R: 12, G: 12, B: 12}, animation.RGB{R: 200, G: 200, B: 200}
			}
			r := image.Rect(x*CellW, y*CellH, (x+1)*CellW, (y+1)*CellH)
			draw.Draw(img, r, &image.Uniform{color.RGBA{bg.R, bg.G, bg.B, 255}}, image.Point{}, draw.Src)
			if g := c.Glyph(mode); g != ' ' {
				d.Src = &image.Uniform{color.RGBA{fg.R, fg.G, fg.B, 255}}
				d.Dot = fixed.P(x*CellW, y*CellH+11)
				d.DrawString(string(g))
			}
		}
	}
	return img
}

// GIF writes the intro as an animated GIF that holds on its last frame.
func GIF(w io.Writer, o Options) error {
	anim := &gif.GIF{}
	delay := 100 / o.FPS
	err := Each(o, func(img *image.RGBA) error {
		pal := image.NewPaletted(img.Bounds(), paletteFor(img))
		draw.Draw(pal, pal.Rect, img, image.Point{}, draw.Src)
		anim.Image = append(anim.Image, pal)
		anim.Delay = append(anim.Delay, delay)
		return nil
	})
	if err != nil {
		return err
	}
	anim.Delay[len(anim.Delay)-1] = 150
	return gif.EncodeAll(w, anim)
}

// ErrNoFFmpeg is returned by Video when ffmpeg isn't installed.
var ErrNoFFmpeg = errors.New("an MP4 needs ffmpeg, which isn't on your PATH")

// Video writes the intro as an MP4 with its sound, through ffmpeg: frames
// scaled up twice for sharpness, H.264 and AAC, ready to post.
func Video(path string, o Options) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return ErrNoFFmpeg
	}
	dir, err := os.MkdirTemp("", "cuck-record-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	n := 0
	err = Each(o, func(img *image.RGBA) error {
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%05d.png", n)))
		if err != nil {
			return err
		}
		n++
		if err := png.Encode(f, img); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
	if err != nil {
		return err
	}
	seconds := float64(n) / float64(o.FPS)
	audio := filepath.Join(dir, "sound.wav")
	if err := os.WriteFile(audio, Soundtrack(o, seconds+1.5), 0o644); err != nil {
		return err
	}
	args := []string{"-loglevel", "error", "-y",
		"-framerate", fmt.Sprint(o.FPS), "-i", filepath.Join(dir, "%05d.png"), "-i", audio,
		"-vf", "scale=iw*2:ih*2:flags=neighbor,pad=ceil(iw/2)*2:ceil(ih/2)*2,format=yuv420p"}
	args = append(args, videoCodec(ffmpeg)...)
	args = append(args, "-c:a", "aac", "-b:a", "160k", "-shortest", "-movflags", "+faststart", path)
	cmd := exec.Command(ffmpeg, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %v: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	return nil
}

// videoCodec picks an H.264 encoder this ffmpeg has, which is what sites
// take: x264, Cisco's OpenH264 (Fedora's ffmpeg has it and not x264), or the
// system's own on macOS and Windows; MPEG-4 if there's none.
func videoCodec(ffmpeg string) []string {
	out, _ := exec.Command(ffmpeg, "-hide_banner", "-encoders").Output()
	for _, c := range [][]string{
		{"libx264", "-crf", "18"},
		{"libopenh264", "-b:v", "4M"},
		{"h264_videotoolbox", "-b:v", "4M"},
		{"h264_mf", "-b:v", "4M"},
	} {
		if bytes.Contains(out, []byte(" "+c[0]+" ")) {
			return append([]string{"-c:v"}, c...)
		}
	}
	return []string{"-c:v", "mpeg4", "-q:v", "3"}
}

// Rate is the soundtrack's sample rate, the clips' own.
const Rate = 22050

// Soundtrack mixes the intro's sound effects at their moments into a WAV
// `seconds` long: mono, 16-bit, at Rate. The voice is the agent's own line.
func Soundtrack(o Options, seconds float64) []byte {
	mix := make([]float64, int(seconds*Rate))
	for _, c := range animation.SoundCues() {
		name := c.Name
		switch {
		case name == sound.Voice:
			name = sound.VoiceClip(o.Agent, o.SFW)
		case o.SFW && (name == sound.Bed || name == sound.Smooch):
			continue
		}
		data, err := sound.Clip(name)
		if err != nil {
			continue
		}
		at := int(c.T * Rate)
		for i, v := range samples(data) {
			if at+i < len(mix) {
				mix[at+i] += v
			}
		}
	}
	pcm := make([]byte, 2*len(mix))
	for i, v := range mix {
		binary.LittleEndian.PutUint16(pcm[2*i:], uint16(int16(max(-1, min(1, v))*32767)))
	}
	return wav(pcm)
}

// samples reads a mono 16-bit WAV at Rate as values in [-1, 1]; anything
// else reads as silence.
func samples(data []byte) []float64 {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil
	}
	ok := false
	for p := 12; p+8 <= len(data); {
		id, size := string(data[p:p+4]), int(binary.LittleEndian.Uint32(data[p+4:]))
		body := data[p+8 : min(len(data), p+8+size)]
		switch id {
		case "fmt ":
			ok = len(body) >= 16 && binary.LittleEndian.Uint16(body[0:]) == 1 && binary.LittleEndian.Uint16(body[2:]) == 1 &&
				binary.LittleEndian.Uint32(body[4:]) == Rate && binary.LittleEndian.Uint16(body[14:]) == 16
		case "data":
			if !ok {
				return nil
			}
			out := make([]float64, len(body)/2)
			for i := range out {
				out[i] = float64(int16(binary.LittleEndian.Uint16(body[2*i:]))) / 32768
			}
			return out
		}
		p += 8 + size + size%2
	}
	return nil
}

func wav(pcm []byte) []byte {
	var b bytes.Buffer
	b.WriteString("RIFF")
	_ = binary.Write(&b, binary.LittleEndian, uint32(36+len(pcm)))
	b.WriteString("WAVEfmt ")
	// PCM, mono, Rate samples a second of 2 bytes each.
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(Rate), uint32(Rate * 2), uint16(2), uint16(16)} {
		_ = binary.Write(&b, binary.LittleEndian, v)
	}
	b.WriteString("data")
	_ = binary.Write(&b, binary.LittleEndian, uint32(len(pcm)))
	b.Write(pcm)
	return b.Bytes()
}

// paletteFor builds a ≤256 colour palette from the frame's most common colours.
func paletteFor(img *image.RGBA) color.Palette {
	counts := map[color.RGBA]int{}
	for i := 0; i < len(img.Pix); i += 4 * CellW {
		counts[color.RGBA{img.Pix[i], img.Pix[i+1], img.Pix[i+2], 255}]++
	}
	type kv struct {
		c color.RGBA
		n int
	}
	var all []kv
	for c, n := range counts {
		all = append(all, kv{c, n})
	}
	// Partial selection sort for the top 256 is fine at this size.
	pal := color.Palette{}
	for len(pal) < 256 && len(all) > 0 {
		best := 0
		for i := range all {
			if all[i].n > all[best].n {
				best = i
			}
		}
		pal = append(pal, all[best].c)
		all[best] = all[len(all)-1]
		all = all[:len(all)-1]
	}
	return pal
}
