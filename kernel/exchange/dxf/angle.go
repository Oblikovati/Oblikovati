// SPDX-License-Identifier: GPL-2.0-only

package dxf

import "math"

// DXF stores ARC start/end angles in DEGREES (group codes 50/51) but ELLIPSE start/end as
// parametric angles in RADIANS (codes 41/42) — a classic asymmetry. The neutral drawing
// model keeps every angle in radians, so the ARC codec converts at the boundary while the
// ELLIPSE codec does not. These helpers make that conversion explicit at both call sites.

const degPerRad = 180 / math.Pi

// degToRad converts a DXF degree angle (ARC) to the radians the model uses.
func degToRad(deg float64) float64 { return deg / degPerRad }

// radToDeg converts a model radian angle to the degrees a DXF ARC stores.
func radToDeg(rad float64) float64 { return rad * degPerRad }
