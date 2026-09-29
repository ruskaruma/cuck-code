package config

import (
	"os"
	"path/filepath"
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
