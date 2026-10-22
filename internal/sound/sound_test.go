package sound

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEveryCueHasAClip(t *testing.T) {
	for _, c := range []string{Knock, Door, Bed, Smooch, Ding, Voice, Boom} {
		b, err := clips.ReadFile("clips/" + c + ".wav")
		if err != nil || len(b) < 1000 || string(b[:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
			t.Errorf("clip %q missing or not a WAV (%d bytes, %v)", c, len(b), err)
		}
	}
}

func TestDisabledIsSilent(t *testing.T) {
	var nilPlayer *Player
	nilPlayer.Play(Ding) // must not panic
	p := New(false, "")
	p.Play(Ding)
	if p.dir != "" {
		t.Error("a disabled player should not touch the disk")
	}
}

func TestExtract(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	p := New(true, "")
	path := p.extract(Voice)
	if path == "" || filepath.Base(path) != "voice.wav" {
		t.Fatalf("extract = %q", path)
	}
	if st, err := os.Stat(path); err != nil || st.Size() < 1000 {
		t.Fatalf("clip not written: %v", err)
	}
	if again := p.extract(Voice); again != path {
		t.Errorf("second extract = %q", again)
	}
}

// A WAV's size depends only on its length, so a clip a release re-records at
// the same length must still replace the copy cached by the last release.
func TestExtractReplacesAChangedClip(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LocalAppData", t.TempDir())
	base, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	want, err := clips.ReadFile("clips/" + Ding + ".wav")
	if err != nil {
		t.Fatal(err)
	}
	old := bytes.Clone(want)
	old[len(old)-1] ^= 0xff // same size, different sound
	dir := filepath.Join(base, "cuck-code", "sounds")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cached := filepath.Join(dir, Ding+".wav")
	if err := os.WriteFile(cached, old, 0o644); err != nil {
		t.Fatal(err)
	}

	if path := New(true, "").extract(Ding); path != cached {
		t.Fatalf("extract = %q, want %q", path, cached)
	}
	if got, err := os.ReadFile(cached); err != nil || !bytes.Equal(got, want) {
		t.Errorf("cached clip of the same size was kept (err %v); it should be the new clip", err)
	}
}
