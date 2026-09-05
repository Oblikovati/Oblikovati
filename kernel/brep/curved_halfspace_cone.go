// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
)

// Cone half-space cut (M2 Phase 1, Oblikovati/Oblikovati#1334). Trims an analytic cone or frustum (one
// geom.Cone side + one or two planar caps) by a plane PERPENDICULAR to its axis: the kept axial band is
// itself a cone or frustum, rebuilt exactly via SolidCylinderCone (the apex side keeps a smaller cone, the
// base side a frustum). An oblique plane (an ellipse/parabola/hyperbola section) or a plane through the
// side defers to the general arrangement — planeConeCurve only solves the perpendicular circle — so it
// returns ErrUnsupportedHalfSpace and the caller keeps the CSG fallback. Mirrors cylinderHalfSpace.

// coneSolidParams recovers a cone/frustum's geom.Cone and its axial extent (distances from the apex along
// the axis to the lowest and highest caps) from a body's faces. ok=false unless the body is exactly one
// cone side plus one cap (a full cone, apex at v=0) or two caps (a frustum) — the bare shapes this path
// rebuilds.
func coneSolidParams(faces []curvedFace) (cone geom.Cone, vMin, vMax float64, ok bool) {
	var caps []geom.Plane
	found := false
	for _, f := range faces {
		switch s := f.surface.(type) {
		case geom.Cone:
			cone, found = s, true
		case geom.Plane:
			caps = append(caps, s)
		default:
			return geom.Cone{}, 0, 0, false
		}
	}
	if !found || len(caps) == 0 || len(caps) > 2 {
		return geom.Cone{}, 0, 0, false
	}
	vMin, vMax = coneCapExtent(cone, caps)
	return cone, vMin, vMax, true
}

// coneCapExtent returns the apex-distance band [vMin, vMax] the solid spans: a full cone runs from the
// apex (v=0) to its one cap; a frustum between its two caps.
func coneCapExtent(cone geom.Cone, caps []geom.Plane) (vMin, vMax float64) {
	axis := cone.AxisDir.AsVector()
	v0 := float64(cone.Apex.VectorTo(caps[0].Origin).Dot(axis))
	if len(caps) == 1 {
		return 0, v0 // full cone: apex at v=0 up to the base cap
	}
	v1 := float64(cone.Apex.VectorTo(caps[1].Origin).Dot(axis))
	return stdmath.Min(v0, v1), stdmath.Max(v0, v1)
}
