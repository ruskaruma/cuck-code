package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ruskaruma/cuck-code/internal/config"
	"github.com/ruskaruma/cuck-code/internal/shellhook"
	"github.com/ruskaruma/cuck-code/internal/tty"
)

var subcommands = map[string]func(args []string, cfg config.Config) int{
	"setup":     cmdSetup,
	"uninstall": cmdUninstall,
	"hook":      cmdHook,
	"agents":    cmdAgents,
}

func hookAgents(cfg config.Config) []string {
	return shellhook.Agents(cfg.HookAgents, cfg.HookExclude)
}

// cmdHook prints the shell code the startup files evaluate.
func cmdHook(args []string, cfg config.Config) int {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: cuck hook <%s>\n", strings.Join(shellhook.Shells, "|"))
		return 2
	}
	s, err := shellhook.Script(args[0], hookAgents(cfg))
	if err != nil {
		fmt.Fprintln(os.Stderr, "cuck:", err)
		return 2
	}
	fmt.Print(s)
	return 0
}

// cmdAgents lists the wrapped agents and where the hook is installed.
func cmdAgents(_ []string, cfg config.Config) int {
	fmt.Println("Agents that play the intro when typed directly:")
	for _, a := range hookAgents(cfg) {
		where := "(not installed)"
		if p, err := exec.LookPath(a); err == nil {
			where = p
		}
		fmt.Printf("  %-14s %s\n", a, where)
	}
	fmt.Println()
	any := false
	for _, system := range []bool{false, true} {
		for _, t := range shellhook.Targets(system) {
			if t.Installed() {
				fmt.Printf("Hook installed in %s (%s)\n", tildify(t.Path), t.Shell)
				any = true
			}
		}
	}
	if !any {
		fmt.Println("Hook not installed. Run `cuck setup` to get cucked.")
	}
	return 0
}

func isRoot() bool { return runtime.GOOS != "windows" && os.Geteuid() == 0 }

func truthyEnv(k string) bool { return truthy(os.Getenv(k)) }

// cmdSetup asks the question and installs the shell hook.
func cmdSetup(args []string, cfg config.Config) int {
	fs := flag.NewFlagSet("cuck setup", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "don't ask, just do it")
	fs.BoolVar(yes, "y", false, "shorthand for --yes")
	system := fs.Bool("system", false, "hook every user on this machine (needs root/Administrator)")
	userOnly := fs.Bool("user", false, "hook only the current user, even when running as root")
	fromNPM := fs.Bool("npm", false, "invoked by the npm postinstall script")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: cuck setup [--yes] [--system | --user]\n\nMakes typing an agent's name directly (claude, codex, cursor-agent, ...) play the intro first.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *fromNPM {
		// Never fail or hang a package install: skip quietly when there is
		// nobody to ask, and don't re-ask on upgrades.
		global := os.Getenv("npm_config_global") == "true" || os.Getenv("npm_config_location") == "global"
		if !global || truthyEnv("CI") || truthyEnv("CUCK_SKIP_SETUP") {
			return 0
		}
	}

	sys := *system || (isRoot() && !*userOnly)
	targets := shellhook.Targets(sys)
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "cuck: no supported shell found (bash, zsh, fish or PowerShell)")
		if *fromNPM {
			return 0
		}
		return 1
	}

	if *fromNPM {
		refreshed := false
		for _, t := range targets {
			if t.Installed() {
				_ = t.Install() // upgrade in place
				refreshed = true
			}
		}
		if refreshed {
			return 0
		}
	}

	in, out := io.Reader(os.Stdin), io.Writer(os.Stdout)
	if *fromNPM || !tty.IsTerminal(os.Stdin) {
		cin, cout, err := tty.OpenConsole()
		switch {
		case err == nil:
			defer cin.Close()
			in, out = cin, cout
		case *fromNPM:
			fmt.Fprintln(os.Stderr, "cuck-code installed. Run `cuck setup` to get cucked.")
			return 0
		case !*yes:
			fmt.Fprintln(os.Stderr, "cuck: no terminal to ask on; rerun with --yes")
			return 1
		}
	}

	pitch(out, targets, sys, cfg)
	if !*yes {
		// The answer goes on its own line: package-manager spinners redraw the current one.
		fmt.Fprint(out, "  "+bold("Do you really want to get cucked?")+" [y/N]\n  > ")
		answer, _ := bufio.NewReader(in).ReadString('\n')
		if a := strings.ToLower(strings.TrimSpace(answer)); a != "y" && a != "yes" {
			fmt.Fprintln(out, "\n  Probably wise. Run `cuck setup` if you change your mind.")
			return 0
		}
	}

	failed := 0
	fmt.Fprintln(out)
	for _, t := range targets {
		if err := t.Install(); err != nil {
			fmt.Fprintf(out, "  x %s: %v\n", tildify(t.Path), err)
			failed++
			continue
		}
		fmt.Fprintf(out, "  + %s\n", tildify(t.Path))
	}
	if failed == len(targets) {
		if !sys {
			return 1
		}
		fmt.Fprintln(out, "\n  Couldn't write the system files. Try `sudo cuck setup --system`, or `cuck setup --user`.")
		return 1
	}
	fmt.Fprintln(out, "\n  Welcome to the chair. Open a new terminal and type `claude` (or any agent).")
	fmt.Fprintln(out, "  Skip it once with `command claude`. Undo everything with `cuck uninstall`.")
	return 0
}

