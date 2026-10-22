package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ruskaruma/cuck-code/internal/animation"
	"github.com/ruskaruma/cuck-code/internal/config"
	"github.com/ruskaruma/cuck-code/internal/shellhook"
	"github.com/ruskaruma/cuck-code/internal/sound"
)

// selfPath is the absolute, symlink-free path of the running cuck binary.
// The shell hook remembers it in case cuck isn't on PATH when a shell starts.
func selfPath() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// liveCheck starts a fresh interactive shell, exactly as a new terminal
// would, and reports what typing agent resolves to at its first prompt:
// "function" means the hook works. It never touches the user's terminal.
func liveCheck(shell, agent string) (string, error) {
	// -c never shows a prompt, so do the hook's first-prompt step by hand.
	first := `__cuck_first >/dev/null 2>&1; `
	probe := map[string]string{
		"bash": first + `printf '\nCUCKCHECK:%s\n' "$(type -t ` + agent + `)"`,
		"zsh":  first + `printf '\nCUCKCHECK:%s\n' "$(whence -w ` + agent + ` 2>/dev/null)"`,
	}[shell]
	if probe == "" {
		return "", fmt.Errorf("can't check %s automatically", shell)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, shell, "-ic", probe)
	cmd.Stdin = nil // /dev/null: anything in the startup files that asks a question gets EOF
	cmd.WaitDelay = time.Second
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	_ = cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", errors.New("your shell took too long to start")
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "CUCKCHECK:"); ok {
			v = strings.TrimSpace(v)
			if i := strings.LastIndex(v, " "); i >= 0 { // zsh: "claude: function"
				v = v[i+1:]
			}
			if v == "" {
				v = "not found"
			}
			return v, nil
		}
	}
	return "", errors.New("couldn't read the result")
}

// userShell is the shell new terminals start: $SHELL if it's one we hook.
func userShell(targets []shellhook.Target) string {
	sh := filepath.Base(os.Getenv("SHELL"))
	for _, t := range targets {
		if t.Shell == sh {
			return sh
		}
	}
	for _, t := range targets {
		if t.Shell == "bash" || t.Shell == "zsh" {
			return t.Shell
		}
	}
	return ""
}

// activateHint is the command that loads the hook into the current terminal.
func activateHint(shell string) string {
	switch shell {
	case "bash":
		return "source ~/.bashrc"
	case "zsh":
		if d := os.Getenv("ZDOTDIR"); d != "" {
			return "source " + tildify(filepath.Join(d, ".zshrc"))
		}
		return "source ~/.zshrc"
	case "fish":
		return "exec fish"
	case "powershell":
		return ". $PROFILE"
	}
	return ""
}

// psPolicyBlocks reports whether PowerShell's execution policy stops profile
// scripts (and so the hook) from loading. Windows only.
func psPolicyBlocks() (string, bool) {
	if runtime.GOOS != "windows" {
		return "", false
	}
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "Get-ExecutionPolicy").Output()
	if err != nil {
		return "", false
	}
	p := strings.TrimSpace(string(out))
	return p, strings.EqualFold(p, "Restricted") || strings.EqualFold(p, "AllSigned")
}

const psPolicyFix = "Set-ExecutionPolicy -Scope CurrentUser RemoteSigned"

// verify runs the live check after setup and says plainly whether it worked.
func verify(out io.Writer, targets []shellhook.Target, agent string) {
	if p, blocked := psPolicyBlocks(); blocked {
		fmt.Fprintf(out, "\n  ! PowerShell's execution policy is %s, so it won't load the hook.\n    Fix it with:  %s\n", p, bold(psPolicyFix))
	}
	sh := userShell(targets)
	if sh == "" {
		return
	}
	kind, err := liveCheck(sh, agent)
	switch {
	case err != nil:
		fmt.Fprintf(out, "\n  (Couldn't double-check a new %s terminal: %v. `cuck doctor` can help.)\n", sh, err)
	case kind == "function":
		fmt.Fprintf(out, "\n  %s Checked: a new %s terminal plays the intro when you type `%s`.\n", mark(true), sh, agent)
	default:
		fmt.Fprintf(out, "\n  %s A new %s terminal still runs plain `%s` (`type` says: %s). Run `cuck doctor` to see why.\n", mark(false), sh, agent, kind)
	}
}

func mark(ok bool) string {
	switch {
	case ok && unicodeOK():
		return "✓"
	case ok:
		return "[ok]"
	case unicodeOK():
		return "✗"
	}
	return "[x]"
}

