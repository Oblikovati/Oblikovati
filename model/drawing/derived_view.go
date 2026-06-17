// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"math"

	"oblikovati.org/kernel/hlr"
	gmath "oblikovati.org/math"
)

// Derived drawing views (M14-F02 PBI-140, #387): view kinds projected from a parent view
// rather than from a standard orientation. An auxiliary view folds the model about a line
// drawn on the parent and projects perpendicular to it, so an inclined face appears true-size.
// Section/detail/break build on the same parent-derived machinery in later increments.

// auxiliaryBasis derives an auxiliary view's projection frame from its parent frame and the
// fold-line angle (radians, measured from the parent's screen X axis). The fold line is an
// in-plane axis of the parent; the auxiliary looks perpendicular to it (rotating the parent's
// view direction 90° about the fold axis), with the fold line as the auxiliary's screen X.
//
// A fold angle of 0 (fold line along the parent's X) yields the downward projection (the top
// view of a front parent); π/2 yields the sideways projection — so it generalises the four
// discrete projected directions to any angle.
func auxiliaryBasis(parent hlr.View, angleRad float64, origin gmath.Point3) hlr.View {
	foldAxis := parent.XAxis.Scale(math.Cos(angleRad)).Add(parent.YAxis.Scale(math.Sin(angleRad)))
	viewDir := foldAxis.Cross(parent.ViewDir) // ⟂ fold line, rotated 90° from the parent's view dir
	// upHint = parent.ViewDir makes the fold axis the auxiliary's screen X (see hlr.NewView).
	return hlr.NewView(origin, viewDir, parent.ViewDir)
}
