package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ruskaruma/cuck-code/internal/config"
	"github.com/ruskaruma/cuck-code/internal/shellhook"
)

// `sudo cuck setup --system` hooks every user; doctor, run by one of them,
// must see that hook and not tell them to install another.
func TestDoctorSeesSystemWideHook(t *testing.T) {
	dir := t.TempDir()
	user := shellhook.Target{Shell: "bash", Path: filepath.Join(dir, "home", ".bashrc")}
	system := shellhook.Target{Shell: "bash", Path: filepath.Join(dir, "etc", "bash.bashrc")}
	if err := system.Install("/usr/local/bin/cuck"); err != nil {
		t.Fatal(err)
	}
	leftovers := func(sys bool) []shellhook.Target {
		if sys {
			return []shellhook.Target{system}
		}
		return []shellhook.Target{user}
	}
	got := hookedTargets(leftovers)
	if len(got) != 1 || got[0].Path != system.Path {
		t.Errorf("hookedTargets = %v, want only %s", got, system.Path)
	}
}

func TestSoundStatusWhenSoundIsOff(t *testing.T) {
	off := false
	t.Setenv("CUCK_SOUND", "")
	os.Unsetenv("CUCK_SOUND")
	for name, cfg := range map[string]config.Config{"config": {Sound: &off}} {
		if got := soundStatus(cfg); !strings.Contains(got, "off") {
			t.Errorf("%s: soundStatus = %q, want it to say sound is off", name, got)
		}
	}
	t.Setenv("CUCK_SOUND", "0")
	if got := soundStatus(config.Config{}); !strings.Contains(got, "off") {
		t.Errorf("CUCK_SOUND=0: soundStatus = %q, want it to say sound is off", got)
	}
}
