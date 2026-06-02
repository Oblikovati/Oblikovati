// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"os"
	"path/filepath"
)

// OSFileSystem is the production [FileSystem]: real files under the user config dir.
type OSFileSystem struct{}

// ReadDir returns the base names of the files in dir. A missing directory (first run)
// yields no names and no error, per the [FileSystem] contract.
func (OSFileSystem) ReadDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ReadFile reads a file's contents.
func (OSFileSystem) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

// WriteFile writes data to path, creating parent directories as needed (0o755 dirs,
// 0o644 file) so the first save into a fresh config dir succeeds.
func (OSFileSystem) WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Remove deletes a file; a missing file is not an error (idempotent delete).
func (OSFileSystem) Remove(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
