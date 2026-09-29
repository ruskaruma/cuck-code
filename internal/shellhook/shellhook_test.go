package shellhook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgents(t *testing.T) {
	got := Agents([]string{"my-agent", "bad name", "claude", "cuck"}, []string{"cursor"})
	has := func(s string) bool {
		for _, a := range got {
			if a == s {
				return true
			}
		}
		return false
	}
	if !has("claude") || !has("my-agent") || has("cursor") || has("bad name") || has("cuck") {
		t.Errorf("Agents() = %v", got)
	}
	n := 0
	for _, a := range got {
		if a == "claude" {
			n++
		}
	}
	if n != 1 {
		t.Errorf("claude listed %d times", n)
	}
}

func TestScript(t *testing.T) {
	for _, sh := range Shells {
		s, err := Script(sh, []string{"claude", "kiro-cli"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(s, "--hooked --agent kiro-cli") {
			t.Errorf("%s script missing wrapper:\n%s", sh, s)
		}
	}
	if _, err := Script("tcsh", nil); err == nil {
		t.Error("expected error for unsupported shell")
	}
}

func TestSkipIntro(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"-p", "hi"}, {"update"}, {"mcp", "list"}} {
		if !SkipIntro(args) {
			t.Errorf("SkipIntro(%q) = false", args)
		}
	}
	for _, args := range [][]string{nil, {"--resume"}, {"fix the login bug"}, {"."}} {
		if SkipIntro(args) {
			t.Errorf("SkipIntro(%q) = true", args)
		}
	}
}

func TestInstallRemoveBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".zshrc")
	orig := "export PATH=$HOME/bin:$PATH\nalias ll='ls -l'\n"
	os.WriteFile(path, []byte(orig), 0o600)
	tg := Target{Shell: "zsh", Path: path}

	for i := 0; i < 2; i++ { // installing twice must not duplicate the block
		if err := tg.Install(); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := os.ReadFile(path)
	if strings.Count(string(b), beginMarker) != 1 || !strings.HasPrefix(string(b), orig) || !tg.Installed() {
		t.Fatalf("bad install:\n%s", b)
	}
	if st, _ := os.Stat(path); st.Mode().Perm() != 0o600 {
		t.Errorf("mode changed to %v", st.Mode().Perm())
	}
	if err := tg.Remove(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(path); string(b) != orig || tg.Installed() {
		t.Fatalf("remove did not restore the file:\n%q", b)
	}
}

func TestInstallOwnFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conf.d", "cuck-code.fish")
	tg := Target{Shell: "fish", Path: path, Own: true}
	if err := tg.Install(); err != nil || !tg.Installed() {
		t.Fatal("fish hook not installed", err)
	}
	if err := tg.Remove(); err != nil || tg.Installed() {
		t.Fatal("fish hook not removed", err)
	}
	if err := tg.Remove(); err != nil {
		t.Fatal("second remove should be a no-op", err)
	}
}
