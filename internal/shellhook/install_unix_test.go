//go:build unix

package shellhook

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// A write that fails part way (disk full, quota, a size limit) must leave the
// user's startup file as it was, not truncated.
func TestFailedWriteKeepsStartupFile(t *testing.T) {
	orig := strings.Repeat("# the user's own configuration line\n", 140) // ~5 KB
	for _, op := range []string{"install", "remove"} {
		path := filepath.Join(t.TempDir(), ".bashrc")
		content := orig
		if op == "remove" {
			content = orig + "\n" + Target{Shell: "bash"}.block("/usr/local/bin/cuck")
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		tg := Target{Shell: "bash", Path: path}

		err := withFileSizeLimit(t, 2048, func() error {
			if op == "install" {
				return tg.Install("/usr/local/bin/cuck")
			}
			return tg.Remove()
		})
		if err == nil {
			t.Errorf("%s: no error although the write could not finish", op)
		}
		if b, _ := os.ReadFile(path); string(b) != content {
			t.Errorf("%s: startup file changed by a failed write: %d bytes, want %d", op, len(b), len(content))
		}
	}
}

// withFileSizeLimit runs f with writes beyond limit bytes failing (EFBIG).
func withFileSizeLimit(t *testing.T, limit uint64, f func() error) error {
	var old syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &old); err != nil {
		t.Skip("can't read the file size limit:", err)
	}
	lim := old
	setLimit(&lim.Cur, limit)
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &lim); err != nil {
		t.Skip("can't set a file size limit:", err)
	}
	defer syscall.Setrlimit(syscall.RLIMIT_FSIZE, &old)
	return f()
}

// setLimit sets an Rlimit field, which is unsigned on most systems and
// signed on FreeBSD and DragonFly.
func setLimit[T int64 | uint64](field *T, v uint64) { *field = T(v) }

// Dotfile managers keep startup files as symlinks: the hook goes into the
// file the link points to, and the link stays a link.
func TestInstallFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "dotfiles", "bashrc")
	if err := os.MkdirAll(filepath.Dir(real), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(real, []byte("export A=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, ".bashrc")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	tg := Target{Shell: "bash", Path: link}
	if err := tg.Install("/usr/local/bin/cuck"); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Lstat(link); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%s is no longer a symlink", link)
	}
	if st, _ := os.Stat(real); st.Mode().Perm() != 0o600 {
		t.Errorf("mode of the link's target changed to %v", st.Mode().Perm())
	}
	if b, _ := os.ReadFile(real); !strings.Contains(string(b), beginMarker) {
		t.Errorf("hook not written to the link's target:\n%s", b)
	}
	if entries, _ := os.ReadDir(filepath.Dir(real)); len(entries) != 1 {
		t.Errorf("temporary files left behind: %v", entries)
	}
}
