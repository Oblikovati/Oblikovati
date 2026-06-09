// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/contract"
	"oblikovati.org/theme"
)

// Themes returns the session's UI theme library (built-ins plus the user's customs with
// an active selection). On first use, before any store is attached, it holds only the
// shipped Dark/Light built-ins — the windowed head/CLI call [Session.LoadThemes] to fold
// in the user's saved customs. Unit sessions therefore see a deterministic built-in-only
// library with no disk access.
func (s *Session) Themes() *theme.Library {
	if s.themes == nil {
		s.themes = theme.NewLibrary(nil, "Dark")
	}
	return s.themes
}

// Theme returns the active theme as the public read-only contract (what the wire router
// serves and add-ins read).
func (s *Session) Theme() contract.Theme { return s.Themes().Active() }

// ThemeRevision is the active library's change counter; the head compares it across
// frames to decide whether to re-apply the theme to Dear ImGui (live preview).
func (s *Session) ThemeRevision() uint64 { return s.Themes().Revision() }

// LoadThemes attaches a persistence store and replaces the in-memory library with one
// built from the user's saved customs and selected theme. The binary (head/CLI) calls
// this once at startup with an OS-backed store; a load error (one corrupt file) is
// returned but the good themes still load. The store is retained so later edits persist.
func (s *Session) LoadThemes(store *theme.Store) error {
	customs, active, err := store.Load()
	s.themes = theme.NewLibrary(customs, active)
	s.themeStore = store
	return err
}

// SetActiveTheme selects a theme by name and persists the choice (when a store is
// attached). Selecting an unknown name is a no-op.
func (s *Session) SetActiveTheme(name string) error {
	s.Themes().SetActive(name)
	return s.persistActive()
}

// DuplicateTheme creates a custom full-snapshot copy of base under newName, makes it
// active, and persists both the new theme file and the selection. It surfaces the
// library's name/uniqueness error.
func (s *Session) DuplicateTheme(base, newName string) error {
	dup, err := s.Themes().Duplicate(base, newName)
	if err != nil {
		return err
	}
	if err := s.persistTheme(dup); err != nil {
		return err
	}
	return s.persistActive()
}

// DeleteTheme removes a custom theme (reselecting Dark if it was active) and deletes its
// file. It surfaces the library's "not found / built-in" error.
func (s *Session) DeleteTheme(name string) error {
	if err := s.Themes().Remove(name); err != nil {
		return err
	}
	if s.themeStore != nil {
		if err := s.themeStore.RemoveTheme(name); err != nil {
			return err
		}
	}
	return s.persistActive()
}

// SaveActiveTheme writes the active theme's current colors to disk (the editor's Save
// after recoloring). It is a no-op for a built-in or when no store is attached.
func (s *Session) SaveActiveTheme() error {
	active := s.Themes().Active()
	if !active.Kind().Editable() {
		return nil
	}
	return s.persistTheme(active)
}

// persistTheme writes one custom theme when a store is attached (else a no-op).
func (s *Session) persistTheme(t *theme.Theme) error {
	if s.themeStore == nil {
		return nil
	}
	return s.themeStore.SaveTheme(t)
}

// persistActive writes the selected-theme preference when a store is attached.
func (s *Session) persistActive() error {
	if s.themeStore == nil {
		return nil
	}
	return s.themeStore.SaveActive(s.Themes().ActiveName())
}
