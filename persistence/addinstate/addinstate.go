// SPDX-License-Identifier: GPL-2.0-only

// Package addinstate persists the per-user add-in load behaviors (when each
// installed add-in activates: startup / demand / disabled) to a single YAML file in
// the OS config directory — the same per-user seam as themes and UI preferences
// (M05-F01, #251). Only non-default behaviors are written, so a fresh install needs
// no file and everything loads on startup.
package addinstate

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati.org/api/types"
	"oblikovati.org/persistence/yamlcodec"
	"oblikovati.org/userconfig"
)

// behaviorsFile is the on-disk document: add-in id → stable behavior name
// ("demand", "disabled"). Names, not numbers, so the file is hand-editable.
type behaviorsFile struct {
	Behaviors map[string]string `yaml:"behaviors"`
}

// FileStore persists the behaviors to a YAML file; it satisfies
// app.AddInBehaviorStore.
type FileStore struct{ path string }

// DefaultPath is the per-user behaviors file in the shared config dir:
// ~/.oblikovati/addin-behaviors.yaml on Linux/macOS.
func DefaultPath() (string, error) {
	return userconfig.File("addin-behaviors.yaml")
}

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore { return &FileStore{path: path} }

// Load reads the stored behaviors; a missing file is an empty map (fresh install).
// An unknown behavior name is skipped rather than guessed, so a hand-edited typo
// falls back to the default instead of disabling an add-in.
func (s *FileStore) Load() (map[string]types.AddInLoadBehavior, error) {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return map[string]types.AddInLoadBehavior{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("addinstate: read %q: %w", s.path, err)
	}
	var f behaviorsFile
	if err := yamlcodec.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("addinstate: parse %q: %w", s.path, err)
	}
	out := map[string]types.AddInLoadBehavior{}
	for id, name := range f.Behaviors {
		if b, ok := types.ParseAddInLoadBehavior(name); ok {
			out[id] = b
		}
	}
	return out, nil
}

// Save writes the behaviors, creating the config directory on first use.
func (s *FileStore) Save(m map[string]types.AddInLoadBehavior) error {
	f := behaviorsFile{Behaviors: map[string]string{}}
	for id, b := range m {
		f.Behaviors[id] = b.String()
	}
	data, err := yamlcodec.Marshal(f)
	if err != nil {
		return fmt.Errorf("addinstate: marshal behaviors: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("addinstate: create config dir for %q: %w", s.path, err)
	}
	return os.WriteFile(s.path, data, 0o644)
}
