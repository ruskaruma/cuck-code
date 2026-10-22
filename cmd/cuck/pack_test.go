package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/ruskaruma/cuck-code/internal/config"
)

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPackFromAFolder(t *testing.T) {
	t.Setenv("CUCK_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	src := filepath.Join(t.TempDir(), "My Pirate Pack!")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, src, map[string]string{
		"pack.json": `{"name": "Pirate", "her": {"arr.wav": "arr!", "../up.wav": "no", "x.mp3": "no"}, "wink": {"booty.wav": "who got the good booty?"}}`,
		"arr.wav":   "RIFF....WAVE",
		"booty.wav": "RIFF....WAVE",
		"extra.wav": "not named in pack.json",
		"notes.txt": "nor this",
	})
	if code := packAdd(src); code != 0 {
		t.Fatalf("packAdd = %d", code)
	}
	entries, err := os.ReadDir(filepath.Join(packsDir(), "my-pirate-pack"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	sort.Strings(got)
	if want := []string{"arr.wav", "booty.wav", "pack.json"}; len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("installed %v, want %v", got, want)
	}
	if packList() != 0 {
		t.Error("list failed")
	}
	if packRemove("my-pirate-pack") != 0 {
		t.Error("remove failed")
	}
	if _, err := os.Stat(filepath.Join(packsDir(), "my-pirate-pack")); !os.IsNotExist(err) {
		t.Error("the pack is still there")
	}
	if packRemove("../..") == 0 {
		t.Error("removed something outside the packs folder")
	}
}

func TestPackRefusesWhatItShouldnt(t *testing.T) {
	t.Setenv("CUCK_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	for name, files := range map[string]map[string]string{
		"no pack.json":   {"arr.wav": "x"},
		"no clips":       {"pack.json": `{"her": {"arr.mp3": "x"}}`},
		"a missing clip": {"pack.json": `{"her": {"arr.wav": "x"}}`},
		"broken json":    {"pack.json": `{"her": `},
	} {
		src := t.TempDir()
		writeFiles(t, src, files)
		if packAdd(src) == 0 {
			t.Errorf("%s: installed anyway", name)
		}
	}
	src := t.TempDir()
	writeFiles(t, src, map[string]string{"pack.json": `{"her": {"link.wav": "x"}}`})
	if err := os.Symlink("/etc/passwd", filepath.Join(src, "link.wav")); err == nil && packAdd(src) == 0 {
		t.Error("followed a symlink out of the pack")
	}
}

func TestRecordGIF(t *testing.T) {
	out := filepath.Join(t.TempDir(), "intro.gif")
	if code := cmdRecord([]string{"codex", "-o", out, "-size", "40x14", "-fps", "5"}, config.Config{}); code != 0 {
		t.Fatalf("record = %d", code)
	}
	b, err := os.ReadFile(out)
	if err != nil || len(b) < 1000 || string(b[:6]) != "GIF89a" {
		t.Fatalf("not a GIF: %d bytes, %v", len(b), err)
	}
	if cmdRecord([]string{"-o", "x.webm"}, config.Config{}) == 0 || cmdRecord([]string{"-size", "9x9"}, config.Config{}) == 0 {
		t.Error("record accepted bad options")
	}
}
