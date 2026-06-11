// SPDX-License-Identifier: GPL-2.0-only

// Package linetype implements sketch line-type patterns (issue #161): the built-in
// styles behind api/types.SketchLineType, a parser for industry-standard .lin
// line-type definition files, and the pure dash-segmentation used by the head to
// render patterned curves (construction geometry, centerlines, custom styles).
package linetype

import (
	"oblikovati.org/api/types"
)

// Definition is one named line-type pattern. Pattern follows the .lin convention:
// a value > 0 is a pen-down dash, < 0 a pen-up gap, 0 a dot; lengths are model
// units (cm).
type Definition struct {
	Name        string
	Description string
	Pattern     []float64
}

// Builtin returns the dash pattern of a built-in sketch line type (lengths in cm,
// the conventional .lin proportions). Continuous, unknown, and custom kinds return
// nil — the caller draws solid (custom patterns live in a loaded Definition, not
// here).
func Builtin(t types.SketchLineType) []float64 {
	switch t {
	case types.SketchLineDashed:
		return []float64{0.5, -0.25}
	case types.SketchLineHidden:
		return []float64{0.25, -0.125}
	case types.SketchLineCenter:
		return []float64{1.25, -0.25, 0.25, -0.25}
	case types.SketchLinePhantom:
		return []float64{1.25, -0.25, 0.25, -0.25, 0.25, -0.25}
	default:
		return nil
	}
}
