// SPDX-License-Identifier: GPL-2.0-only

package app

import "github.com/Oblikovati/oblikovati/model/sketch"

// The Sketch tab's Format panel: toggle the selected sketch geometry between normal and
// construction, or mark/unmark selected lines as centerlines (an axis for revolve/mirror).

// ToggleConstruction flips the construction flag on every selected sketch curve, returning how
// many it changed. Construction geometry shapes constraints but is excluded from profiles.
func (s *Session) ToggleConstruction() int {
	n := 0
	for _, it := range s.Selection().Items() {
		h, ok := it.(SketchEntityHandle)
		if !ok {
			continue
		}
		if c, ok := h.Entity.(interface {
			IsConstruction() bool
			SetConstruction(bool)
		}); ok {
			c.SetConstruction(!c.IsConstruction())
			n++
		}
	}
	return n
}

// ToggleCenterline flips the centerline flag on every selected sketch line (an axis for
// revolve/mirror/symmetry; centerlines are always construction).
func (s *Session) ToggleCenterline() int {
	n := 0
	for _, it := range s.Selection().Items() {
		h, ok := it.(SketchEntityHandle)
		if !ok {
			continue
		}
		if l, ok := h.Entity.(*sketch.Line); ok {
			l.SetCenterline(!l.IsCenterline())
			n++
		}
	}
	return n
}

// sketchFormatCommands is the Sketch tab's Format panel: toggle Construction / Centerline on the
// selected sketch geometry (matching Inventor's Format panel).
func sketchFormatCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Sketch.Construction", "Construction", "Format", func(s *Session) error {
			s.ToggleConstruction()
			return nil
		}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).
			WithIcon("construction").WithButtonStyle(SmallIconButton).
			WithTooltip("Construction — toggle the selected geometry as construction (excluded from profiles)."),
		NewCommand("Sketch.Centerline", "Centerline", "Format", func(s *Session) error {
			s.ToggleCenterline()
			return nil
		}).WithTab("Sketch").WithEnvironment(SketchEnvironment).WithEnable(inSketch).
			WithIcon("centerline").WithButtonStyle(SmallIconButton).
			WithTooltip("Centerline — toggle the selected line(s) as a centerline axis (revolve/mirror)."),
	}
}
