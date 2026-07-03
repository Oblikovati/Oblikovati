// SPDX-License-Identifier: GPL-2.0-only

// Package userprefs persists global, document-independent UI preferences (e.g. whether the
// ViewCube compass is shown) to a single per-user file in the OS config directory. Unlike
// viewstate (per-document) or windowstate (the window), these apply across every document
// and session. Fields use the "hidden/disabled" sense so the zero value is the friendly
// default (shown/enabled) and a fresh install needs no file.
package userprefs

import (
	"oblikovati.org/persistence/filestore"
	"oblikovati.org/userconfig"
)

// Prefs is the set of global user preferences.
type Prefs struct {
	// CompassHidden hides the ViewCube compass when true (default false ⇒ shown).
	CompassHidden bool `yaml:"compassHidden,omitempty"`
	// InactiveOpacity is the ViewCube's face opacity when not hovered (0 ⇒ default 1.0).
	InactiveOpacity float64 `yaml:"inactiveOpacity,omitempty"`
	// CubeSizePx is the ViewCube radius in pixels (0 ⇒ default).
	CubeSizePx int `yaml:"cubeSizePx,omitempty"`
	// CubeCorner is the viewport corner the cube anchors to (0=top-right, 1=top-left,
	// 2=bottom-right, 3=bottom-left).
	CubeCorner int `yaml:"cubeCorner,omitempty"`
	// CubeHidden hides the ViewCube entirely when true (the View-tab toggle; default shown).
	CubeHidden bool `yaml:"cubeHidden,omitempty"`
	// LockToSelection makes the ViewCube orbit around the current selection when true.
	LockToSelection bool `yaml:"lockToSelection,omitempty"`
	// NavBarHidden hides the floating Navigation Bar when true (the View-tab toggle; default shown).
	NavBarHidden bool `yaml:"navBarHidden,omitempty"`
}

// Store loads and saves the global preferences.
type Store interface {
	Load() (Prefs, bool, error)
	Save(Prefs) error
}

// FileStore persists the preferences to a YAML file under the user config directory
// (the shared filestore core, #1651).
type FileStore struct{ file *filestore.FileStore[Prefs] }

// DefaultPath is the per-user preferences file in the shared config dir (userconfig). Named
// distinctly from the theme store's preferences.yaml (which holds the selected theme) to
// avoid clobbering it in the same directory: ~/.oblikovati/ui-preferences.yaml on
// Linux/macOS, %AppData%\oblikovati\ui-preferences.yaml on Windows.
func DefaultPath() (string, error) {
	return userconfig.File("ui-preferences.yaml")
}

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore {
	return &FileStore{file: filestore.New[Prefs](path)}
}

// Load reads the preferences; ok is false when there is no (or an unreadable) file, in
// which case the caller keeps the zero-value defaults.
func (s *FileStore) Load() (Prefs, bool, error) { return s.file.Load() }

// Save writes the preferences, creating the config directory as needed.
func (s *FileStore) Save(p Prefs) error { return s.file.Save(p) }

// MemStore is the shared in-memory Store for tests (#1651).
type MemStore = filestore.MemStore[Prefs]

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore { return filestore.NewMemStore[Prefs]() }
