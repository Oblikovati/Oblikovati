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
