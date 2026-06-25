// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
)

// Cylinder-side split (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). The general looped split tangles
// on a FULL periodic cylinder side: its seam runs up one constant angle, and when that seam lands in the
// kept region the two rim circles' kept arcs join across it into runs the imprint cannot bridge. The
// cylinder is the degenerate cone (constant radius), so its side cut is now built by the SAME (u,v)
// arrangement split the cone uses (cylinderSideUVSplit, curved_halfspace_ruled_uv.go): the axis-parallel
// flat is a non-wrapping span whose two ends are the vertical cut lines, and an OBLIQUE cut (a section
// ellipse — the case the old line-only split deferred to CSG) is the same within-band / clips-rim / tongue
// family the cone handles. This file keeps only the band extractor the split builds on. After the first
// cut the side is a seam-free arc band, so every later cut composes through the general loopedSplit.

// fullCylinderSideBand reports whether f is the GENUINE full periodic cylinder side — a geom.Cylinder
// bounded by TWO closed-circle rims and the seam — and returns the cylinder plus its band (axial distance
// v measured from the bottom rim centre, so vMin=0 and vMax the height; both rim radii are R). An
// already-trimmed band (one circle plus an ellipse/arc rim, after a first cut) fails the two-circle test
// here and falls through to loopedSplit, so a later clearing plane composes instead of re-entering here.
func fullCylinderSideBand(f curvedFace) (geom.Cylinder, coneSideBand_, bool) {
	cyl, ok := f.surface.(geom.Cylinder)
	if !ok || len(f.loops) == 0 {
		return geom.Cylinder{}, coneSideBand_{}, false
	}
	var circles []geom.Circle
	for _, le := range f.loops[0].edges {
		if c, isCircle := le.curve.(geom.Circle); isCircle && isFullDomain(le.t0, le.t1) {
			circles = append(circles, c)
		}
	}
	if len(circles) != 2 {
		return geom.Cylinder{}, coneSideBand_{}, false
	}
	axis := cyl.AxisDir.AsVector()
	if float64(circles[0].Center.VectorTo(circles[1].Center).Dot(axis)) < 0 {
		circles[0], circles[1] = circles[1], circles[0] // order low→high along the axis
	}
	height := float64(circles[0].Center.VectorTo(circles[1].Center).Dot(axis))
	band := coneSideBand_{
		bottom: circles[0].Center, top: circles[1].Center,
		bottomCirc: circles[0], topCirc: circles[1],
		vMin: 0, vMax: height, rBot: cyl.Radius, rTop: cyl.Radius,
	}
	return cyl, band, true
}
