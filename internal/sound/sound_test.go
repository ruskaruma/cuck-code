package sound

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEveryCueHasAClip(t *testing.T) {
	cues := []string{Knock, Door, Bed, Smooch, Ding, Voice, Boom, VoiceClip("", true)}
	for _, v := range voices {
		cues = append(cues, v)
	}
	for _, c := range cues {
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

func TestVoiceClip(t *testing.T) {
	for agent, want := range map[string]string{"claude": Voice, "": Voice, "codex": "voice_codex", "cursor-agent": "voice_cursor", "goose": "voice_goose"} {
		if got := VoiceClip(agent, false); got != want {
			t.Errorf("VoiceClip(%q) = %q, want %q", agent, got, want)
		}
	}
	if got := VoiceClip("codex", true); got != "voice_sfw" {
		t.Errorf("safe for work, VoiceClip = %q", got)
	}
}

func TestSafeForWorkSkipsTheBedAndTheKiss(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	p := New(true, "")
	p.SFW = true
	p.Play(Bed)
	p.Play(Smooch)
	if p.dir != "" {
		t.Error("safe for work, the bed and the kiss should stay silent")
	}
}

func TestOnCallPatterns(t *testing.T) {
	reg := "HKEY_CURRENT_USER\\...\\NonPackaged\\C:#Zoom#bin#Zoom.exe\r\n    LastUsedTimeStop    REG_QWORD    0x0\r\n"
	if !micInUse.MatchString(reg) || micInUse.MatchString(strings.Replace(reg, "0x0", "0x1d9a3b2c4e5f600", 1)) {
		t.Error("the Windows microphone check reads reg's output wrong")
	}
	if !callApps.MatchString("/Applications/zoom.us.app/Contents/Frameworks/CptHost.app/Contents/MacOS/CptHost\n") ||
		callApps.MatchString("/Applications/zoom.us.app/Contents/MacOS/zoom.us\n") {
		t.Error("the macOS call check reads ps's output wrong")
	}
}
