// SPDX-License-Identifier: GPL-2.0-only

// Package keymap holds the user's command alias & keyboard-shortcut customization
// (M05-F17, #831) and its YAML persistence. It stores only the user's DELTAS from the
// defaults — the effective bindings are derived live by the app's binding engine from
// the command registry plus this overlay, so the registry stays the single source of
// truth for predefined shortcuts. Chords are kept as their canonical string form
// ([oblikovati.org/api/types.KeyChord.String]), matching the keymap.* wire surface.
package keymap

import (
	"maps"

	"oblikovati.org/persistence/filestore"
	"oblikovati.org/userconfig"
)

// Customization is the user's binding overlay: only entries that differ from the
// derived defaults. Keys are action ids (a command id or a built-in action id); chord
// values are canonical chord strings, alias values are the typed command aliases. The
// zero value (empty maps) means "every binding at its default".
type Customization struct {
	Chords  map[string]string `yaml:"chords,omitempty"`
	Aliases map[string]string `yaml:"aliases,omitempty"`
}

// Defaults returns the empty overlay: with no customization every binding takes its
// derived default. Unlike app/options, the effective defaults are NOT stored here —
// they are computed by the binding engine from the registry — so this is bare on
// purpose.
func Defaults() Customization { return Customization{} }

// Clone returns a deep copy so callers mutate without aliasing the stored maps.
func (c Customization) Clone() Customization {
	return Customization{Chords: cloneMap(c.Chords), Aliases: cloneMap(c.Aliases)}
}

// cloneMap copies a string map, returning nil for an empty source (so omitempty holds).
func cloneMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	maps.Copy(out, m)
	return out
}

// Store persists the customization across sessions.
type Store interface {
	Load() (Customization, error)
	Save(Customization) error
}

// FileStore persists the customization to one YAML file in the user config directory
// (the shared filestore core, #1651).
type FileStore struct {
	file *filestore.FileStore[Customization]
}

// DefaultPath is the per-user keymap file: ~/.oblikovati/keymap.yaml on Linux/macOS.
func DefaultPath() (string, error) { return userconfig.File("keymap.yaml") }

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore {
	return &FileStore{file: filestore.New[Customization](path)}
}

// Load reads the stored customization; a missing file (fresh install) is the empty
// overlay, so a clean install runs entirely on derived defaults.
func (s *FileStore) Load() (Customization, error) {
	c, _, err := s.file.Load()
	if err != nil {
		return Defaults(), err
	}
	return c, nil
}

// Save writes the customization, creating the config directory on first use.
func (s *FileStore) Save(c Customization) error { return s.file.Save(c) }

// MemStore is an in-memory Store for tests, over the shared filestore fake (#1651).
// It keeps keymap's two local behaviors: Load's (Customization, error) shape and
// cloning on Save so callers' maps are never aliased (TestMemStoreSavesACopy).
type MemStore struct {
	filestore.MemStore[Customization]
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore { return &MemStore{} }

// Load returns the last saved customization (empty if none).
func (s *MemStore) Load() (Customization, error) {
	c, _, err := s.MemStore.Load()
	return c, err
}

// Save records a copy of the customization and counts the call.
func (s *MemStore) Save(c Customization) error { return s.MemStore.Save(c.Clone()) }