// cmdDoctor explains why typing an agent does or doesn't play the intro.
func cmdDoctor(_ []string, cfg config.Config) int {
	problems := 0
	say := func(ok bool, format string, a ...any) {
		if !ok {
			problems++
		}
		fmt.Printf("  %s "+format+"\n", append([]any{mark(ok)}, a...)...)
	}
	fmt.Println()
	fmt.Println("  " + bold("cuck doctor"))
	fmt.Println()

	self := selfPath()
	say(true, "cuck %s at %s", version, tildify(self))
	if p, err := exec.LookPath("cuck"); err == nil {
		say(true, "cuck is on PATH (%s)", tildify(p))
	} else {
		fmt.Printf("  %s cuck isn't on PATH in this shell; the hook falls back to %s\n", mark(true), tildify(self))
	}

	targets := shellhook.Leftovers(isRoot())
	var hooked []shellhook.Target
	for _, t := range targets {
		if t.Installed() {
			hooked = append(hooked, t)
		}
	}
	if len(hooked) == 0 {
		say(false, "the hook isn't installed. Run: %s", bold("cuck setup"))
	} else {
		for _, t := range hooked {
			say(true, "hook installed in %s (%s)", tildify(t.Path), t.Shell)
		}
	}

	agents := hookAgents(cfg)
	var found []string
	for _, a := range agents {
		if p, err := exec.LookPath(a); err == nil {
			found = append(found, a+" ("+tildify(p)+")")
		}
	}
	if len(found) == 0 {
		fmt.Printf("  %s none of the supported agents are on PATH (they may still be reachable through aliases)\n", mark(true))
	} else {
		say(true, "agents found: %s", strings.Join(found, ", "))
	}

	if len(hooked) > 0 {
		seen := map[string]bool{}
		for _, t := range hooked {
			if seen[t.Shell] || (t.Shell != "bash" && t.Shell != "zsh") {
				continue
			}
			seen[t.Shell] = true
			if _, err := exec.LookPath(t.Shell); err != nil {
				continue
			}
			kind, err := liveCheck(t.Shell, agents[0])
			switch {
			case err != nil:
				fmt.Printf("  %s couldn't test a new %s terminal: %v\n", mark(true), t.Shell, err)
			case kind == "function":
				say(true, "a new %s terminal plays the intro for `%s`", t.Shell, agents[0])
			default:
				say(false, "a new %s terminal runs plain `%s` (`type` says: %s): something in your startup files redefines it after the hook. Make sure the cuck-code block is the last thing in %s", t.Shell, agents[0], kind, tildify(t.Path))
			}
		}
		if sh := filepath.Base(os.Getenv("SHELL")); os.Getenv("SHELL") != "" && !seen[sh] && sh != "fish" && sh != "pwsh" && sh != "powershell" {
			fmt.Printf("  %s your login shell is %s, which cuck doesn't hook (bash, zsh, fish and PowerShell are supported)\n", mark(false), sh)
			problems++
		}
	}

	if p, blocked := psPolicyBlocks(); blocked {
		say(false, "PowerShell's execution policy is %s, so the hook can't load. Fix: %s", p, bold(psPolicyFix))
	}

	if haveClaudeCode() && !isRoot() {
		if modInstalled() {
			say(true, "the Claude Code mod is installed (/cuck-quiet mutes it)")
		} else {
			fmt.Printf("  %s the Claude Code mod isn't installed; `cuck setup` adds it\n", mark(true))
		}
	}

	if name := sound.PlayerName(); name != "" {
		say(true, "sound plays through %s", name)
	} else {
		fmt.Printf("  %s no audio player found, so the intro will be silent\n", mark(true))
	}
	mode := animation.DetectColorMode(os.Getenv, runtime.GOOS == "windows")
	fmt.Printf("  %s colours: %s\n", mark(true), mode)

	fmt.Println()
	if problems == 0 {
		fmt.Println("  All good. Already-open terminals need " + bold(activateHint(userShell(hooked))) + " (or a new window).")
		fmt.Println("  Watch it any time with " + bold("cuck demo") + ".")
		return 0
	}
	fmt.Printf("  %d thing(s) to fix above.\n", problems)
	return 1
}

// cmdDemo plays the intro without launching anything, to check it works.
func cmdDemo(args []string, cfg config.Config) int {
	agent := "claude"
	if len(args) > 0 {
		agent = args[0]
	}
	opts, err := animationOptions(cfg, 0, 0, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cuck:", err)
		return 2
	}
	opts.Agent, opts.Tagline = agent, "-> "+agent
	opts.Sound = sound.New(wantSound(cfg, false, false), expandHome(cfg.SoundFile))
	switch err := animation.Play(opts); {
	case errors.Is(err, animation.ErrInterrupted):
		return 130
	case errors.Is(err, animation.ErrNotTerminal):
		fmt.Fprintln(os.Stderr, "cuck: the demo needs a terminal to play in")
		return 1
	case err != nil:
		fmt.Fprintln(os.Stderr, "cuck:", err)
		return 1
	}
	return 0
}
