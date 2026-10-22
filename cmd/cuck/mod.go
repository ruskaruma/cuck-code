package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

// The Claude Code mod (packaging/claude-code) plays a scene above Claude
// Code's prompt while it works. Setup installs it alongside the shell hook,
// through Claude Code's own plugin command, from the marketplace this
// repository is.
const (
	modPlugin = "cuck-code@cuck-code" // <plugin>@<marketplace>
	modName   = "cuck-code"
)

// modMarketplace is where the mod is installed from: this GitHub repository,
// or $CUCK_MOD_MARKETPLACE (a path or URL, for testing a local checkout).
func modMarketplace() string {
	if m := os.Getenv("CUCK_MOD_MARKETPLACE"); m != "" {
		return m
	}
	return "ruskaruma/cuck-code"
}

// claudeCode runs `claude plugin ...` and returns its single JSON result line.
func claudeCode(timeout time.Duration, args ...string) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "claude", append([]string{"plugin"}, args...)...)
	cmd.Stdin = nil
	out, err := cmd.Output()
	var res map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if json.Unmarshal([]byte(line), &res) == nil {
			break
		}
	}
	if res != nil && res["outcome"] == "ok" {
		return res, nil
	}
	if msg, ok := res["message"].(string); ok && msg != "" {
		return res, errors.New(msg)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, errors.New("Claude Code took too long to answer")
	}
	if err != nil {
		return nil, err
	}
	return nil, errors.New("Claude Code gave no answer")
}

func haveClaudeCode() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// modInstalled reports whether the mod is installed in Claude Code.
func modInstalled() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "claude", "plugin", "list", "--json").Output()
	if err != nil {
		return false
	}
	var list []struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	if json.Unmarshal(out, &list) != nil {
		return false
	}
	for _, p := range list {
		if p.ID == modPlugin {
			return true
		}
	}
	return false
}

// installMod adds the marketplace if needed and installs the mod for the user.
func installMod() error {
	_, err := claudeCode(3*time.Minute, "install", modName, "--marketplace", modMarketplace(), "--scope", "user", "--json")
	return err
}

// removeMod uninstalls the mod and forgets its marketplace.
func removeMod() error {
	if _, err := claudeCode(time.Minute, "uninstall", modPlugin, "--scope", "user", "--json"); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_ = exec.CommandContext(ctx, "claude", "plugin", "marketplace", "remove", modName).Run()
	return nil
}
