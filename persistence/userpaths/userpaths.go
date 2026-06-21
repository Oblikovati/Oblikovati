// SPDX-License-Identifier: GPL-2.0-only

// Package userpaths resolves the per-user Oblikovati directory that holds installed add-ins,
// the machine-identity globals file, and other no-admin-required user data. Centralizing the
// base resolution keeps add-in install paths (addincat) and telemetry identity (usagestats)
// pointing at the same ~/oblikovati (or %AppData%\oblikovati) tree.
package userpaths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// OblikovatiHome is the per-user Oblikovati directory:
//
//   - Linux & macOS: ~/oblikovati
//   - Windows:       %AppData%\oblikovati
//
// It needs no admin rights. Subdirectories (e.g. "addins") and files (e.g. "globals") live
// under it; callers join their own leaf.
func OblikovatiHome() (string, error) {
	base, err := userBase()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "oblikovati"), nil
}

// userBase is the parent of the "oblikovati" directory: $HOME on the Unix-likes (a visible
// ~/oblikovati folder a user can manage), %AppData% on Windows.
func userBase() (string, error) {
	if runtime.GOOS == "windows" {
		cfg, err := os.UserConfigDir() // %AppData%
		if err != nil {
			return "", fmt.Errorf("userpaths: locate AppData dir: %w", err)
		}
		return cfg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("userpaths: locate home dir: %w", err)
	}
	return home, nil
}
