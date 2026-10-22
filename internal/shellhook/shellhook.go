// Package shellhook makes typing an agent's name directly (claude, codex,
// cursor-agent, ...) go through cuck. It does this with shell functions
// loaded from the user's (or the system's) shell startup files, so it only
// affects interactive shells. Scripts, IDEs and other programs that run the
// agents are left alone.
package shellhook

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// DefaultAgents are the coding agents wrapped by the hook. Commands that are
// not installed are harmless: cuck reports "command not found" like the shell would.
var DefaultAgents = []string{
	"claude",       // Claude Code
	"codex",        // OpenAI Codex CLI
	"cursor-agent", // Cursor CLI
	"cursor",       // Cursor editor launcher
	"gemini",       // Gemini CLI
	"agy",          // Google Antigravity CLI
	"copilot",      // GitHub Copilot CLI
	"opencode",     // OpenCode
	"aider",        // Aider
	"goose",        // Block Goose
	"amp",          // Sourcegraph Amp
	"crush",        // Charm Crush
	"qwen",         // Qwen Code
	"cline",        // Cline CLI
	"kiro-cli",     // Kiro CLI (formerly Amazon Q Developer CLI)
	"droid",        // Factory Droid
	"auggie",       // Augment Code CLI
	"kimi",         // Kimi Code CLI
	"openhands",    // OpenHands CLI
	"kilo",         // Kilo Code CLI
	"kilocode",     // Kilo Code CLI (older name)
	"codebuff",     // Codebuff
	"jules",        // Google Jules CLI
	"junie",        // JetBrains Junie CLI
	"letta",        // Letta Code
	"qodo",         // Qodo CLI
	"plandex",      // Plandex
	"grok",         // Grok CLI
	"windsurf",     // Windsurf editor launcher
}

var validName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Agents returns the default list plus extra, minus exclude, dropping names
// that are not safe to use as shell function names.
func Agents(extra, exclude []string) []string {
	var out []string
	for _, a := range append(slices.Clone(DefaultAgents), extra...) {
		a = strings.TrimSpace(a)
		if a == "cuck" || !validName.MatchString(a) || slices.Contains(exclude, a) || slices.Contains(out, a) {
			continue
		}
		out = append(out, a)
	}
	return out
}

// Shells lists the shells Script supports.
var Shells = []string{"bash", "zsh", "fish", "powershell"}

