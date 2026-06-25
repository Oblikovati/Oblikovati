// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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

// coneHalfSpace keeps the axial band of the cone on the plane's negative side, rebuilt as a fresh cone or
// frustum (exact geom.Cone side + planar caps). The plane must be perpendicular to the axis. A plane clear
// of the kept band keeps the solid whole or empties it.
func coneHalfSpace(body *topo.Body, cone geom.Cone, vMin, vMax float64, plane geom.Plane) (*topo.Body, error) {
	n := unit(plane.Normal())
	axis := cone.AxisDir.AsVector()
	along := float64(n.Dot(axis))
	vCut := float64(cone.Apex.VectorTo(plane.Origin).Dot(axis))
	vLo, vHi := keptConeBand(vCut, vMin, vMax, along > 0)
	// Apex-distance band lengths are model-relative (#1399).
	axialTol := geom.ResolutionForBox(body.RangeBox()).Plane()
	if vHi-vLo <= axialTol {
		return topo.MergeBodies(topo.NewLineage(topo.Tok("halfspace", "empty", 0)), true), nil
	}
	if vLo <= vMin+axialTol && vHi >= vMax-axialTol {
		return body, nil // plane clears the cone on the kept side
	}
	t := stdmath.Tan(cone.HalfAngle)
	bottom := cone.Apex.TranslateBy(axis.Scale(math.Scalar(vLo)))
	top := cone.Apex.TranslateBy(axis.Scale(math.Scalar(vHi)))
	return SolidCylinderCone(bottom, top, vLo*t, vHi*t, "halfspace")
}

// keptConeBand returns the [vLo, vHi] apex-distance interval (within [vMin, vMax]) kept on the plane's
// negative side. With n along +axis the negative side is toward the apex (v ≤ cut); with n along −axis it
// is toward the base (v ≥ cut).
func keptConeBand(vCut, vMin, vMax float64, nAlongAxis bool) (vLo, vHi float64) {
	vCut = stdmath.Max(vMin, stdmath.Min(vMax, vCut))
	if nAlongAxis {
		return vMin, vCut
	}
	return vCut, vMax
}

// perpendicularToConeAxis reports whether the cut plane normal is parallel to the cone axis (a
// constant-apex-distance cut), the only orientation the fast cone path handles.
func perpendicularToConeAxis(n math.Vector3, cone geom.Cone) bool {
	return stdmath.Abs(float64(n.Dot(cone.AxisDir.AsVector()))) >= 1-cylinderAxisCosTol
}
