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
	Voice  = "voice" // "who got the good d? who got the good d?", or the agent's own line
	Boom   = "boom"
)

// voices are the agents with a wink line of their own; the rest say the usual.
var voices = map[string]string{
	"codex":        "voice_codex",  // who got the good diff?
	"gemini":       "voice_gemini", // who got the good d? twice.
	"cursor":       "voice_cursor", // who got the good tab?
	"cursor-agent": "voice_cursor",
	"copilot":      "voice_copilot", // who got the good autocomplete?
	"aider":        "voice_aider",   // who got the good commit?
	"goose":        "voice_goose",   // honk honk. who got the good d?
}

// VoiceClip is the clip agent says as he winks: his own line, or "who got
// the good code?" when it's safe for work.
func VoiceClip(agent string, sfw bool) string {
	if sfw {
		return "voice_sfw"
	}
	if v, ok := voices[agent]; ok {
		return v
	}
	return Voice
}

// Clip returns a clip's WAV bytes.
func Clip(name string) ([]byte, error) { return clips.ReadFile("clips/" + name + ".wav") }

// Player plays cues in the background. The zero value is silent.
type Player struct {
	// VoiceFile, if set, is played instead of the built-in voice line.
	VoiceFile string
	// Agent is who's winking: some agents have a line of their own.
	Agent string
	// SFW keeps it safe for work: no bed or kiss, and a clean wink line.
	SFW bool

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
	if p == nil || !p.enabled || (p.SFW && (cue == Bed || cue == Smooch)) {
		return
	}
	if cue == Voice {
		if p.VoiceFile != "" && !p.SFW {
			if _, err := os.Stat(p.VoiceFile); err == nil {
				start(p.VoiceFile)
				return
			}
		}
		cue = VoiceClip(p.Agent, p.SFW)
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
	data, err := Clip(cue)
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
