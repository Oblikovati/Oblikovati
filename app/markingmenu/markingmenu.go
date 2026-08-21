// SPDX-License-Identifier: GPL-2.0-only

// Package markingmenu persists the per-user radial marking menu customization
// and the classic/radial style toggle across sessions. It mirrors the keymap
// package pattern: a fresh install (no file) runs on the session's defaults.
// Chords in the keymap analogue; environments and quadrant slots here.
package markingmenu

import (
	"sort"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/persistence/filestore"
	"oblikovati.org/userconfig"
)

// Customization is the stored marking-menu state: per-environment radial menus
// and the classic/radial style toggle. The zero value (no menus, classic=false)
// means "run on the session's enriched defaults with the radial style."
//
// Explicit yaml: tags are required because wire.MarkingMenuView and friends carry
// only json: tags; yaml.v3 falls back to lowercased field names without them.
type Customization struct {
	Menus   []MenuEntry `yaml:"menus,omitempty"`
	Classic bool        `yaml:"classic,omitempty"`
}

// MenuEntry is one environment's persisted radial menu.
type MenuEntry struct {
	Environment int         `yaml:"environment"`
	Quadrants   []SlotEntry `yaml:"quadrants,omitempty"`
	Overflow    []string    `yaml:"overflow,omitempty"`
}

// SlotEntry is one quadrant slot in a persisted radial menu.
type SlotEntry struct {
	Quadrant int    `yaml:"quadrant"`
	Command  string `yaml:"command"`
}

// Defaults returns the empty customization: a fresh install runs on the session's
// built-in defaults with the radial menu style.
func Defaults() Customization { return Customization{} }

// Clone returns a deep copy so callers mutate without aliasing stored slices.
func (c Customization) Clone() Customization {
	out := Customization{Classic: c.Classic}
	if len(c.Menus) == 0 {
		return out
	}
	out.Menus = make([]MenuEntry, len(c.Menus))
	for i, m := range c.Menus {
		entry := MenuEntry{
			Environment: m.Environment,
			Overflow:    cloneStrings(m.Overflow),
		}
		if len(m.Quadrants) > 0 {
			entry.Quadrants = make([]SlotEntry, len(m.Quadrants))
			copy(entry.Quadrants, m.Quadrants)
		}
		out.Menus[i] = entry
	}
	return out
}

func cloneStrings(ss []string) []string {
	if len(ss) == 0 {
		return nil
	}
	out := make([]string, len(ss))
	copy(out, ss)
	return out
}

// ToCustomization converts a map of MarkingMenuViews and the style flag into the
// persisted form.
func ToCustomization(menus map[types.Environment]wire.MarkingMenuView, classic bool) Customization {
	c := Customization{Classic: classic}
	environments := make([]types.Environment, 0, len(menus))
	for env := range menus {
		environments = append(environments, env)
	}
	sort.Slice(environments, func(i, j int) bool { return environments[i] < environments[j] })
	for _, env := range environments {
		m := menus[env]
		entry := MenuEntry{Environment: int(m.Environment), Overflow: cloneStrings(m.Overflow)}
		for _, slot := range m.Quadrants {
			entry.Quadrants = append(entry.Quadrants, SlotEntry{
				Quadrant: int(slot.Quadrant),
				Command:  slot.CommandID,
			})
		}
		c.Menus = append(c.Menus, entry)
	}
	return c
}

// ApplyToMenus rebuilds the MarkingMenuView map from a Customization, returning
// the menus and the classic flag. Environments with no entries produce no map entry.
func ApplyToMenus(c Customization) (map[types.Environment]wire.MarkingMenuView, bool) {
	menus := make(map[types.Environment]wire.MarkingMenuView, len(c.Menus))
	for _, entry := range c.Menus {
		env := types.Environment(entry.Environment)
		view := wire.MarkingMenuView{Environment: env, Overflow: cloneStrings(entry.Overflow)}
		for _, slot := range entry.Quadrants {
			view.Quadrants = append(view.Quadrants, wire.MarkingMenuItem{
				Quadrant:  types.ScreenQuadrant(slot.Quadrant),
				CommandID: slot.Command,
			})
		}
		menus[env] = view
	}
	return menus, c.Classic
}

// Store persists the customization across sessions.
type Store interface {
	Load() (Customization, error)
	Save(Customization) error
}

// DefaultPath is the per-user marking menu file (e.g. ~/.oblikovati/markingmenu.yaml).
func DefaultPath() (string, error) { return userconfig.File("markingmenu.yaml") }

// FileStore persists the customization to one YAML file in the user config directory
// (the shared filestore core, #1651).
type FileStore struct {
	file *filestore.FileStore[Customization]
}

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore {
	return &FileStore{file: filestore.New[Customization](path)}
}

// Load reads the stored customization; a missing file (fresh install) is the empty
// customization, so the session runs on its enriched defaults.
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

// Save records a deep copy of the customization and counts the call.
func (s *MemStore) Save(c Customization) error { return s.MemStore.Save(c.Clone()) }
