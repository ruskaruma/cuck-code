package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ruskaruma/cuck-code/internal/config"
)

// Voice packs are lines of your own for the Claude Code mod: a pack.json
// naming .wav files that sit next to it. They live in a folder next to the
// config file, where the mod looks for them (/cuck pack <name>).

// packsDir is where installed packs live.
func packsDir() string { return filepath.Join(filepath.Dir(config.Path()), "packs") }

// Limits on what a pack may hold.
const (
	maxPackFiles = 100
	maxClipSize  = 5 << 20
	maxPackSize  = 50 << 20
)

var (
	clipName    = regexp.MustCompile(`^[\w.-]+\.wav$`)
	packName    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	githubRepo  = regexp.MustCompile(`^([\w.-]+)/([\w.-]+?)(?:\.git)?(?:@([\w./-]+))?$`)
	errNotAPack = errors.New("there's no pack.json there")
)

// packFile is a pack.json: each section maps a .wav file to its caption.
type packFile struct {
	Name string            `json:"name"`
	Her  map[string]string `json:"her"`
	His  map[string]string `json:"his"`
	Wink map[string]string `json:"wink"`
}

func (p packFile) clips() []string {
	var names []string
	for _, section := range []map[string]string{p.Her, p.His, p.Wink} {
		for name := range section {
			if clipName.MatchString(name) && !strings.HasPrefix(name, ".") {
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}

func cmdPack(args []string, _ config.Config) int {
	if len(args) == 0 {
		args = []string{"list"}
	}
	switch args[0] {
	case "add", "install":
		if len(args) != 2 {
			break
		}
		return packAdd(args[1])
	case "list", "ls":
		return packList()
	case "remove", "rm", "uninstall":
		if len(args) != 2 {
			break
		}
		return packRemove(args[1])
	}
	fmt.Fprintln(os.Stderr, "usage: cuck pack add <owner/repo | folder>\n       cuck pack list\n       cuck pack remove <name>")
	return 2
}

// packAdd installs a pack from GitHub (owner/repo, owner/repo@branch or its
// URL) or from a folder on disk.
func packAdd(src string) int {
	from, name := src, ""
	if st, err := os.Stat(src); err == nil && st.IsDir() {
		abs, _ := filepath.Abs(src)
		name = filepath.Base(abs)
	} else {
		m := githubRepo.FindStringSubmatch(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(src, "https://"), "github.com/"), "/"))
		if m == nil {
			fmt.Fprintf(os.Stderr, "cuck: %s is neither a folder nor a GitHub repository (owner/repo)\n", src)
			return 2
		}
		tmp, err := os.MkdirTemp("", "cuck-pack-")
		if err != nil {
			fmt.Fprintln(os.Stderr, "cuck:", err)
			return 1
		}
		defer os.RemoveAll(tmp)
		from, name = filepath.Join(tmp, "pack"), m[2]
		fmt.Printf("  Fetching github.com/%s/%s...\n", m[1], m[2])
		if err := clone(m[1], m[2], m[3], from); err != nil {
			fmt.Fprintln(os.Stderr, "cuck:", err)
			return 1
		}
	}
	name = strings.Trim(regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(strings.ToLower(name), "-"), "-.")
	if !packName.MatchString(name) {
		fmt.Fprintln(os.Stderr, "cuck: can't make a pack name out of", src)
		return 2
	}
	pack, err := installPack(from, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cuck:", err)
		return 1
	}
	fmt.Printf("  + %s: %s\n", name, describe(pack))
	fmt.Printf("  In Claude Code: /cuck pack %s\n", name)
	return 0
}

// clone fetches a GitHub repository's files with git, shallowly.
func clone(owner, repo, ref, dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	args := []string{"clone", "--quiet", "--depth", "1"}
	if ref != "" {
		args = append(args, "--branch", ref)
	}
	args = append(args, "https://github.com/"+owner+"/"+repo+".git", dir)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.New("fetching a pack from GitHub needs git")
		}
		return fmt.Errorf("couldn't fetch github.com/%s/%s: %s", owner, repo, strings.TrimSpace(string(out)))
	}
	return nil
}

