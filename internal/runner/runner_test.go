package runner

import (
	"reflect"
	"runtime"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		line  string
		extra []string
		argv  []string
		shell string
	}{
		{"claude", nil, []string{"claude"}, ""},
		{"  aider --model sonnet ", []string{"--yes"}, []string{"aider", "--model", "sonnet", "--yes"}, ""},
		{`my-agent --msg "hello world" 'it''s'`, nil, []string{"my-agent", "--msg", "hello world", "its"}, ""},
		{"claude | tee log", nil, nil, "claude | tee log"},
		{"FOO=1 codex $HOME", nil, nil, "FOO=1 codex $HOME"},
		// Forms that shell aliases take.
		{"nocorrect claude", []string{"-c"}, []string{"claude", "-c"}, ""},
		{"command  noglob claude --x", nil, []string{"claude", "--x"}, ""},
		{"~/.claude/local/claude", nil, nil, "~/.claude/local/claude"},
	}
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			line  string
			extra []string
			argv  []string
			shell string
		}{"FOO=1 claude --x", []string{"-c"}, nil, "FOO=1 claude --x"})
	}
	for _, tt := range tests {
		c, err := Parse(tt.line, tt.extra)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.line, err)
		}
		if !reflect.DeepEqual(c.Argv, tt.argv) || c.Shell != tt.shell {
			t.Errorf("Parse(%q) = %q / %q, want %q / %q", tt.line, c.Argv, c.Shell, tt.argv, tt.shell)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, line := range []string{"", "   ", `agent "unterminated`} {
		if _, err := Parse(line, nil); err == nil {
			t.Errorf("Parse(%q): expected error", line)
		}
	}
}

func TestSplitBackslash(t *testing.T) {
	got, err := Split(`a\ b c`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a b", "c"}
	if runtime.GOOS == "windows" {
		want = []string{`a\`, "b", "c"}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Split = %q, want %q", got, want)
	}
}

func TestCheckMissing(t *testing.T) {
	c, _ := Parse("definitely-not-a-real-agent-9f3a", nil)
	if err := c.Check(); err == nil {
		t.Error("expected missing agent error")
	}
}

func TestString(t *testing.T) {
	c, _ := Parse("claude", []string{"--append-system-prompt", "be nice"})
	if got, want := c.String(), "claude --append-system-prompt 'be nice'"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestName(t *testing.T) {
	names := map[string]string{
		"claude --x":                     "claude",
		"~/.claude/local/claude":         "claude",
		"nocorrect /opt/bin/codex | cat": "codex",
	}
	if runtime.GOOS != "windows" { // cmd.exe has no FOO=1 prefix
		names["FOO=1 BAR=2 claude"] = "claude"
	}
	for line, want := range names {
		c, err := Parse(line, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := c.Name(); got != want {
			t.Errorf("Parse(%q).Name() = %q, want %q", line, got, want)
		}
	}
}
