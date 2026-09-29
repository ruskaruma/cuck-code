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
// what `cuck hook <shell>` prints and what the startup files evaluate.
func Script(shell string, agents []string) (string, error) {
	var b strings.Builder
	switch shell {
	case "bash", "zsh":
		b.WriteString("# Cuck Code: coding agents play the intro first. Bypass once with: command <agent>\n")
		for _, a := range agents {
			fmt.Fprintf(&b, "%s() { command cuck --hooked --agent %s -- \"$@\"; }\n", a, a)
		}
	case "fish":
		b.WriteString("# Cuck Code: coding agents play the intro first. Bypass once with: command <agent>\n")
		for _, a := range agents {
			fmt.Fprintf(&b, "function %s --wraps %s; command cuck --hooked --agent %s -- $argv; end\n", a, a, a)
		}
	case "powershell", "pwsh":
		b.WriteString("# Cuck Code: coding agents play the intro first. Bypass once with: & (Get-Command <agent> -CommandType Application)\n")
		for _, a := range agents {
			fmt.Fprintf(&b, "function global:%s { & cuck --hooked --agent %s -- @args }\n", a, a)
		}
	default:
		return "", fmt.Errorf("unsupported shell %q (want %s)", shell, strings.Join(Shells, ", "))
	}
	return b.String(), nil
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
