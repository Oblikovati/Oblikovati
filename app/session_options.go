// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/app/options"
	"oblikovati.org/persistence/userprefs"
)

// The session half of M05-F11 (#618): the typed option groups live in
// [oblikovati.org/app/options]; this file binds them to the running state (the
// sketch grid, the chamfer default) and persists edits, whether they arrive from
// the Preferences window or the options.* wire methods.

// Options returns the current application options.
func (s *Session) Options() options.All { return s.appOptions }

// UseOptionsStore installs persistence, loads the stored groups over the defaults,
// and applies the live ones to the running session. The head calls it at startup
// before reading General.StartupAction.
func (s *Session) UseOptionsStore(store options.Store) error {
	loaded, err := store.Load()
	if err != nil {
		return err
	}
	s.optionsStore = store
	s.appOptions = loaded
	s.applySketchOptions(loaded.Sketch)
	s.chamferFlatCorners = loaded.Part.ChamferFlatCorners
	return nil
}

// SetGeneralOptions stores the general group (honored at the next startup).
func (s *Session) SetGeneralOptions(g options.General) error {
	s.appOptions.General = g
	return s.saveOptions()
}

// SetSketchOptions applies and stores the sketch group.
func (s *Session) SetSketchOptions(o options.Sketch) error {
	if o.GridSpacingCm <= 0 {
		return fmt.Errorf("app: grid spacing %v cm is not positive", o.GridSpacingCm)
	}
	s.appOptions.Sketch = o
	s.applySketchOptions(o)
	return s.saveOptions()
}

// SetPartOptions applies and stores the part-defaults group.
func (s *Session) SetPartOptions(p options.Part) error {
	s.appOptions.Part = p
	s.chamferFlatCorners = p.ChamferFlatCorners
	return s.saveOptions()
}

// PersistLiveOptions snapshots the live, UI-edited state (the grid the Preferences
// tab mutates in place, the chamfer toggle) back into the option groups and saves —
// so the Preferences window keeps its direct-edit style and still persists.
func (s *Session) PersistLiveOptions() error {
	g := s.Grid()
	s.appOptions.Sketch = options.Sketch{
		GridSpacingCm:  g.SpacingModel(),
		GridVisible:    g.Visible,
		GridMajorEvery: g.MajorEvery,
		SnapToPoints:   g.SnapToPoints,
		SnapToGrid:     g.SnapToGrid,
	}
	s.appOptions.Part = options.Part{ChamferFlatCorners: s.chamferFlatCorners}
	return s.saveOptions()
}

// applySketchOptions writes the sketch group into the live grid settings.
func (s *Session) applySketchOptions(o options.Sketch) {
	g := s.Grid()
	_ = g.SetSpacingModel(o.GridSpacingCm) // validated by the setters; defaults are positive
	g.Visible = o.GridVisible
	g.MajorEvery = o.GridMajorEvery
	g.SnapToPoints = o.SnapToPoints
	g.SnapToGrid = o.SnapToGrid
}

// saveOptions persists the groups when a store is wired (in-session only otherwise).
func (s *Session) saveOptions() error {
	if s.optionsStore == nil {
		return nil
	}
	return s.optionsStore.Save(s.appOptions)
}

// ViewCubePrefs returns the global display preferences (ViewCube placement and
// visibility) the options.display wire group proxies.
func (s *Session) ViewCubePrefs() userprefs.Prefs { return s.prefs }

// SetViewCubePrefs replaces the global display preferences and persists them
// through the userprefs store (the display group's write path).
func (s *Session) SetViewCubePrefs(p userprefs.Prefs) {
	s.prefs = p
	s.savePrefs()
}
