package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/ruskaruma/cuck-code/internal/animation"
	"github.com/ruskaruma/cuck-code/internal/tty"
)

// The logo as pixel art: the winking armchair and the CUCK CODE wordmark,
// the same artwork as docs/assets/logo.svg.
var chairArt = []string{
	".................*.",
	"....##########..*.*",
	"...############..*.",
	"...#t###t###t###...",
	"...##e#####ww###...",
	"...#t##m###m##t#...",
	"...#####mmm#####...",
	".aa#t###t###t##aa..",
	"aAAA##########AAAa.",
	"AAAA##########AAAA.",
	"AAAASSSSSSSSSSAAAA.",
	"AAAASSSSSSSSSSAAAA.",
	"AAAAAAAAAAAAAAAAAA.",
	"AAAAAAAAAAAAAAAAAA.",
	".LL............LL..",
	".LL............LL..",
}

var (
	logoCream = animation.RGB{R: 251, G: 233, B: 215}
	logoRed   = animation.RGB{R: 224, G: 68, B: 77}

	chairColors = map[byte]animation.RGB{
		'#': {R: 179, G: 38, B: 47}, 't': {R: 125, G: 24, B: 32}, 'A': {R: 156, G: 32, B: 41},
		'a': {R: 217, G: 84, B: 91}, 'S': {R: 138, G: 28, B: 37}, 'L': {R: 59, G: 36, B: 25},
		'e': logoCream, 'w': logoCream, 'm': logoCream, '*': {R: 255, G: 215, B: 106},
	}

	logoFont = map[byte][7]string{
		'C': {".###.", "#...#", "#....", "#....", "#....", "#...#", ".###."},
		'U': {"#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
		'K': {"#...#", "#..#.", "#.#..", "##...", "#.#..", "#..#.", "#...#"},
		'O': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
		'D': {"####.", "#...#", "#...#", "#...#", "#...#", "#...#", "####."},
		'E': {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
	}
)

// logoPixels composes the chair and, if wide is set, the wordmark beside it.
// A nil entry is a transparent pixel.
func logoPixels(wide bool) [][]*animation.RGB {
	const rows = 16
	grid := make([][]*animation.RGB, rows)
	put := func(x, y int, c animation.RGB) {
		for len(grid[y]) <= x {
			grid[y] = append(grid[y], nil)
		}
		grid[y][x] = &c
	}
	for y, row := range chairArt {
		for x := 0; x < len(row); x++ {
			if c, ok := chairColors[row[x]]; ok {
				put(x, y, c)
			}
		}
	}
	if wide {
		x0 := len(chairArt[0]) + 3
		for _, word := range []struct {
			text  string
			color animation.RGB
		}{{"CUCK", logoRed}, {"CODE", logoCream}} {
			for i := 0; i < len(word.text); i++ {
				for y, row := range logoFont[word.text[i]] {
					for x := 0; x < len(row); x++ {
						if row[x] == '#' {
							put(x0+x, 5+y, word.color)
						}
					}
				}
				x0 += 6
			}
			x0 += 3
		}
	}
	return grid
}

// banner prints the logo at the top of `cuck setup`. It uses half-block
// characters, two pixel rows per line of text, and falls back to a plain
// title when the terminal can't show colour or Unicode.
func banner(out io.Writer) {
	mode := animation.DetectColorMode(os.Getenv, runtime.GOOS == "windows" && os.Getenv("WT_SESSION") != "")
	cols := 0
	if f, ok := out.(*os.File); ok {
		cols = tty.Width(f)
	}
	if mode == animation.ModeNone || mode == animation.Mode16 || !unicodeOK() || (cols > 0 && cols < 24) {
		fmt.Fprintln(out, "  "+bold("CUCK CODE"))
		return
	}
	grid := logoPixels(cols == 0 || cols >= 78)
	at := func(x, y int) *animation.RGB {
		if x < len(grid[y]) {
			return grid[y][x]
		}
		return nil
	}
	width := 0
	for _, row := range grid {
		width = max(width, len(row))
	}
	var b strings.Builder
	for y := 0; y < len(grid); y += 2 {
		b.WriteString("  ")
		for x := 0; x < width; x++ {
			top, bottom := at(x, y), at(x, y+1)
			switch {
			case top == nil && bottom == nil:
				b.WriteString(" ")
			case bottom == nil:
				b.WriteString(mode.Paint(*top, false) + "▀\x1b[0m")
			case top == nil:
				b.WriteString(mode.Paint(*bottom, false) + "▄\x1b[0m")
			default:
				b.WriteString(mode.Paint(*top, false) + mode.Paint(*bottom, true) + "▀\x1b[0m")
			}
		}
		b.WriteString("\n")
	}
	fmt.Fprint(out, b.String())
}

// unicodeOK reports whether the terminal is likely to render block characters.
func unicodeOK() bool {
	if runtime.GOOS == "windows" {
		return os.Getenv("WT_SESSION") != ""
	}
	if runtime.GOOS == "darwin" {
		return true
	}
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := strings.ToLower(os.Getenv(k)); v != "" {
			return strings.Contains(v, "utf-8") || strings.Contains(v, "utf8")
		}
	}
	return false
}
