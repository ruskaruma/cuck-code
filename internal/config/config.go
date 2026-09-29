// Package config loads the optional user config file.
//
// The file lives at $CUCK_CONFIG, or <user config dir>/cuck-code/config.json
// (~/.config on Linux, ~/Library/Application Support on macOS, %AppData% on
// Windows). Every field is optional:
//
//	{
//	  "agent": "claude",
//	  "agents": { "yolo": "claude --dangerously-skip-permissions" },
//	  "animation": true,
//	  "duration_ms": 7500,
//	  "fps": 30,
//	  "color": "auto",
//	  "sound": true,
//	  "sound_file": "~/Music/good-d.mp3",
//	  "hook_agents": ["my-agent"],
//	  "hook_exclude": ["cursor"]
//	}
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Config is the user's saved preferences.
type Config struct {
	// Agent is the default agent name or command line.
	Agent string `json:"agent"`
	// Agents maps short names to full command lines, e.g. "yolo" → "claude --dangerously-skip-permissions".
	Agents map[string]string `json:"agents"`
	// Animation can be set to false to always skip the intro.
	Animation *bool `json:"animation"`
	// DurationMS is the intro length in milliseconds.
	DurationMS int `json:"duration_ms"`
	// FPS is the intro frame rate.
	FPS int `json:"fps"`
	// Color is the colour mode: auto, truecolor, 256, 16 or none.
	Color string `json:"color"`
	// Sound turns on the intro's sound effects and voice line (off by default).
	Sound *bool `json:"sound"`
	// SoundFile replaces the built-in voice line with your own clip (wav, mp3, ...).
	SoundFile string `json:"sound_file"`
	// HookAgents are extra commands for the shell hook to wrap.
	HookAgents []string `json:"hook_agents"`
	// HookExclude are built-in agents the shell hook should leave alone.
	HookExclude []string `json:"hook_exclude"`
}

// Path returns the config file location.
func Path() string {
	if p := os.Getenv("CUCK_CONFIG"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "cuck-code", "config.json")
}

// Load reads the config file. A missing file is not an error; a malformed one is.
func Load() (Config, error) {
	var c Config
	path := Path()
	if path == "" {
		return c, nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// Set writes one top-level key into the config file, keeping everything else
// in it (including fields this version doesn't know about).
func Set(key string, value any) error {
	path := Path()
	if path == "" {
		return errors.New("no config directory")
	}
	m := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &m); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	m[key] = value
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// Resolve expands an agent alias to its command line. Names that are not
// aliases are returned unchanged.
func (c Config) Resolve(name string) string {
	if cmd, ok := c.Agents[name]; ok && cmd != "" {
		return cmd
	}
	return name
}
