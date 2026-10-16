package shellhook

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
		s, err := Script(sh, []string{"claude", "kiro-cli"}, "/opt/it's/cuck")
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"kiro-cli", "--hooked --agent", "it"} {
			if !strings.Contains(s, want) {
				t.Errorf("%s script missing %q:\n%s", sh, want, s)
			}
		}
		if sh != "fish" && !strings.Contains(s, "__cuck_t_kiro_cli") {
			t.Errorf("%s script doesn't remember alias targets:\n%s", sh, s)
		}
	}
	if _, err := Script("tcsh", nil, ""); err == nil {
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
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	tg := Target{Shell: "zsh", Path: path}

	for i := 0; i < 2; i++ { // installing twice must not duplicate the block
		if err := tg.Install("/usr/local/bin/cuck"); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := os.ReadFile(path)
	if strings.Count(string(b), beginMarker) != 1 || !strings.HasPrefix(string(b), orig) || !tg.Installed() {
		t.Fatalf("bad install:\n%s", b)
	}
	// Editing a startup file must not change its permissions (0600 on Unix;
	// Windows reports its own mode, which must equally be preserved).
	if st, _ := os.Stat(path); st.Mode().Perm() != before.Mode().Perm() {
		t.Errorf("mode changed from %v to %v", before.Mode().Perm(), st.Mode().Perm())
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
	if err := tg.Install("/usr/local/bin/cuck"); err != nil || !tg.Installed() {
		t.Fatal("fish hook not installed", err)
	}
	if err := tg.Remove(); err != nil || tg.Installed() {
		t.Fatal("fish hook not removed", err)
	}
	if err := tg.Remove(); err != nil {
		t.Fatal("second remove should be a no-op", err)
	}
}

// TestHookInRealShells types commands into real interactive bash and zsh
// sessions with the hook loaded, the way a person would at a prompt. On macOS
// this covers bash 3.2, which lacks much of modern bash.
func TestHookInRealShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the bash and zsh hooks are tested on Unix")
	}
	for _, sh := range []string{"bash", "zsh"} {
		if _, err := exec.LookPath(sh); err != nil {
			t.Logf("%s isn't installed; skipping it", sh)
			continue
		}
		t.Run(sh, func(t *testing.T) { testShell(t, sh) })
	}
}

func testShell(t *testing.T, sh string) {
	cases := []struct{ name, before, after, typed, want string }{
		{"plain", "", "", "claude -x", "cuck --hooked --agent claude -- -x"},
		{"dashed name", "", "", "kiro-cli 'two words'", "cuck --hooked --agent kiro-cli -- two words"},
		{"alias above the hook", "alias claude=LOCAL", "", "claude -x", "cuck --hooked --agent LOCAL -- -x"},
		{"alias below the hook", "", "alias claude=LOCAL", "claude -x", "cuck --hooked --agent LOCAL -- -x"},
		{"alias setting a variable", "alias claude='FOO=1 claude'", "", "claude -x", "cuck --hooked --agent FOO=1 claude -- -x"},
		{"hook loaded twice", "alias claude=LOCAL", `eval "$(cat HOOK)"`, "claude -x", "cuck --hooked --agent LOCAL -- -x"},
		{"bypass", "", "", "command claude -x", "claude -x"},
		{"cuck uninstalled", "", "", "rm CUCK; claude -x", "claude -x"},
		{"cuck uninstalled, alias", "alias claude='FOO=1 LOCAL'", "", "rm CUCK; claude -x", "local claude -x FOO=1"},
	}
	for _, c := range cases {
		dir := t.TempDir()
		bin, local := filepath.Join(dir, "bin"), filepath.Join(dir, "local")
		write := func(path, body string, mode os.FileMode) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), mode); err != nil {
				t.Fatal(err)
			}
		}
		cuck := filepath.Join(dir, "cuck", "cuck") // off PATH: only the remembered path finds it
		write(cuck, "#!/bin/sh\necho \"cuck $*\"\n", 0o755)
		write(filepath.Join(bin, "claude"), "#!/bin/sh\necho \"claude $*${FOO:+ FOO=$FOO}\"\n", 0o755)
		write(filepath.Join(bin, "kiro-cli"), "#!/bin/sh\necho \"kiro-cli $*\"\n", 0o755)
		write(filepath.Join(local, "claude"), "#!/bin/sh\necho \"local claude $*${FOO:+ FOO=$FOO}\"\n", 0o755)
		hook, err := Script(sh, []string{"claude", "kiro-cli"}, cuck)
		if err != nil {
			t.Fatal(err)
		}
		hookFile := filepath.Join(dir, "hook")
		write(hookFile, hook, 0o644)
		r := strings.NewReplacer("LOCAL", filepath.Join(local, "claude"), "HOOK", hookFile, "CUCK", cuck)
		rc := filepath.Join(dir, ".zshrc")
		write(rc, r.Replace(c.before)+"\neval \"$(cat "+hookFile+")\"\n"+r.Replace(c.after)+"\n", 0o644)

		args := []string{"-i"}
		if sh == "bash" {
			args = []string{"--rcfile", rc, "-i"}
		}
		cmd := exec.Command(sh, args...)
		cmd.Env = []string{"HOME=" + dir, "ZDOTDIR=" + dir, "PATH=" + bin + ":/usr/bin:/bin", "TERM=dumb"}
		cmd.Stdin = strings.NewReader(r.Replace(c.typed) + "\n")
		var stdout, stderr strings.Builder
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		_ = cmd.Run()
		if got, want := strings.TrimSpace(stdout.String()), r.Replace(c.want); got != want {
			t.Errorf("%s: typing %q printed %q, want %q\nstderr: %s", c.name, c.typed, got, want, stderr.String())
		}
		for _, bad := range []string{"not found", "parse error", "syntax error", "defining function", "No such file"} {
			if strings.Contains(stderr.String(), bad) {
				t.Errorf("%s: shell complained (%s):\n%s", c.name, bad, stderr.String())
			}
		}
	}
}