// Script returns shell code defining one wrapper function per agent. It is
// what `cuck hook <shell>` prints and what the startup files evaluate. self
// is the absolute path of the cuck binary, used when cuck isn't on PATH yet
// (nvm, fnm, volta and friends often add npm's bin directory late or lazily).
//
// Each wrapper:
//   - takes over an existing alias for the agent instead of losing to it
//     (Claude Code's installer adds `alias claude=~/.claude/local/claude`),
//     and launches the agent through that alias's target. Aliases defined
//     later in the startup files are taken over at the first prompt;
//   - falls back to the plain agent if cuck has since been uninstalled.
func Script(shell string, agents []string, self string) (string, error) {
	var b strings.Builder
	switch shell {
	case "bash", "zsh":
		b.WriteString("# Cuck Code: coding agents play the intro first. Bypass once with: command <agent>\n")
		// command -v can name a hashed cuck that has since been deleted, hence the -x checks.
		fmt.Fprintf(&b, "__cuck_bin() { local c; c=$(command -v cuck 2>/dev/null); [ -x \"$c\" ] || c=%s; [ -x \"$c\" ] && printf '%%s\\n' \"$c\"; }\n", shQuote(self))
		// The alias builtin (not BASH_ALIASES, which bash 3.2 on macOS lacks)
		// prints the alias already quoted, ready for eval.
		b.WriteString("__cuck_take() {\n  local a\n")
		for _, a := range agents {
			fmt.Fprintf(&b, "  if alias %s >/dev/null 2>&1; then a=$(alias %s); eval \"__cuck_t_%s=${a#*=}\"; unalias %s; fi\n", a, a, varName(a), a)
		}
		b.WriteString("}\n")
		for _, a := range agents {
			v := "__cuck_t_" + varName(a)
			fmt.Fprintf(&b, "%s=${%s:-%s}\n", v, v, a)
		}
		b.WriteString("__cuck_take\n")
		// A function the user already wrote for an agent (claude() { ...; })
		// is kept as __cuck_user_<agent> and still runs, after the intro.
		// typeset -f works in bash 3.2 and zsh; our own wrapper, from a hook
		// loaded twice, is the one that mentions __cuck_bin.
		b.WriteString("__cuck_keep() {\n  local d\n")
		for _, a := range agents {
			fmt.Fprintf(&b, "  d=$(typeset -f %s 2>/dev/null); case $d in ''|*__cuck_bin*) ;; *) eval \"__cuck_user_%s${d#%s}\"; __cuck_f_%s=1 ;; esac\n", a, varName(a), a, varName(a))
		}
		b.WriteString("}\n__cuck_keep\n")
		for _, a := range agents {
			v, n := "__cuck_t_"+varName(a), varName(a)
			// With a kept function: play the intro (Ctrl+C there cancels),
			// then run the function. Otherwise cuck launches the agent itself.
			// Without cuck, run the alias target in a subshell where the
			// wrapper is gone, so `claude` can't call itself.
			fmt.Fprintf(&b, "function %s { local c; c=$(__cuck_bin); if [ -n \"${__cuck_f_%s:-}\" ]; then if [ -n \"$c\" ]; then \"$c\" --hooked --intro-only --agent %s -- \"$@\" || [ $? -ne 130 ] || return 130; fi; __cuck_user_%s \"$@\"; elif [ -n \"$c\" ]; then \"$c\" --hooked --agent \"$%s\" -- \"$@\"; else (unset -f %s; eval \"$%s \\\"\\$@\\\"\"); fi; }\n", a, n, a, n, v, a, v)
		}
		if shell == "zsh" {
			b.WriteString("__cuck_first() { __cuck_take; precmd_functions=(${precmd_functions:#__cuck_first}); }\n")
			b.WriteString("precmd_functions=(${precmd_functions:#__cuck_first} __cuck_first)\n")
		} else {
			b.WriteString("__cuck_first() { __cuck_take; PROMPT_COMMAND=${PROMPT_COMMAND//__cuck_first;/}; }\n")
			b.WriteString("PROMPT_COMMAND=\"__cuck_first;${PROMPT_COMMAND//__cuck_first;/}\"\n")
		}
	case "fish":
		b.WriteString("# Cuck Code: coding agents play the intro first. Bypass once with: command <agent>\n")
		for _, a := range agents {
			fmt.Fprintf(&b, "function %s --wraps %s; if type -q cuck; command cuck --hooked --agent %s -- $argv; else if test -x %s; %s --hooked --agent %s -- $argv; else; command %s $argv; end; end\n",
				a, a, a, fishQuote(self), fishQuote(self), a, a)
		}
	case "powershell", "pwsh":
		b.WriteString("# Cuck Code: coding agents play the intro first. Bypass once with: & (Get-Command <agent> -CommandType Application)\n")
		fmt.Fprintf(&b, "function global:__cuck_bin { $c = Get-Command cuck -ErrorAction SilentlyContinue | Select-Object -First 1; if ($c) { $c.Source } elseif (Test-Path -LiteralPath %s) { %s } }\n", psQuote(self), psQuote(self))
		for _, a := range agents {
			v := "global:__cuck_t_" + varName(a)
			fmt.Fprintf(&b, "if (-not $%s) { $%s = %s }\n", v, v, psQuote(a))
			fmt.Fprintf(&b, "$__a = Get-Alias -Name %s -ErrorAction SilentlyContinue; if ($__a) { $%s = $__a.Definition; Remove-Item -Path Alias:%s -Force -ErrorAction SilentlyContinue }\n", psQuote(a), v, a)
			fmt.Fprintf(&b, "function global:%s { $c = __cuck_bin; if ($c) { & $c --hooked --agent $%s -- @args } else { & (Get-Command %s -CommandType Application | Select-Object -First 1) @args } }\n", a, v, psQuote(a))
		}
	default:
		return "", fmt.Errorf("unsupported shell %q (want %s)", shell, strings.Join(Shells, ", "))
	}
	return b.String(), nil
}

// varName turns an agent name like "cursor-agent" into a valid variable name.
func varName(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, s)
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
func fishQuote(s string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, "'", `\'`).Replace(s) + "'"
}

// SkipIntro reports whether a hooked invocation is a quick, non-interactive
// one (version checks, help, updates, print mode...) that should launch
// immediately instead of sitting through the show.
func SkipIntro(args []string) bool {
	for i, a := range args {
		switch a {
		case "--version", "-v", "-V", "--help", "-h", "-p", "--print":
			return true
		}
		if i == 0 {
			switch a {
			case "version", "help", "update", "upgrade", "login", "logout", "doctor", "mcp", "config", "exec", "install", "uninstall", "auth", "status":
				return true
			}
		}
	}
	return false
}
