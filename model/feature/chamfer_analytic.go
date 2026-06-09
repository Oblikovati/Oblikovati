// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// analyticChamferTol is the axial tolerance matching a selected circular edge to a cylinder rim.
const analyticChamferTol = 1e-6

// analyticCylinderChamfer bevels the rim(s) of a SIMPLE analytic cylinder (an extruded circle /
// revolved disc) with a TRUE conical chamfer: it rebuilds the body as a surface of revolution whose
// meridian gains a 45° cone segment at each selected rim, so the chamfer is one analytic geom.Cone
// face — not the faceted many-sided frustum the wedge-cut path leaves (Oblikovati/Oblikovati#127).
//
// It applies only when the body is exactly one cylinder + 2 caps and every selected edge is one of
// its two circular rims, with 0 < dist < min(radius, height). Otherwise ok is false and the caller
// keeps the general (re-faceted wedge-cut) chamfer, so complex bodies are untouched.
func analyticCylinderChamfer(body *topo.Body, edges []*topo.Edge, dist float64, feat string) (*topo.Body, bool) {
	cyl, base, height, ok := simpleCylinderParams(body)
	if !ok || dist <= 0 || dist >= height || dist >= cyl.Radius {
		return nil, false
	}
	bottom, top, ok := selectedRims(edges, base, cyl.AxisDir, height)
	if !ok {
		return nil, false // a non-rim edge: not an analytic-cylinder chamfer
	}
	mer := chamferedCylinderMeridian(cyl.Radius, height, dist, bottom, top)
	out, err := brep.SolidOfRevolution(base, cyl.AxisDir.AsVector(), mer, originalFeature(body, feat))
	if err != nil || out == nil {
		return nil, false
	}
	return out, true
}

type cylinderRimKind int

const (
	rimNone cylinderRimKind = iota
	rimBottom
	rimTop
)

// selectedRims classifies the selected edges as the bottom and/or top circular rims of the cylinder.
// ok is false if any edge is not one of the two rims, so the analytic chamfer/fillet fast path
// declines and the caller keeps the general wedge/blend.
func selectedRims(edges []*topo.Edge, base math.Point3, axis math.UnitVector3, height float64) (bottom, top, ok bool) {
	for _, e := range edges {
		switch cylinderRim(e, base, axis, height) {
		case rimBottom:
			bottom = true
		case rimTop:
			top = true
		default:
			return false, false, false
		}
	}
	return bottom, top, true
}

// cylinderRim classifies a selected edge as the bottom (z≈0) or top (z≈height) circular rim of the
// cylinder, or rimNone if it is not a circle on a rim.
func cylinderRim(e *topo.Edge, base math.Point3, axis math.UnitVector3, height float64) cylinderRimKind {
	c, ok := e.Geometry().(geom.Circle)
	if !ok {
		return rimNone
	}
	z := float64(base.VectorTo(c.Center).Dot(axis.AsVector()))
	switch {
	case stdmath.Abs(z) <= analyticChamferTol:
		return rimBottom
	case stdmath.Abs(z-height) <= analyticChamferTol:
		return rimTop
	default:
		return rimNone
	}
}

// chamferedCylinderMeridian returns the (radius,height) meridian of a cylinder (radius r, height h)
// with a 45° chamfer of setback d on the selected rims: the rim vertex (r at z=0 or z=h) is split
// into two — one set back d along the cap (radius r−d) and one along the wall (offset d in z) —
// joined by a cone segment.
func chamferedCylinderMeridian(r, h, d float64, bottom, top bool) []math.Point2 {
	mer := []math.Point2{math.P2(0, 0)}
	if bottom {
		mer = append(mer, math.P2(r-d, 0), math.P2(r, d))
	} else {
		mer = append(mer, math.P2(r, 0))
	}
	if top {
		mer = append(mer, math.P2(r, h-d), math.P2(r-d, h))
	} else {
		mer = append(mer, math.P2(r, h))
	}
	return append(mer, math.P2(0, h))
}
