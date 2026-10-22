package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ruskaruma/cuck-code/internal/animation"
	"github.com/ruskaruma/cuck-code/internal/config"
	"github.com/ruskaruma/cuck-code/internal/record"
)

// cmdRecord saves the intro to a file to share: an MP4 with its sound when
// ffmpeg is around, or a GIF.
func cmdRecord(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("cuck record", flag.ContinueOnError)
	out := fs.String("o", "", "the file to write: .mp4 (with sound; needs ffmpeg) or .gif (default cuck-code.mp4, or .gif without ffmpeg)")
	size := fs.String("size", "100x32", "the terminal it plays in, columns x rows")
	fps := fs.Int("fps", 0, "frames a second (default 30 for an MP4, 15 for a GIF)")
	sfw := fs.Bool("sfw", cfg.SFW != nil && *cfg.SFW, "safe for work: cleaner lines and sounds")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: cuck record [-o file] [-size 100x32] [-sfw] [agent]\n\nSaves the intro for agent (default claude) to share.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(reorder(args)); err != nil {
		return 2
	}
	agent := "claude"
	if fs.NArg() > 0 {
		agent = fs.Arg(0)
	}
	var cols, rows int
	if _, err := fmt.Sscanf(strings.ToLower(*size), "%dx%d", &cols, &rows); err != nil || cols < 40 || rows < 14 || cols > 240 || rows > 80 {
		fmt.Fprintln(os.Stderr, "cuck: -size is columns x rows, from 40x14 to 240x80, like 100x32")
		return 2
	}
	path := *out
	if path == "" {
		path = "cuck-code.gif"
		if _, err := exec.LookPath("ffmpeg"); err == nil {
			path = "cuck-code.mp4"
		}
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".gif" && ext != ".mp4" {
		fmt.Fprintln(os.Stderr, "cuck: record to a .gif or an .mp4")
		return 2
	}
	o := record.Options{Cols: cols, Rows: rows, FPS: *fps, Agent: agent, SFW: *sfw, Mode: animation.ModeTrueColor}
	if o.FPS <= 0 {
		o.FPS = map[string]int{".gif": 15, ".mp4": 30}[ext]
	}
	o.FPS = min(max(o.FPS, 5), 60)

	fmt.Printf("  Recording the intro for %s...\n", agent)
	var err error
	if ext == ".mp4" {
		err = record.Video(path, o)
	} else {
		var f *os.File
		if f, err = os.Create(path); err == nil {
			err = record.GIF(f, o)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
	}
	switch {
	case errors.Is(err, record.ErrNoFFmpeg):
		fmt.Fprintf(os.Stderr, "cuck: %v. Install it, or record a GIF: cuck record -o cuck-code.gif\n", err)
		return 1
	case err != nil:
		fmt.Fprintln(os.Stderr, "cuck:", err)
		return 1
	}
	what := "a GIF, without sound"
	if ext == ".mp4" {
		what = "with sound"
	}
	fmt.Printf("  %s Saved %s (%s). Post it.\n", mark(true), tildify(path), what)
	return 0
}

// reorder puts flags before the agent's name, so `cuck record codex -o x.gif` works.
func reorder(args []string) []string {
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case strings.HasPrefix(a, "-") && !strings.Contains(a, "=") && a != "-sfw" && a != "--sfw" && i+1 < len(args):
			flags = append(flags, a, args[i+1])
			i++
		case strings.HasPrefix(a, "-"):
			flags = append(flags, a)
		default:
			rest = append(rest, a)
		}
	}
	return append(flags, rest...)
}

// cmdSFW turns safe-for-work mode on or off for the intro.
func cmdSFW(args []string, cfg config.Config) int {
	return toggle("sfw", args, cfg.SFW != nil && *cfg.SFW,
		"Safe for work: the intro keeps the same story with cleaner lines and sounds.",
		"Safe for work is off. Back to the usual.")
}

// cmdMeetings turns going quiet and safe for work on calls on or off.
func cmdMeetings(args []string, cfg config.Config) int {
	return toggle("meetings", args, cfg.Meetings == nil || *cfg.Meetings,
		"On calls the intro goes quiet and safe for work.",
		"The intro plays as usual on calls now.")
}

// toggle sets a true/false setting in the config file from `on`, `off`, or
// nothing (flip it), and says what it is now.
func toggle(key string, args []string, now bool, on, off string) int {
	switch {
	case len(args) == 0:
		now = !now
	case len(args) == 1 && (args[0] == "on" || args[0] == "off"):
		now = args[0] == "on"
	default:
		fmt.Fprintf(os.Stderr, "usage: cuck %s [on|off]\n", key)
		return 2
	}
	if err := config.Set(key, now); err != nil {
		fmt.Fprintln(os.Stderr, "cuck:", err)
		return 1
	}
	if now {
		fmt.Println("  " + on)
	} else {
		fmt.Println("  " + off)
	}
	return 0
}
