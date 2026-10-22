package main

import (
	"flag"
	"os"
	"reflect"
	"testing"

	"github.com/ruskaruma/cuck-code/internal/config"
)

// setStdin points os.Stdin at f for the rest of the test.
func setStdin(t *testing.T, f *os.File) {
	old := os.Stdin
	os.Stdin = f
	t.Cleanup(func() { os.Stdin = old })
}

// `echo prompt | agent` runs the agent non-interactively, and keys typed
// during the show would go nowhere, so it must not wait for the intro.
func TestNoIntroWhenStdinIsPiped(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	setStdin(t, r)
	t.Setenv("CUCK_NO_ANIMATION", "")
	t.Setenv("TERM", "xterm-256color")
	if shouldAnimate(false, config.Config{}) {
		t.Error("shouldAnimate = true with stdin piped, want false")
	}
}

func TestSplitArgs(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	fs.String("agent", "", "")
	fs.String("a", "", "")
	fs.Bool("no-animation", false, "")
	fs.Int("duration", 0, "")
	tests := []struct {
		args      []string
		own, rest []string
	}{
		{[]string{"-a", "claude", "--resume"}, []string{"-a", "claude"}, []string{"--resume"}},
		{[]string{"--agent=claude", "--dangerously-skip-permissions", "--resume"}, []string{"--agent=claude"}, []string{"--dangerously-skip-permissions", "--resume"}},
		{[]string{"--no-animation", "claude", "--resume"}, []string{"--no-animation"}, []string{"claude", "--resume"}},
		{[]string{"--duration", "5000", "-a", "codex", "--", "--full-auto"}, []string{"--duration", "5000", "-a", "codex"}, []string{"--full-auto"}},
		{[]string{"claude", "--no-animation"}, []string{}, []string{"claude", "--no-animation"}},
		{[]string{"--help"}, []string{"--help"}, nil},
	}
	for _, tt := range tests {
		own, rest := splitArgs(fs, tt.args)
		if len(own) == 0 {
			own = []string{}
		}
		if !reflect.DeepEqual(own, tt.own) || !reflect.DeepEqual(rest, tt.rest) {
			t.Errorf("splitArgs(%q) = %q, %q; want %q, %q", tt.args, own, rest, tt.own, tt.rest)
		}
	}
}
