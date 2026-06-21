// SPDX-License-Identifier: GPL-2.0-only

package addincat

import (
	"os"
	"path/filepath"
	"runtime"

	"oblikovati.org/persistence/userpaths"
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
	home, err := userpaths.OblikovatiHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "addins"), nil
}

// Platform is the current host's bundle key ("linux-amd64", "darwin-arm64", "windows-amd64"),
// matching the platform keys the catalogue stores bundles under.
func Platform() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}
