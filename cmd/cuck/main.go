// Command cuck launches a coding agent, after a short cinematic from the chair.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ruskaruma/cuck-code/internal/animation"
	"github.com/ruskaruma/cuck-code/internal/config"
	"github.com/ruskaruma/cuck-code/internal/runner"
	"github.com/ruskaruma/cuck-code/internal/shellhook"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const defaultAgent = "claude"

const usage = `cuck - launch a coding agent, from the chair.

Usage:
  cuck [flags] [agent [agent args...]]
  cuck [flags] --agent <agent> [-- agent args...]
  cuck setup [--yes] [--system]    make typing any agent's name play the intro
  cuck uninstall                   undo setup
  cuck agents                      list the agents setup wraps
  cuck hook <bash|zsh|fish|powershell>  print the shell hook (used by setup)

Examples:
  cuck                          launch the default agent (claude)
  cuck codex                    launch codex
  cuck aider --model sonnet     arguments after the agent name go to the agent
  cuck -a claude -- --resume    same, with --agent
  cuck 'my-agent --flag'        any command line works
  cuck --no-animation claude    skip straight to the agent

While the intro plays, press any key to skip ahead; press again (or q) to skip it entirely.

Flags:
`

const envHelp = `
Environment:
  CUCK_AGENT          default agent when none is given
  CUCK_NO_ANIMATION   set to 1 to always skip the intro
  CUCK_CONFIG         config file path (default %s)
  NO_COLOR            render the intro without colour
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 {
		if sub, ok := subcommands[args[0]]; ok {
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintln(os.Stderr, "cuck: ignoring config:", err)
			}
			return sub(args[1:], cfg)
		}
	}

	fs := flag.NewFlagSet("cuck", flag.ContinueOnError)
	var agent string
	fs.StringVar(&agent, "agent", "", "agent name, config alias or full command line")
	fs.StringVar(&agent, "a", "", "shorthand for --agent")
	noAnim := fs.Bool("no-animation", false, "skip the intro")
	durationMS := fs.Int("duration", 0, fmt.Sprintf("intro length in milliseconds (%d-%d, default %d)",
		animation.MinDuration.Milliseconds(), animation.MaxDuration.Milliseconds(), animation.DefaultDuration.Milliseconds()))
	fps := fs.Int("fps", 0, fmt.Sprintf("intro frame rate (default %d)", animation.DefaultFPS))
	colorFlag := fs.String("color", "", "colour mode: auto, truecolor, 256, 16 or none (default auto)")
	dryRun := fs.Bool("dry-run", false, "print the command that would be launched and exit")
	showVersion := fs.Bool("version", false, "print the version and exit")
	// --hooked marks a launch from the shell hook, i.e. the user typed the agent's name directly.
	hooked := fs.Bool("hooked", false, "")
	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprint(out, usage)
		fs.VisitAll(func(f *flag.Flag) {
			if f.Name != "hooked" {
				fmt.Fprintf(out, "  -%s\n    \t%s\n", f.Name, f.Usage)
			}
		})
		fmt.Fprintf(out, envHelp, config.Path())
	}
	own, passthrough := splitArgs(fs, args)
	if err := fs.Parse(own); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Println("cuck", version)
		return 0
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cuck: ignoring config:", err)
	}

	// Precedence: --agent, then a positional agent, then $CUCK_AGENT, then the config file.
	rest := append(fs.Args(), passthrough...)
	name := agent
	if name == "" && len(rest) > 0 {
		name, rest = rest[0], rest[1:]
	}
	if name == "" {
		name = os.Getenv("CUCK_AGENT")
	}
	if name == "" {
		name = cfg.Agent
	}
	if name == "" {
		name = defaultAgent
	}

	cmd, err := runner.Parse(cfg.Resolve(name), rest)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cuck:", err)
		return 2
	}
	if *dryRun {
		fmt.Println(cmd.String())
		return 0
	}
	// Fail before the show, not after it.
	if err := cmd.Check(); err != nil {
		if *hooked {
			fmt.Fprintf(os.Stderr, "%s: command not found\n", name) // what the shell would have said
		} else {
			fmt.Fprintln(os.Stderr, "cuck:", err)
		}
		return 127
	}

	if shouldAnimate(*noAnim, cfg) && !(*hooked && shellhook.SkipIntro(rest)) {
		opts, err := animationOptions(cfg, *durationMS, *fps, *colorFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cuck:", err)
			return 2
		}
		opts.Agent = cmd.Name()
		opts.Tagline = "-> " + cmd.Name()
		switch err := animation.Play(opts); {
		case err == nil:
			// The agent takes a moment to boot; say so instead of sitting on a stale screen.
			fmt.Fprintf(os.Stderr, "\x1b[2m%s is taking it from here...\x1b[0m\n", cmd.Name())
		case errors.Is(err, animation.ErrInterrupted):
			return 130
		case errors.Is(err, animation.ErrNotTerminal):
			// Piped or redirected: nobody to watch, launch straight away.
		case err != nil:
			fmt.Fprintln(os.Stderr, "cuck: animation:", err)
		}
	}

	code, err := runner.Exec(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cuck: could not launch %s: %v\n", cmd.Name(), err)
	}
	return code
}

func shouldAnimate(noAnim bool, cfg config.Config) bool {
	if noAnim || truthy(os.Getenv("CUCK_NO_ANIMATION")) || os.Getenv("TERM") == "dumb" {
		return false
	}
	return cfg.Animation == nil || *cfg.Animation
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "0", "false", "no", "off":
		return false
	}
	return true
}

func animationOptions(cfg config.Config, durationMS, fps int, colorFlag string) (animation.Options, error) {
	if durationMS == 0 {
		durationMS = cfg.DurationMS
	}
	if fps == 0 {
		fps = cfg.FPS
	}
	if colorFlag == "" {
		colorFlag = cfg.Color
	}
	mode, auto, err := animation.ParseColorMode(colorFlag)
	if err != nil {
		return animation.Options{}, err
	}
	return animation.Options{
		Duration:  time.Duration(durationMS) * time.Millisecond,
		FPS:       fps,
		Color:     mode,
		ColorAuto: auto,
	}, nil
}

// splitArgs separates cuck's own leading flags from everything meant for the
// agent. Parsing stops at "--", at the first non-flag (the agent's name) or
// at the first flag cuck doesn't know, so `cuck -a claude --resume` passes
// --resume through instead of rejecting it.
func splitArgs(fs *flag.FlagSet, args []string) (own, rest []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			return args[:i], args[i+1:]
		}
		if len(a) < 2 || a[0] != '-' {
			return args[:i], args[i:]
		}
		name, _, hasValue := strings.Cut(strings.TrimLeft(a, "-"), "=")
		f := fs.Lookup(name)
		if f == nil && name != "h" && name != "help" {
			return args[:i], args[i:]
		}
		if f != nil && !hasValue && !isBoolFlag(f) {
			i++ // skip the flag's value
		}
	}
	return args, nil
}

func isBoolFlag(f *flag.Flag) bool {
	b, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && b.IsBoolFlag()
}
