// Package runner turns an agent command line into a process and hands the
// terminal over to it.
package runner

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Command is a parsed agent invocation.
type Command struct {
	// Argv is the program and its arguments when the command line is simple
	// enough to run directly (the common case: "claude", "aider --model x").
	Argv []string
	// Shell holds the raw command line when it uses shell syntax (pipes,
	// variables, globs...). It is run with sh -c, or cmd /c on Windows.
	Shell string
	// Extra are arguments appended after the command line in shell mode.
	Extra []string
}

// shellChars are the characters that need a real shell to interpret.
const shellChars = "|&;<>()$`*?{}~\n"

// Parse splits line into a Command and appends extra arguments.
func Parse(line string, extra []string) (*Command, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, errors.New("empty agent command")
	}
	line = stripPrecommands(line)
	if strings.ContainsAny(line, shellChars) {
		return &Command{Shell: line, Extra: extra}, nil
	}
	words, err := Split(line)
	if err != nil {
		return nil, err
	}
	if isAssignment(words[0]) && runtime.GOOS != "windows" {
		// FOO=1 claude, as in alias claude='FOO=1 claude': the shell sets the variable.
		return &Command{Shell: line, Extra: extra}, nil
	}
	return &Command{Argv: append(words, extra...)}, nil
}

// precommands are shell keywords that can lead an alias (alias
// claude='nocorrect claude') and mean nothing once cuck runs the command.
var precommands = []string{"command", "nocorrect", "noglob", "exec"}

func stripPrecommands(line string) string {
	for {
		w, rest, ok := strings.Cut(line, " ")
		if !ok || !slices.Contains(precommands, w) {
			return line
		}
		line = strings.TrimLeft(rest, " \t")
	}
}

// isAssignment reports whether w is a shell variable assignment (NAME=value).
func isAssignment(w string) bool {
	name, _, ok := strings.Cut(w, "=")
	if !ok || name == "" || name[0] >= '0' && name[0] <= '9' {
		return false
	}
	for _, r := range name {
		if r != '_' && !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// Split breaks a command line into words, honouring single and double quotes.
// Backslash escapes are recognised outside Windows, where backslash is the
// path separator.
func Split(s string) ([]string, error) {
	var (
		words   []string
		cur     strings.Builder
		inWord  bool
		quote   rune
		escapes = runtime.GOOS != "windows"
	)
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case quote == '"':
			switch {
			case r == '"':
				quote = 0
			case r == '\\' && escapes && i+1 < len(rs) && strings.ContainsRune(`"\$`+"`", rs[i+1]):
				i++
				cur.WriteRune(rs[i])
			default:
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == '\\' && escapes:
			if i+1 < len(rs) {
				i++
				cur.WriteRune(rs[i])
			}
			inWord = true
		case r == ' ' || r == '\t':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated %c quote in %q", quote, s)
	}
	if inWord {
		words = append(words, cur.String())
	}
	if len(words) == 0 {
		return nil, errors.New("empty agent command")
	}
	return words, nil
}

// Check verifies the program exists so a typo fails before the intro plays,
// not after. Shell command lines cannot be checked without running them.
func (c *Command) Check() error {
	if c.Shell != "" {
		return nil
	}
	if _, err := exec.LookPath(c.Argv[0]); err != nil {
		return fmt.Errorf("agent %q not found in PATH", c.Argv[0])
	}
	return nil
}

// Name is a short display name for the agent.
func (c *Command) Name() string {
	if c.Shell != "" {
		for _, w := range strings.Fields(c.Shell) {
			if !isAssignment(w) {
				return filepath.Base(w)
			}
		}
		return "sh"
	}
	return filepath.Base(c.Argv[0])
}

// String renders the command for --dry-run.
func (c *Command) String() string {
	if c.Shell != "" {
		s := c.Shell
		if len(c.Extra) > 0 {
			s += " " + quoteAll(c.Extra)
		}
		return s
	}
	return quoteAll(c.Argv)
}

func quoteAll(args []string) string {
	out := make([]string, len(args))
	for i, a := range args {
		if a == "" || strings.ContainsAny(a, " \t'\"\\"+shellChars) {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		out[i] = a
	}
	return strings.Join(out, " ")
}
