// SPDX-License-Identifier: GPL-2.0-only

// Package windowstate persists the head window's last placement (position, size, and
// maximized flag) to a per-user file in the OS config directory, so the application
// reopens where it was — same spot, same monitor — across sessions. Position is stored in
// GLFW virtual-screen coordinates, which already encode which monitor the window is on.
package windowstate

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati/persistence/yamlcodec"
)

// State is the saved window placement.
type State struct {
	X         int  `yaml:"x"`
	Y         int  `yaml:"y"`
	Width     int  `yaml:"width"`
	Height    int  `yaml:"height"`
	Maximized bool `yaml:"maximized,omitempty"`
}

// Valid reports whether the state has a usable size (a zero/garbage record is ignored so
// the app falls back to its default geometry).
func (s State) Valid() bool { return s.Width > 0 && s.Height > 0 }

// FilePath is the per-user window-state file (e.g. ~/.config/oblikovati/window.yaml).
func FilePath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("windowstate: locate user config dir: %w", err)
	}
	return filepath.Join(cfg, "oblikovati", "window.yaml"), nil
}

// Load reads the saved placement; ok is false when there is no (or an unreadable) file.
func Load() (State, bool) {
	path, err := FilePath()
	if err != nil {
		return State{}, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return State{}, false
	}
	var s State
	if err := yamlcodec.Unmarshal(raw, &s); err != nil || !s.Valid() {
		return State{}, false
	}
	return s, true
}

// Save writes the placement, creating the config directory if needed. A no-op for an
// invalid (zero-size) state.
func Save(s State) error {
	if !s.Valid() {
		return nil
	}
	path, err := FilePath()
	if err != nil {
		return err
	}
	raw, err := yamlcodec.Marshal(s)
	if err != nil {
		return fmt.Errorf("windowstate: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("windowstate: create config dir: %w", err)
	}
	return os.WriteFile(path, raw, 0o644)
}
