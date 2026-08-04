// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/sketch"
)

// The Format panel's three selection lists (#2015): line type, colour and thickness, applied to
// the selected geometry. Each list's "Default" entry clears just that field, so an entity can
// override its colour while inheriting its line type; clearing the last override removes the
// entry entirely, keeping absence as the single representation of Default.
//
// These are the values a DWG import carries in from the file's layer table, which is why they
// exist at all.

// SetSelectionLineType sets — or with "" clears — the line type of every selected entity,
// returning how many it changed.
func (s *Session) SetSelectionLineType(name string) int {
	return s.editSelectionFormat(func(f *sketch.EntityFormat) { f.LineType = name })
}

// SetSelectionColor sets the colour of every selected entity. Pass a Color whose Source is
// types.AutomaticColorSource to clear the override.
func (s *Session) SetSelectionColor(c types.Color) int {
	return s.editSelectionFormat(func(f *sketch.EntityFormat) { f.Color = c })
}

// SetSelectionLineWeight sets — or with 0 clears — the stroke width of every selected entity.
func (s *Session) SetSelectionLineWeight(w float64) int {
	return s.editSelectionFormat(func(f *sketch.EntityFormat) { f.LineWeight = w })
}

// editSelectionFormat applies edit to each selected entity's format, reading the current value
// first so the three lists compose rather than overwrite one another.
func (s *Session) editSelectionFormat(edit func(*sketch.EntityFormat)) int {
	sk := s.ActiveSketch()
	if sk == nil {
		return 0
	}
	n := 0
	for _, e := range s.selectedSketchEntities() {
		f, _ := sk.EntityFormat(e.EntityID())
		edit(&f)
		sk.SetEntityFormat(e.EntityID(), f)
		n++
	}
	return n
}

// SelectionFormat is what the three lists display: the format shared by the whole selection.
// Fields that differ across the selection read as Default, so a list never claims a value the
// selection does not uniformly have.
func (s *Session) SelectionFormat() sketch.EntityFormat {
	sk := s.ActiveSketch()
	if sk == nil {
		return sketch.EntityFormat{}
	}
	ents := s.selectedSketchEntities()
	if len(ents) == 0 {
		return sketch.EntityFormat{}
	}
	common, _ := sk.EntityFormat(ents[0].EntityID())
	for _, e := range ents[1:] {
		f, _ := sk.EntityFormat(e.EntityID())
		common = commonFormat(common, f)
	}
	return common
}

// commonFormat keeps only the fields two formats agree on, so a mixed selection reads as Default
// in the lists where it differs.
func commonFormat(a, b sketch.EntityFormat) sketch.EntityFormat {
	var out sketch.EntityFormat
	if a.LineType == b.LineType {
		out.LineType = a.LineType
	}
	if a.Color == b.Color {
		out.Color = a.Color
	}
	if a.LineWeight == b.LineWeight {
		out.LineWeight = a.LineWeight
	}
	return out
}

// --- API seam ---------------------------------------------------------------

// EntityFormatOf returns one entity's formatting overrides, or ok=false when it inherits
// everything. It is the in-proc form of what the api/wire sketch.getEntityFormat serves.
func (s *Session) EntityFormatOf(id sketch.ID) (types.SketchEntityFormat, bool) {
	sk := s.ActiveSketch()
	if sk == nil {
		return types.SketchEntityFormat{}, false
	}
	f, ok := sk.EntityFormat(id)
	if !ok {
		return types.SketchEntityFormat{}, false
	}
	return types.SketchEntityFormat{
		LineType:      types.SketchLineType(f.LineType),
		OverrideColor: f.Color,
		LineWeight:    f.LineWeight,
	}, true
}

// SetEntityFormatOf sets one entity's formatting overrides; a format that overrides nothing
// clears them. It errors when no sketch is open or the id names no entity in it.
func (s *Session) SetEntityFormatOf(id sketch.ID, f types.SketchEntityFormat) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("sketch format: no active sketch")
	}
	if _, ok := sk.EntityByID(id); !ok {
		return fmt.Errorf("sketch format: entity %d is not in the active sketch", id)
	}
	sk.SetEntityFormat(id, sketch.EntityFormat{
		LineType:   string(f.LineType),
		Color:      f.OverrideColor,
		LineWeight: f.LineWeight,
	})
	return nil
}

// FormatModes returns the Format panel's armed creation modes.
func (s *Session) FormatModes() types.SketchFormatModes {
	return types.SketchFormatModes{
		Construction:            s.formatModes.construction,
		Centerline:              s.formatModes.centerline,
		CenterPoint:             s.formatModes.centerPoint,
		DrivenDimension:         s.formatModes.drivenDim,
		SuppressFormatOverrides: s.ShowFormat(),
	}
}

// SetFormatModes replaces the armed creation modes and persists the Show Format toggle, which
// lives with the other sketch application options rather than in session-only state.
func (s *Session) SetFormatModes(m types.SketchFormatModes) error {
	s.formatModes = sketchFormatModes{
		construction: m.Construction,
		centerline:   m.Centerline,
		centerPoint:  m.CenterPoint,
		drivenDim:    m.DrivenDimension,
	}
	if s.ShowFormat() != m.SuppressFormatOverrides {
		s.ToggleShowFormat()
	}
	return nil
}
