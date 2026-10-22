// Package sound plays the intro's sound effects. It has no dependencies: the
// clips are embedded WAV files, played through whatever audio player the OS
// already has (afplay on macOS; PipeWire, PulseAudio, ALSA or ffplay on
// Linux; PowerShell on Windows). With no player or no audio device it stays
// silent.
package sound

import (
	"bytes"
	"embed"
	"os"
	"path/filepath"
	"strconv"
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
// system player can open it. With no cache dir it uses a private dir in the
// temp dir instead, and stays silent if that dir isn't safely ours.
func (p *Player) extract(cue string) string {
	p.once.Do(func() {
		if base, err := os.UserCacheDir(); err == nil {
			p.dir = filepath.Join(base, "cuck-code", "sounds")
			_ = os.MkdirAll(p.dir, 0o755)
			return
		}
		// Other users share the temp dir and could have put a dir, or a link
		// to one, here first: only use a real dir the current user owns.
		dir := filepath.Join(os.TempDir(), "cuck-code-"+strconv.Itoa(os.Getuid()))
		_ = os.MkdirAll(dir, 0o700)
		if fi, err := os.Lstat(dir); err == nil && fi.IsDir() && ownedByCurrentUser(fi) {
			p.dir = dir
		}
	})
	if p.dir == "" {
		return ""
	}
	data, err := clips.ReadFile("clips/" + cue + ".wav")
	if err != nil {
		return ""
	}
	path := filepath.Join(p.dir, cue+".wav")
	// Compare contents, not sizes: a WAV's size depends only on its length,
	// so a re-recorded clip of the same length would otherwise never land.
	if old, err := os.ReadFile(path); err != nil || !bytes.Equal(old, data) {
		if writeClip(path, data) != nil {
			return ""
		}
	}
	return path
}

// writeClip writes data to a new temp file next to path and renames it over
// path, so a link already at path is replaced rather than written through.
func writeClip(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Chmod(0o644)
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		_ = os.Remove(f.Name())
	}
	return err
}
