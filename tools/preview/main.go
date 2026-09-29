// Command preview renders the Cuck Code intro to images without a terminal:
// a PNG of a single moment, or an animated GIF of the whole sequence. It is a
// development tool (used for the README demo) and is not part of the cuck binary.
//
//	go run ./tools/preview -t 5.4 -o frame.png
//	go run ./tools/preview -gif -o assets/demo.gif
//	go run ./tools/preview -t 5.4 -text            # plain ASCII to stdout
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"log"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/ruskaruma/cuck-code/internal/animation"
)

func main() {
	w := flag.Int("w", 100, "terminal columns")
	h := flag.Int("h", 32, "terminal rows")
	t := flag.Float64("t", 6.9, "story time in seconds (ignored with -gif)")
	fx := flag.Float64("fx", -1, "render the transition at this many seconds after the wink")
	out := flag.String("o", "frame.png", "output file")
	asGIF := flag.Bool("gif", false, "render the whole intro as an animated GIF")
	fps := flag.Int("fps", 15, "GIF frame rate")
	text := flag.Bool("text", false, "print the frame as plain ASCII instead of an image")
	mono := flag.Bool("mono", false, "with -text or images: use the monochrome glyph set")
	agent := flag.String("agent", "claude", "agent name shown on the name tag")
	flag.Parse()

	mode := animation.ModeTrueColor
	if *mono {
		mode = animation.ModeNone
	}

	frameAt := func(story, fxT float64) *animation.Canvas {
		cv := animation.NewCanvas(*w, *h)
		st := animation.StateAt(story, *agent)
		animation.Render(cv, &st)
		if fxT >= 0 {
			base := cv
			cv = animation.NewCanvas(*w, *h)
			animation.RenderFX(cv, base, fxT, int(fxT*60), "-> "+*agent)
		}
		return cv
	}

	if *text {
		fmt.Print(frameAt(*t, *fx).Plain(mode))
		return
	}
	if !*asGIF {
		img := rasterize(frameAt(*t, *fx), mode)
		f, err := os.Create(*out)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			log.Fatal(err)
		}
		return
	}

	story, fxLen := animation.StoryLength(), animation.FXLength()
	anim := &gif.GIF{}
	delay := 100 / *fps
	for i := 0; ; i++ {
		ts := float64(i) / float64(*fps)
		var cv *animation.Canvas
		switch {
		case ts < story:
			cv = frameAt(ts, -1)
		case ts < story+fxLen:
			cv = frameAt(story, ts-story)
		default:
			i = -1
		}
		if i < 0 {
			break
		}
		rgba := rasterize(cv, mode)
		pal := image.NewPaletted(rgba.Bounds(), paletteFor(rgba))
		draw.Draw(pal, pal.Rect, rgba, image.Point{}, draw.Src)
		anim.Image = append(anim.Image, pal)
		anim.Delay = append(anim.Delay, delay)
	}
	anim.Delay[len(anim.Delay)-1] = 150
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := gif.EncodeAll(f, anim); err != nil {
		log.Fatal(err)
	}
}

const cellW, cellH = 7, 14

func rasterize(cv *animation.Canvas, mode animation.ColorMode) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, cv.W*cellW, cv.H*cellH))
	d := &font.Drawer{Dst: img, Face: basicfont.Face7x13}
	for y := 0; y < cv.H; y++ {
		for x := 0; x < cv.W; x++ {
			c := cv.Cells[y*cv.W+x]
			bg, fg := c.BG, c.FG
			if mode == animation.ModeNone {
				bg, fg = animation.RGB{R: 12, G: 12, B: 12}, animation.RGB{R: 200, G: 200, B: 200}
			}
			r := image.Rect(x*cellW, y*cellH, (x+1)*cellW, (y+1)*cellH)
			draw.Draw(img, r, &image.Uniform{color.RGBA{bg.R, bg.G, bg.B, 255}}, image.Point{}, draw.Src)
			if g := c.Glyph(mode); g != ' ' {
				d.Src = &image.Uniform{color.RGBA{fg.R, fg.G, fg.B, 255}}
				d.Dot = fixed.P(x*cellW, y*cellH+11)
				d.DrawString(string(g))
			}
		}
	}
	return img
}

// paletteFor builds a ≤256 colour palette from the frame's most common colours.
func paletteFor(img *image.RGBA) color.Palette {
	counts := map[color.RGBA]int{}
	for i := 0; i < len(img.Pix); i += 4 * cellW {
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
