// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Coaxial cylinder union (M2 Phase 3, Oblikovati/Oblikovati#1336 — the coplanar/tangent curved-overlap
// case). Two coaxial same-radius cylinders that overlap or abut along their shared axis have COINCIDENT
// side surfaces (both lie on the very same geom.Cylinder) and their inner caps fall inside the other body
// — the curved analogue of a coplanar overlap. The general boolean sends this to triangle-soup CSG, which
// faceted it (a coaxial-cylinder union came back ~2.5% under volume with zero analytic faces). The union
// is exactly one taller cylinder spanning the merged axial interval, so build that directly and keep the
// analytic surface. Anything off the coaxial-same-radius-overlapping case (different radius, skew axis, an
// axial gap) returns ok=false so the caller keeps its fallback.

// CoaxialCylinderUnion returns a ∪ b when a and b are coaxial, equal-radius cylinders overlapping or
// abutting along the axis — one cylinder spanning their merged extent — or ok=false to defer.
//
// Example:
//
//	a, _ := brep.SolidCylinder(math.P3(0,0,0), math.V3(0,0,1), 2, 4) // z 0..4
//	b, _ := brep.SolidCylinder(math.P3(0,0,3), math.V3(0,0,1), 2, 4) // z 3..7
//	u, ok := brep.CoaxialCylinderUnion(a, b)                          // ok: one cylinder z 0..7
func CoaxialCylinderUnion(a, b *topo.Body) (*topo.Body, bool) {
	ca, baseA, hA, okA := cylinderSolidParams(facesOfAny(a))
	cb, baseB, hB, okB := cylinderSolidParams(facesOfAny(b))
	if !okA || !okB || !coaxialEqualRadius(ca, baseA, cb, baseB) {
		return nil, false
	}
	axis := ca.AxisDir.AsVector()
	loA, loB := axialOffset(baseA, baseA, axis), axialOffset(baseA, baseB, axis)
	lo, hi := stdmath.Min(loA, loB), stdmath.Max(loA+hA, loB+hB)
	if hi-lo > hA+hB+geom.ResolutionForSize(hA+hB).Plane() { // model-relative axial-gap test (#1399)
		return nil, false // an axial gap between the two — not a single solid, leave it to MergeBodies
	}
	union, err := SolidCylinder(baseA.TranslateBy(axis.Scale(math.Scalar(lo))), axis, ca.Radius, hi-lo)
	if err != nil {
		return nil, false
	}
	return union, true
}

// coaxialEqualRadius reports whether two cylinders share the same axis line (parallel directions, each
// base on the other's axis) and the same radius — the precondition for their side faces to be coincident.
func coaxialEqualRadius(ca geom.Cylinder, baseA math.Point3, cb geom.Cylinder, baseB math.Point3) bool {
	ua, ub := ca.AxisDir.AsVector(), cb.AxisDir.AsVector()
	if stdmath.Abs(float64(ua.Dot(ub))) < 1-1e-7 { // tol:angular — axes-parallel cosine
		return false // axes not parallel
	}
	if !nearEqual(ca.Radius, cb.Radius) {
		return false
	}
	// Off-axis coincidence is model-relative (#1399), scaled by the cylinder radius (matching nearEqual).
	return offAxisDistance(baseA, ua, baseB) <= geom.ResolutionForSize(ca.Radius).Plane() // baseB on a's axis line
}

// offAxisDistance returns the perpendicular distance of point p from the line through origin along the unit
// axis ua.
func offAxisDistance(origin math.Point3, ua math.Vector3, p math.Point3) float64 {
	d := origin.VectorTo(p)
	along := ua.Scale(d.Dot(ua))
	return float64(d.Sub(along).Length())
}

// axialOffset returns the signed projection of p onto the axis through origin (the axial coordinate used
// to merge the two extents).
func axialOffset(origin, p math.Point3, ua math.Vector3) float64 {
	return float64(origin.VectorTo(p).Dot(ua))
}
