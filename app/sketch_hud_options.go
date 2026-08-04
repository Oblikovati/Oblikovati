// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/app/options"
)

// The in-canvas sketch input configuration (#2014). Two separate surfaces are offered while
// geometry is placed: pointer input, which enters a shape's first point by coordinate, and
// dimension input, which sizes the shape being placed. They answer different questions, so each
// can be switched off without the other.
//
// The session already carried a single on/off for the heads-up display (#790); these settings
// refine it and persist alongside it in the application options.

// HUDOptions returns the in-canvas sketch input configuration.
//
//	opts := s.HUDOptions()
//	opts.CreateDimensionsOnValueInput = false
//	_ = s.SetHUDOptions(opts)
func (s *Session) HUDOptions() types.HeadsUpDisplayOptions {
	o := s.appOptions.Sketch
	return types.HeadsUpDisplayOptions{
		Enabled:                              s.hudEnabled,
		PointerInputEnabled:                  o.PointerInput,
		PointerInputInCartesianCoordinates:   o.PointerInputCartesian,
		DimensionInputEnabled:                o.DimensionInput,
		DimensionInputInCartesianCoordinates: o.DimensionInputCartesian,
		CreateDimensionsOnValueInput:         o.CreateDimensionsOnValueInput,
	}
}

// SetHUDOptions replaces the in-canvas sketch input configuration and persists it. Clearing
// Enabled also drops any half-typed entry, so the panels do not reappear mid-value.
func (s *Session) SetHUDOptions(o types.HeadsUpDisplayOptions) error {
	s.hudEnabled = o.Enabled
	s.appOptions.Sketch.HeadsUpDisplay = o.Enabled
	s.appOptions.Sketch.PointerInput = o.PointerInputEnabled
	s.appOptions.Sketch.PointerInputCartesian = o.PointerInputInCartesianCoordinates
	s.appOptions.Sketch.DimensionInput = o.DimensionInputEnabled
	s.appOptions.Sketch.DimensionInputCartesian = o.DimensionInputInCartesianCoordinates
	s.appOptions.Sketch.CreateDimensionsOnValueInput = o.CreateDimensionsOnValueInput
	if !o.Enabled {
		s.sketchHUD = sketchHUD{}
		s.placementFields = placementFieldState{}
	}
	return s.saveOptions()
}

// DimensionInputEnabled reports whether the in-place dimension boxes should be shown and take
// keystrokes while a shape is placed. The head gates both drawing and key routing on it.
func (s *Session) DimensionInputEnabled() bool {
	return s.hudEnabled && s.appOptions.Sketch.DimensionInput
}

// CreateDimensionsOnValueInput reports whether a value typed into a dimension box becomes a
// persistent driving dimension when the shape commits. With it off the typed value still sizes
// the shape but states nothing afterwards.
func (s *Session) CreateDimensionsOnValueInput() bool {
	return s.appOptions.Sketch.CreateDimensionsOnValueInput
}

// sketchOptionsFromHUD is the options-group form of a HUD configuration, used when the whole
// sketch group is rewritten (PersistLiveOptions).
func sketchOptionsFromHUD(o types.HeadsUpDisplayOptions, group options.Sketch) options.Sketch {
	group.HeadsUpDisplay = o.Enabled
	group.PointerInput = o.PointerInputEnabled
	group.PointerInputCartesian = o.PointerInputInCartesianCoordinates
	group.DimensionInput = o.DimensionInputEnabled
	group.DimensionInputCartesian = o.DimensionInputInCartesianCoordinates
	group.CreateDimensionsOnValueInput = o.CreateDimensionsOnValueInput
	return group
}
