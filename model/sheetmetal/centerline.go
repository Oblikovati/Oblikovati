// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

import "oblikovati.org/math"

// CosmeticCenterline is a manufacturing annotation line drawn on the flat pattern (M13-F06,
// #809) — for example a centerline through a hole pattern. It is a plain line segment in the
// flat's 2D coordinates; it carries no geometry, only the annotation.
type CosmeticCenterline struct {
	Start, End math.Point2
}
