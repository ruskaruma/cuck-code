package shellhook

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	beginMarker = "# >>> cuck-code >>>"
	endMarker   = "# <<< cuck-code <<<"
)

// Target is one startup file the hook is written to.
type Target struct {
	Shell string // bash, zsh, fish or powershell
	Path  string
	Own   bool // the whole file belongs to cuck (fish conf.d); otherwise a marked block inside it
}

func (t Target) loader() string {
	switch t.Shell {
	case "fish":
		return "if type -q cuck\n    cuck hook fish | source\nend"
	case "powershell":
		return "if (Get-Command cuck -ErrorAction SilentlyContinue) { Invoke-Expression ((& cuck hook powershell) -join \"`n\") }"
	}
	return "if command -v cuck >/dev/null 2>&1; then eval \"$(cuck hook " + t.Shell + ")\"; fi"
}

func (t Target) block() string {
	return beginMarker + "\n# Coding agents play the Cuck Code intro first. Undo with: cuck uninstall\n" + t.loader() + "\n" + endMarker + "\n"
}

// Targets returns the startup files to hook for the shells installed on this
// machine. With system set, it returns the machine-wide files (needs root or
// Administrator); otherwise the current user's.
func Targets(system bool) []Target {
	var ts []Target
	have := func(name string) bool { _, err := exec.LookPath(name); return err == nil }
	home, _ := os.UserHomeDir()
	first := func(paths ...string) string {
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return paths[len(paths)-1]
	}

	if runtime.GOOS != "windows" {
		if have("bash") {
			if system {
				ts = append(ts, Target{Shell: "bash", Path: first("/etc/bash.bashrc", "/etc/bashrc")})
			} else if home != "" {
				ts = append(ts, Target{Shell: "bash", Path: filepath.Join(home, ".bashrc")})
				if runtime.GOOS == "darwin" {
					// macOS Terminal starts bash as a login shell, which skips .bashrc.
					ts = append(ts, Target{Shell: "bash", Path: filepath.Join(home, ".bash_profile")})
				}
			}
		}
		if have("zsh") {
			if system {
				ts = append(ts, Target{Shell: "zsh", Path: first("/etc/zsh/zshrc", "/etc/zshrc")})
			} else if home != "" {
				dir := os.Getenv("ZDOTDIR")
				if dir == "" {
					dir = home
				}
				ts = append(ts, Target{Shell: "zsh", Path: filepath.Join(dir, ".zshrc")})
			}
		}
		if have("fish") {
			dir := filepath.Join(home, ".config", "fish", "conf.d")
			if system {
				dir = "/etc/fish/conf.d"
			}
			if system || home != "" {
				ts = append(ts, Target{Shell: "fish", Path: filepath.Join(dir, "cuck-code.fish"), Own: true})
			}
		}
	} else if have("bash") && !system && home != "" {
		ts = append(ts, Target{Shell: "bash", Path: filepath.Join(home, ".bashrc")}) // Git Bash
	}

	// PowerShell: ask each installed edition where its profile lives, since
	// Documents may be redirected (OneDrive) on Windows.
	which := "$PROFILE.CurrentUserAllHosts"
	if system {
		which = "$PROFILE.AllUsersAllHosts"
	}
	for _, ps := range []string{"pwsh", "powershell"} {
		if !have(ps) || (ps == "powershell" && runtime.GOOS != "windows") {
			continue
		}
		out, err := exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", which).Output()
		if p := strings.TrimSpace(string(out)); err == nil && p != "" {
			ts = append(ts, Target{Shell: "powershell", Path: p})
		}
	}
	return ts
}

// Installed reports whether t currently contains the hook.
func (t Target) Installed() bool {
	b, err := os.ReadFile(t.Path)
	if err != nil {
		return false
	}
	return t.Own || bytes.Contains(b, []byte(beginMarker))
}

// Install writes (or refreshes) the hook in t.
func (t Target) Install() error {
	if err := os.MkdirAll(filepath.Dir(t.Path), 0o755); err != nil {
		return err
	}
	if t.Own {
		return os.WriteFile(t.Path, []byte(t.block()), 0o644)
	}
	old, err := os.ReadFile(t.Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	content := stripBlock(string(old))
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" {
		content += "\n"
	}
	// WriteFile follows symlinks, so dotfile managers keep working.
	return os.WriteFile(t.Path, []byte(content+t.block()), fileMode(t.Path))
}

// Remove deletes the hook from t. It is a no-op when the hook is absent.
func (t Target) Remove() error {
	if t.Own {
		err := os.Remove(t.Path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	old, err := os.ReadFile(t.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !strings.Contains(string(old), beginMarker) {
		return nil
	}
	return os.WriteFile(t.Path, []byte(stripBlock(string(old))), fileMode(t.Path))
}

// stripBlock removes the marked block and the blank line Install put before it.
func stripBlock(s string) string {
	i := strings.Index(s, beginMarker)
	if i < 0 {
		return s
	}
	j := strings.Index(s[i:], endMarker)
	if j < 0 {
		return s
	}
	end := i + j + len(endMarker)
	if end < len(s) && s[end] == '\n' {
		end++
	}
	head := strings.TrimRight(s[:i], "\n")
	if head != "" {
		head += "\n"
	}
	return head + s[end:]
}

func fileMode(path string) fs.FileMode {
	if st, err := os.Stat(path); err == nil {
		return st.Mode().Perm()
	}
	return 0o644
}
