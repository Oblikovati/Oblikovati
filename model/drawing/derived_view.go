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

// sectionLine is a section view's cut line on its parent, in sheet millimetres.
type sectionLine struct{ x1, y1, x2, y2 float64 }

// sectionBasis derives a section view's cut-plane frame from its parent frame and the cut line
// drawn on the parent. The line (sheet mm) is mapped back through the parent's placement into the
// parent's model plane; the cut plane contains that line and is parallel to the parent's view
// direction (it cuts straight into the part). The section looks along the plane normal — the
// retained half is the side the normal points into.
func sectionBasis(parent hlr.View, line sectionLine, scale, cx, cy float64, origin gmath.Point3) hlr.View {
	// Sheet mm → parent model 2D (cm), inverting place(): (mm - centre) / (scale·cmToMM).
	s := cmToMM * scale
	mx1, my1 := (line.x1-cx)/s, (line.y1-cy)/s
	mx2, my2 := (line.x2-cx)/s, (line.y2-cy)/s
	dirX, dirY := mx2-mx1, my2-my1
	// Cut-plane normal = the line's left normal (-dy, dx), mapped into the parent's plane.
	normal := parent.XAxis.Scale(-dirY).Add(parent.YAxis.Scale(dirX))
	midX, midY := (mx1+mx2)/2, (my1+my2)/2
	planeOrigin := origin.TranslateBy(parent.XAxis.Scale(midX).Add(parent.YAxis.Scale(midY)))
	return hlr.NewView(planeOrigin, normal, parent.ViewDir)
}
