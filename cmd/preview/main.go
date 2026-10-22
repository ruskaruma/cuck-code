// Command preview renders the Cuck Code intro to images without a terminal:
// a PNG of a single moment, or an animated GIF of the whole sequence. It is a
// development tool (used for the README demo) and is not part of the cuck binary.
//
//	go run ./cmd/preview -t 5.4 -o frame.png
//	go run ./cmd/preview -gif -o docs/assets/demo.gif
//	go run ./cmd/preview -t 5.4 -text            # plain ASCII to stdout
package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"

	"github.com/ruskaruma/cuck-code/internal/animation"
	"github.com/ruskaruma/cuck-code/internal/record"
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
	o := record.Options{Cols: *w, Rows: *h, FPS: *fps, Agent: *agent, Mode: mode}

	if *text {
		fmt.Print(record.Frame(o, *t, *fx).Plain(mode))
		return
	}
	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if *asGIF {
		err = record.GIF(f, o)
	} else {
		err = png.Encode(f, record.Rasterize(record.Frame(o, *t, *fx), mode))
	}
	if err != nil {
		log.Fatal(err)
	}
}
