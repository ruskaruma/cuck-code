// Package sound plays the intro's sound effects. It has no dependencies: the
// clips are embedded WAV files, played through whatever audio player the OS
// already has (afplay on macOS; PipeWire, PulseAudio, ALSA or ffplay on
// Linux; PowerShell on Windows). With no player or no audio device it stays
// silent.
package sound

import (
	"embed"
	"os"
	"path/filepath"
	"sync"
)

//go:embed clips/*.wav
var clips embed.FS

// Cue names, one embedded clip each.
const (
	Knock  = "knock"
	Door   = "door"
	Bed    = "bed"
	Smooch = "smooch"
	Ding   = "ding"
	Voice  = "voice" // "who got the good d? who got the good d?"
	Boom   = "boom"
)

// Player plays cues in the background. The zero value is silent.
type Player struct {
	// VoiceFile, if set, is played instead of the built-in voice line.
	VoiceFile string

	enabled bool
	dir     string
	once    sync.Once
}

// New returns a player; when enabled is false every call is a no-op.
func New(enabled bool, voiceFile string) *Player {
	return &Player{enabled: enabled, VoiceFile: voiceFile}
}

// Play starts a cue and returns immediately. Clips keep playing after cuck
// hands the terminal over to the agent.
func (p *Player) Play(cue string) {
	if p == nil || !p.enabled {
		return
	}
	if cue == Voice && p.VoiceFile != "" {
		if _, err := os.Stat(p.VoiceFile); err == nil {
			start(p.VoiceFile)
			return
		}
	}
	if path := p.extract(cue); path != "" {
		start(path)
	}
}

// extract writes the embedded clip to the user's cache dir (once) so the
// system player can open it.
func (p *Player) extract(cue string) string {
	p.once.Do(func() {
		base, err := os.UserCacheDir()
		if err != nil {
			base = os.TempDir()
		}
		p.dir = filepath.Join(base, "cuck-code", "sounds")
		_ = os.MkdirAll(p.dir, 0o755)
	})
	data, err := clips.ReadFile("clips/" + cue + ".wav")
	if err != nil {
		return ""
	}
	path := filepath.Join(p.dir, cue+".wav")
	if st, err := os.Stat(path); err != nil || st.Size() != int64(len(data)) {
		if os.WriteFile(path, data, 0o644) != nil {
			return ""
		}
	}
	return path
}
