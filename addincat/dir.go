// SPDX-License-Identifier: GPL-2.0-only

package addincat

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// UserAddInsDir is the per-user directory add-ins install into and the host scans at startup.
// It needs no admin rights (#1164):
//
//   - Linux & macOS: ~/oblikovati/addins
//   - Windows:       %AppData%\oblikovati\addins
//
// OBK_USER_ADDINS_DIR overrides it (used by tests and by a user who relocates the folder).
func UserAddInsDir() (string, error) {
	if dir := os.Getenv("OBK_USER_ADDINS_DIR"); dir != "" {
		return dir, nil
	}
	base, err := userBase()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "oblikovati", "addins"), nil
}

// userBase is the parent of the "oblikovati/addins" path: $HOME on the Unix-likes (a visible
// ~/oblikovati folder a user can manage), %AppData% on Windows.
func userBase() (string, error) {
	if runtime.GOOS == "windows" {
		cfg, err := os.UserConfigDir() // %AppData%
		if err != nil {
			return "", fmt.Errorf("addincat: locate AppData dir: %w", err)
		}
		return cfg, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("addincat: locate home dir: %w", err)
	}
	return home, nil
}

// Platform is the current host's bundle key ("linux-amd64", "darwin-arm64", "windows-amd64"),
// matching the platform keys the catalogue stores bundles under.
func Platform() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}
