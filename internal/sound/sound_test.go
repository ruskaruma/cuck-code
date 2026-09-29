package sound

import (
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
