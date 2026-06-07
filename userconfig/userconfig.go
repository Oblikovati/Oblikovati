// SPDX-License-Identifier: GPL-2.0-only

// Package userconfig is the single source of truth for where Oblikovati keeps its global,
// per-user files (themes, view state, window placement, preferences). Every store resolves
// its path through here so they all live in ONE directory:
//
//   - Linux & macOS: ~/.oblikovati
//   - Windows:       %AppData%\oblikovati (os.UserConfigDir)
//
// A single dotfolder in $HOME on the Unix-likes keeps a user's settings together and
// discoverable; Windows uses the platform-native roaming AppData location.
package userconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Dir returns the per-user Oblikovati config directory, creating nothing.
func Dir() (string, error) {
	if runtime.GOOS == "windows" {
		cfg, err := os.UserConfigDir() // %AppData%\…
		if err != nil {
			return "", fmt.Errorf("userconfig: locate AppData dir: %w", err)
		}
		return filepath.Join(cfg, "oblikovati"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("userconfig: locate home dir: %w", err)
	}
	return filepath.Join(home, ".oblikovati"), nil
}

// File returns the path to name inside the config directory (e.g. "preferences.yaml").
func File(name string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}
