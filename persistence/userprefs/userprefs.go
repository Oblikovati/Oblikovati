// SPDX-License-Identifier: GPL-2.0-only

// Package userprefs persists global, document-independent UI preferences (e.g. whether the
// ViewCube compass is shown) to a single per-user file in the OS config directory. Unlike
// viewstate (per-document) or windowstate (the window), these apply across every document
// and session. Fields use the "hidden/disabled" sense so the zero value is the friendly
// default (shown/enabled) and a fresh install needs no file.
package userprefs

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati/persistence/yamlcodec"
	"oblikovati/userconfig"
)

// Prefs is the set of global user preferences.
type Prefs struct {
	// CompassHidden hides the ViewCube compass when true (default false ⇒ shown).
	CompassHidden bool `yaml:"compassHidden,omitempty"`
	// InactiveOpacity is the ViewCube's face opacity when not hovered (0 ⇒ default 1.0).
	InactiveOpacity float64 `yaml:"inactiveOpacity,omitempty"`
}

// Store loads and saves the global preferences.
type Store interface {
	Load() (Prefs, bool, error)
	Save(Prefs) error
}

// FileStore persists the preferences to a YAML file under the user config directory.
type FileStore struct{ path string }

// DefaultPath is the per-user preferences file in the shared config dir (userconfig). Named
// distinctly from the theme store's preferences.yaml (which holds the selected theme) to
// avoid clobbering it in the same directory: ~/.oblikovati/ui-preferences.yaml on
// Linux/macOS, %AppData%\oblikovati\ui-preferences.yaml on Windows.
func DefaultPath() (string, error) {
	return userconfig.File("ui-preferences.yaml")
}

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore { return &FileStore{path: path} }

// Load reads the preferences; ok is false when there is no (or an unreadable) file, in
// which case the caller keeps the zero-value defaults.
func (s *FileStore) Load() (Prefs, bool, error) {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return Prefs{}, false, nil
	}
	if err != nil {
		return Prefs{}, false, fmt.Errorf("userprefs: read %q: %w", s.path, err)
	}
	var p Prefs
	if err := yamlcodec.Unmarshal(raw, &p); err != nil {
		return Prefs{}, false, fmt.Errorf("userprefs: parse %q: %w", s.path, err)
	}
	return p, true, nil
}

// Save writes the preferences, creating the config directory as needed.
func (s *FileStore) Save(p Prefs) error {
	raw, err := yamlcodec.Marshal(p)
	if err != nil {
		return fmt.Errorf("userprefs: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("userprefs: create config dir: %w", err)
	}
	return os.WriteFile(s.path, raw, 0o644)
}

// MemStore is an in-memory Store for tests.
type MemStore struct {
	prefs Prefs
	saved bool
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore { return &MemStore{} }

// Load implements Store.
func (s *MemStore) Load() (Prefs, bool, error) { return s.prefs, s.saved, nil }

// Save implements Store.
func (s *MemStore) Save(p Prefs) error {
	s.prefs, s.saved = p, true
	return nil
}
