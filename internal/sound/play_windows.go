//go:build windows

package sound

import (
	"os/exec"
	"strings"
	"syscall"
)

// start plays path in the background with PowerShell: SoundPlayer for WAV,
// the WPF MediaPlayer for anything else (mp3, m4a, ...).
func start(path string) {
	p := strings.ReplaceAll(path, "'", "''")
	script := "(New-Object Media.SoundPlayer '" + p + "').PlaySync()"
	if !strings.HasSuffix(strings.ToLower(path), ".wav") {
		script = "Add-Type -AssemblyName PresentationCore; $m = New-Object System.Windows.Media.MediaPlayer; " +
			"$m.Open([Uri]'" + p + "'); $m.Play(); Start-Sleep -Milliseconds 300; " +
			"while (-not $m.NaturalDuration.HasTimeSpan) { Start-Sleep -Milliseconds 50 }; " +
			"Start-Sleep -Milliseconds ([int]$m.NaturalDuration.TimeSpan.TotalMilliseconds)"
	}
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008} // DETACHED_PROCESS
	_ = cmd.Start()
}
