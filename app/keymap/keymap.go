// SPDX-License-Identifier: GPL-2.0-only

// Package keymap holds the user's command alias & keyboard-shortcut customization
// (M05-F17, #831) and its YAML persistence. It stores only the user's DELTAS from the
// defaults — the effective bindings are derived live by the app's binding engine from
// the command registry plus this overlay, so the registry stays the single source of
// truth for predefined shortcuts. Chords are kept as their canonical string form
// ([oblikovati.org/api/types.KeyChord.String]), matching the keymap.* wire surface.
package keymap

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati.org/persistence/yamlcodec"
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
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Store persists the customization across sessions.
type Store interface {
	Load() (Customization, error)
	Save(Customization) error
}

// FileStore persists the customization to one YAML file in the user config directory.
type FileStore struct{ path string }

// DefaultPath is the per-user keymap file: ~/.oblikovati/keymap.yaml on Linux/macOS.
func DefaultPath() (string, error) { return userconfig.File("keymap.yaml") }

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore { return &FileStore{path: path} }

// Load reads the stored customization; a missing file (fresh install) is the empty
// overlay, so a clean install runs entirely on derived defaults.
func (s *FileStore) Load() (Customization, error) {
	c := Defaults()
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("keymap: read %q: %w", s.path, err)
	}
	if err := yamlcodec.Unmarshal(raw, &c); err != nil {
		return Defaults(), fmt.Errorf("keymap: parse %q: %w", s.path, err)
	}
	return c, nil
}

// Save writes the customization, creating the config directory on first use.
func (s *FileStore) Save(c Customization) error {
	data, err := yamlcodec.Marshal(c)
	if err != nil {
		return fmt.Errorf("keymap: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("keymap: create config dir for %q: %w", s.path, err)
	}
	return os.WriteFile(s.path, data, 0o644)
}

// MemStore is an in-memory Store for tests.
type MemStore struct {
	stored Customization
	Saved  int // number of Save calls, for assertions
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore { return &MemStore{} }

// Load returns the last saved customization (empty if none).
func (s *MemStore) Load() (Customization, error) { return s.stored, nil }

// Save records a copy of the customization and counts the call.
func (s *MemStore) Save(c Customization) error {
	s.stored = c.Clone()
	s.Saved++
	return nil
}
