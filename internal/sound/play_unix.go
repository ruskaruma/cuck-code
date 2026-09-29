//go:build !windows

package sound

import (
	"os/exec"
	"runtime"
	"strings"
)

// player returns the command line that plays path, or nil if nothing can.
func player(path string) []string {
	if runtime.GOOS == "darwin" {
		return []string{"afplay", path}
	}
	wav := strings.HasSuffix(strings.ToLower(path), ".wav")
	candidates := [][]string{
		{"pw-play", path},
		{"paplay", path},
		{"aplay", "-q", path},
		{"ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", path},
		{"mpv", "--no-video", "--really-quiet", path},
	}
	if !wav {
		// Only the general-purpose players understand mp3/ogg/m4a reliably.
		candidates = candidates[3:]
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err == nil {
			return c
		}
	}
	return nil
}

// start plays path in the background. It goes through `sh -c '... &'` so the
// player is orphaned to init: it keeps playing after cuck execs the agent and
// never lingers as the agent's zombie child. Ignoring SIGHUP lets the clip
// finish even if the agent exits and the terminal closes straight away.
func start(path string) {
	argv := player(path)
	if argv == nil {
		return
	}
	cmd := exec.Command("/bin/sh", append([]string{"-c", `trap '' HUP; "$@" >/dev/null 2>&1 &`, "sh"}, argv...)...)
	_ = cmd.Run()
}
