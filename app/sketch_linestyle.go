// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/sketch"
)

// SketchEntityPattern picks the .lin dash pattern a sketch entity renders with
// (issue #161): centerlines use the center style, construction geometry the dashed
// style, and normal geometry the sketch's line-type override (built-in or loaded
// custom). nil means draw solid.
//
//	pattern := app.SketchEntityPattern(sk, entity)
func SketchEntityPattern(sk *sketch.Sketch, e sketch.Entity) []float64 {
	if l, ok := e.(*sketch.Line); ok && l.IsCenterline() {
		return linetype.Builtin(types.SketchLineCenter)
	}
	if c, ok := e.(interface{ IsConstruction() bool }); ok && c.IsConstruction() {
		return linetype.Builtin(types.SketchLineDashed)
	}
	return sketchOverridePattern(sk)
}

// sketchOverridePattern resolves the sketch-level line-type override into a pattern.
func sketchOverridePattern(sk *sketch.Sketch) []float64 {
	t := types.SketchLineType(sk.LineType())
	if t != types.SketchLineCustom {
		return linetype.Builtin(t)
	}
	if d, _, ok := sk.CustomLineType(); ok {
		return d.Pattern
	}
	return nil
}

// EntityStyle is how one sketch entity draws: its dash pattern, colour and stroke width. The
// colour and weight are zero when the entity takes the sketch defaults, which is the common case.
type EntityStyle struct {
	Pattern    []float64
	Color      types.Color
	LineWeight float64
}

// SketchEntityStyle resolves an entity's draw style, applying its per-entity format overrides on
// top of the sketch-level pattern (#2015).
//
// suppress is the Show Format toggle: when set, the overrides are ignored and the entity draws
// with default attributes. That is the documented behaviour of the button, whose "on" state shows
// the DEFAULT format rather than the user's.
//
//	style := app.SketchEntityStyle(sk, entity, s.ShowFormat())
func SketchEntityStyle(sk *sketch.Sketch, e sketch.Entity, suppress bool) EntityStyle {
	style := EntityStyle{Pattern: SketchEntityPattern(sk, e)}
	if suppress {
		return style
	}
	f, ok := sk.EntityFormat(e.EntityID())
	if !ok {
		return style
	}
	if f.LineType != "" {
		style.Pattern = linetype.Builtin(types.SketchLineType(f.LineType))
	}
	style.Color, style.LineWeight = f.Color, f.LineWeight
	return style
}
