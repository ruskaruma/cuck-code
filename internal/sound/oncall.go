package sound

import (
	"context"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// OnCall reports whether you seem to be on a call, so the intro can keep
// quiet and clean: the microphone is in use (Linux with PulseAudio or
// PipeWire, Windows), or a Zoom or FaceTime call is up (macOS). It gives up,
// saying no, after a moment.
func OnCall() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	switch runtime.GOOS {
	case "windows":
		// Each app that has used the microphone has a LastUsedTimeStop; 0 while it still is.
		out, err := exec.CommandContext(ctx, "reg", "query",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\microphone`,
			"/s", "/v", "LastUsedTimeStop").Output()
		return err == nil && micInUse.Match(out)
	case "darwin":
		out, err := exec.CommandContext(ctx, "ps", "-A", "-o", "comm=").Output()
		return err == nil && callApps.Match(out)
	default:
		out, err := exec.CommandContext(ctx, "pactl", "list", "source-outputs", "short").Output()
		return err == nil && strings.TrimSpace(string(out)) != ""
	}
}

var (
	micInUse = regexp.MustCompile(`(?m)LastUsedTimeStop\s+REG_QWORD\s+0x0\s*$`)
	callApps = regexp.MustCompile(`(?im)(CptHost|aomhost|FaceTime)$`)
)
