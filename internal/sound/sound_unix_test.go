//go:build !windows

package sound

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// With no cache dir (no $HOME) the clips fall back to the temp dir, which
// other users share: whatever they plant there first must never get written
// through.
func TestExtractWithoutACacheDirIgnoresPlantedLinks(t *testing.T) {
	const victimText = "someone's precious file"
	// setup points TMPDIR at a fresh dir, makes os.UserCacheDir fail and
	// returns the temp dir and a victim file outside it.
	setup := func(t *testing.T) (tmp, victim string) {
		tmp, victim = t.TempDir(), filepath.Join(t.TempDir(), "victim")
		if err := os.WriteFile(victim, []byte(victimText), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("HOME", "")
		t.Setenv("XDG_CACHE_HOME", "")
		t.Setenv("TMPDIR", tmp)
		if dir, err := os.UserCacheDir(); err == nil {
			t.Fatalf("UserCacheDir = %q, want an error", dir)
		}
		return tmp, victim
	}
	// plant makes dir/knock.wav a link to victim.
	plant := func(t *testing.T, dir, victim string) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(victim, filepath.Join(dir, Knock+".wav")); err != nil {
			t.Fatal(err)
		}
	}
	intact := func(t *testing.T, victim string) {
		got, err := os.ReadFile(victim)
		if err != nil || string(got) != victimText {
			t.Errorf("victim file overwritten: now %d bytes starting %q (err %v)", len(got), got[:min(4, len(got))], err)
		}
	}
	mine := "cuck-code-" + strconv.Itoa(os.Getuid())

	t.Run("shared dir", func(t *testing.T) {
		tmp, victim := setup(t)
		plant(t, filepath.Join(tmp, "cuck-code", "sounds"), victim)
		New(true, "").extract(Knock)
		intact(t, victim)
	})

	t.Run("per-user dir is a link", func(t *testing.T) {
		tmp, victim := setup(t)
		elsewhere := t.TempDir()
		plant(t, elsewhere, victim)
		if err := os.Symlink(elsewhere, filepath.Join(tmp, mine)); err != nil {
			t.Fatal(err)
		}
		if path := New(true, "").extract(Knock); path != "" {
			t.Errorf("extract = %q, want \"\" (stay silent)", path)
		}
		intact(t, victim)
	})

	t.Run("link planted in the per-user dir", func(t *testing.T) {
		tmp, victim := setup(t)
		dir := filepath.Join(tmp, mine)
		plant(t, dir, victim)
		path := New(true, "").extract(Knock)
		intact(t, victim)
		if filepath.Dir(path) != dir {
			t.Fatalf("extract = %q, want a file in %q", path, dir)
		}
		want, _ := clips.ReadFile("clips/" + Knock + ".wav")
		got, _ := os.ReadFile(path)
		if fi, err := os.Lstat(path); err != nil || !fi.Mode().IsRegular() || !bytes.Equal(got, want) {
			t.Errorf("%s is not a plain copy of the clip (err %v)", path, err)
		}
	})
}

func TestOwnedByCurrentUser(t *testing.T) {
	if fi, err := os.Lstat(t.TempDir()); err != nil || !ownedByCurrentUser(fi) {
		t.Errorf("a dir this test made is not its user's (err %v)", err)
	}
	if os.Getuid() == 0 {
		t.Skip("running as root, which owns /")
	}
	if fi, err := os.Lstat("/"); err != nil || ownedByCurrentUser(fi) {
		t.Errorf("/ counts as the current user's (err %v)", err)
	}
}
