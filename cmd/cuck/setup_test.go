package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// A terminal can exist with nobody at it (docker run -t, ssh -t in a
// script); an npm install asking on it must still finish.
func TestAskGivesUpWhenNobodyAnswers(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	// w stays open until cleanup, so reads block as on an unattended terminal;
	// closing it first ends the abandoned read with EOF.
	t.Cleanup(func() { w.Close(); r.Close() })
	done := make(chan bool, 1)
	go func() { _, ok := ask(r, 50*time.Millisecond); done <- ok }()
	select {
	case ok := <-done:
		if ok {
			t.Error("ask reported an answer nobody gave")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ask still waiting 5s into a 50ms timeout")
	}
}

func TestAskReturnsTheAnswer(t *testing.T) {
	for _, timeout := range []time.Duration{0, time.Second} {
		if a, ok := ask(strings.NewReader("y\n"), timeout); !ok || a != "y\n" {
			t.Errorf("ask(timeout %v) = %q, %v; want \"y\\n\", true", timeout, a, ok)
		}
	}
}

// `command claude` skips aliases, so it can't reach an agent installed as
// one (Claude Code's alias claude=~/.claude/local/claude); the environment
// variable works however the agent is installed.
func TestBypassHintWorksForAliasInstalls(t *testing.T) {
	for shell, want := range map[string]string{
		"bash":       "CUCK_NO_ANIMATION=1 claude",
		"zsh":        "CUCK_NO_ANIMATION=1 claude",
		"fish":       "CUCK_NO_ANIMATION=1 claude",
		"":           "CUCK_NO_ANIMATION=1 claude",
		"powershell": "$env:CUCK_NO_ANIMATION=1; claude; Remove-Item Env:CUCK_NO_ANIMATION",
	} {
		if got := bypassHint(shell, "claude"); got != want {
			t.Errorf("bypassHint(%q) = %q, want %q", shell, got, want)
		}
	}
}
