package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("CUCK_CONFIG", path)

	if c, err := Load(); err != nil || c.Agent != "" {
		t.Fatalf("missing file: got %+v, %v", c, err)
	}

	os.WriteFile(path, []byte(`{"agent":"codex","agents":{"yolo":"claude --dangerously-skip-permissions"},"animation":false}`), 0o600)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Agent != "codex" || c.Animation == nil || *c.Animation {
		t.Errorf("unexpected config %+v", c)
	}
	if got := c.Resolve("yolo"); got != "claude --dangerously-skip-permissions" {
		t.Errorf("Resolve(yolo) = %q", got)
	}
	if got := c.Resolve("aider"); got != "aider" {
		t.Errorf("Resolve(aider) = %q", got)
	}

	os.WriteFile(path, []byte(`{"agent":`), 0o600)
	if _, err := Load(); err == nil {
		t.Error("expected error for malformed config")
	}
}

func TestSetKeepsOtherKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	t.Setenv("CUCK_CONFIG", path)
	if err := Set("agent", "codex"); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, []byte(`{"agent":"codex","future_field":42}`), 0o600)
	if err := Set("sound", true); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil || c.Agent != "codex" || c.Sound == nil || !*c.Sound {
		t.Fatalf("got %+v, %v", c, err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), "future_field") {
		t.Errorf("unknown keys were dropped:\n%s", b)
	}
}