// installPack copies a pack's pack.json and the clips it names (plain .wav
// files right next to it, nothing else) into the packs folder as name.
func installPack(from, name string) (packFile, error) {
	raw, err := readSmall(filepath.Join(from, "pack.json"), 1<<20)
	if errors.Is(err, os.ErrNotExist) {
		return packFile{}, errNotAPack
	}
	if err != nil {
		return packFile{}, err
	}
	var pack packFile
	if err := json.Unmarshal(raw, &pack); err != nil {
		return packFile{}, fmt.Errorf("pack.json: %v", err)
	}
	clips := pack.clips()
	if len(clips) == 0 {
		return packFile{}, errors.New("pack.json names no .wav files")
	}
	if len(clips) > maxPackFiles {
		return packFile{}, fmt.Errorf("a pack holds up to %d clips", maxPackFiles)
	}
	if err := os.MkdirAll(packsDir(), 0o755); err != nil {
		return packFile{}, err
	}
	stage, err := os.MkdirTemp(packsDir(), ".adding-")
	if err != nil {
		return packFile{}, err
	}
	defer os.RemoveAll(stage)
	total := int64(0)
	for _, clip := range clips {
		src := filepath.Join(from, clip)
		st, err := os.Lstat(src)
		if err != nil || !st.Mode().IsRegular() {
			return packFile{}, fmt.Errorf("%s: not a file in the pack", clip)
		}
		if total += st.Size(); st.Size() > maxClipSize || total > maxPackSize {
			return packFile{}, fmt.Errorf("%s: too big (clips up to 5 MB, 50 MB all told)", clip)
		}
		if err := copyFile(src, filepath.Join(stage, clip)); err != nil {
			return packFile{}, err
		}
	}
	if err := os.WriteFile(filepath.Join(stage, "pack.json"), raw, 0o644); err != nil {
		return packFile{}, err
	}
	dest := filepath.Join(packsDir(), name)
	if err := os.RemoveAll(dest); err != nil {
		return packFile{}, err
	}
	return pack, os.Rename(stage, dest)
}

func readSmall(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err == nil && int64(len(b)) > limit {
		err = fmt.Errorf("%s is too big", filepath.Base(path))
	}
	return b, err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// describe says what a pack has: "Pirate, 3 lines for her, 1 for him, a wink".
func describe(p packFile) string {
	parts := []string{}
	if p.Name != "" {
		parts = append(parts, p.Name)
	}
	if n := len(p.Her); n > 0 {
		parts = append(parts, fmt.Sprintf("%d line(s) for her", n))
	}
	if n := len(p.His); n > 0 {
		parts = append(parts, fmt.Sprintf("%d for him", n))
	}
	if len(p.Wink) > 0 {
		parts = append(parts, "a wink")
	}
	return strings.Join(parts, ", ")
}

func packList() int {
	entries, err := os.ReadDir(packsDir())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "cuck:", err)
		return 1
	}
	found := 0
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		raw, err := readSmall(filepath.Join(packsDir(), e.Name(), "pack.json"), 1<<20)
		var pack packFile
		if err != nil || json.Unmarshal(raw, &pack) != nil {
			continue
		}
		found++
		fmt.Printf("  %s: %s\n", e.Name(), describe(pack))
	}
	if found == 0 {
		fmt.Println("  No voice packs yet. Add one: cuck pack add owner/repo")
		return 0
	}
	fmt.Println("\n  In Claude Code: /cuck pack <name>")
	return 0
}

func packRemove(name string) int {
	dir := filepath.Join(packsDir(), name)
	if !packName.MatchString(name) {
		fmt.Fprintln(os.Stderr, "cuck: no voice pack called", name)
		return 1
	}
	if _, err := os.Stat(filepath.Join(dir, "pack.json")); err != nil {
		fmt.Fprintln(os.Stderr, "cuck: no voice pack called", name)
		return 1
	}
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintln(os.Stderr, "cuck:", err)
		return 1
	}
	fmt.Println("  - " + name)
	return 0
}