func pitch(out io.Writer, targets []shellhook.Target, sys bool, cfg config.Config) {
	who := "you"
	if u, err := user.Current(); err == nil {
		who = "you (" + filepath.Base(u.Username) + ")"
	}
	if sys {
		who = "every user on this machine"
	}
	var found []string
	for _, a := range hookAgents(cfg) {
		if _, err := exec.LookPath(a); err == nil {
			found = append(found, a)
		}
	}
	var files []string
	for _, t := range targets {
		files = append(files, tildify(t.Path))
	}

	fmt.Fprintln(out)
	banner(out)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  From now on, when you type any of these, you watch from the chair first:")
	fmt.Fprintln(out)
	line := "   "
	for _, a := range hookAgents(cfg) {
		if len(line)+len(a)+1 > 76 {
			fmt.Fprintln(out, line)
			line = "   "
		}
		line += " " + a
	}
	fmt.Fprintln(out, line)
	fmt.Fprintln(out)
	if len(found) > 0 {
		fmt.Fprintf(out, "  Installed here:  %s\n", strings.Join(found, ", "))
	}
	fmt.Fprintf(out, "  Applies to:      %s\n", who)
	fmt.Fprintf(out, "  Shell files:     %s\n", strings.Join(files, ", "))
	fmt.Fprintln(out, "  Only interactive shells are affected; scripts and IDEs run agents as usual.")
	fmt.Fprintln(out)
}

// cmdUninstall removes the shell hook from every place it was installed.
func cmdUninstall(args []string, _ config.Config) int {
	fs := flag.NewFlagSet("cuck uninstall", flag.ContinueOnError)
	fs.Bool("yes", false, "accepted for symmetry with setup; uninstall never asks")
	fs.Bool("y", false, "")
	quiet := fs.Bool("quiet", false, "print nothing")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	removed, failed := 0, 0
	for _, system := range []bool{false, true} {
		for _, t := range shellhook.Targets(system) {
			if !t.Installed() {
				continue
			}
			if err := t.Remove(); err != nil {
				failed++
				if !*quiet {
					fmt.Fprintf(os.Stderr, "  x %s: %v\n", tildify(t.Path), err)
				}
				continue
			}
			removed++
			if !*quiet {
				fmt.Printf("  - %s\n", tildify(t.Path))
			}
		}
	}
	if *quiet {
		return 0
	}
	switch {
	case failed > 0:
		fmt.Fprintln(os.Stderr, "\n  Some system files need root: sudo cuck uninstall")
		return 1
	case removed == 0:
		fmt.Println("Nothing to undo: the hook isn't installed.")
	default:
		fmt.Println("\n  You're out of the chair. Open a new terminal for it to take effect.")
	}
	return 0
}

// bold emphasises s where the terminal is known to understand escapes.
func bold(s string) string {
	if os.Getenv("NO_COLOR") != "" || (runtime.GOOS == "windows" && os.Getenv("WT_SESSION") == "") {
		return s
	}
	return "\x1b[1m" + s + "\x1b[0m"
}

func tildify(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}
