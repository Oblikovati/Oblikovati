// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Curved half-space cut (M2 Phase 1, Oblikovati/Oblikovati#1334). The first operation of the
// general curved boolean: trim an analytic curved solid by ONE plane, keeping the part on the
// plane's negative side and closing it with a planar lid — an EXACT curved B-rep (the kept curved
// faces preserve their analytic surface, not a tessellated soup). It is the building block the box
// boolean composes (a convex planar tool is an intersection of half-spaces) and the natural place
// the imprint→split→classify→stitch pipeline starts, restricted to a single planar cutter so
// classification is just "which side of the plane" (no point-in-solid needed yet).

// ErrUnsupportedHalfSpace reports a body the half-space cut does not yet handle (a surface type or
// trim configuration beyond the analytic primitives wired so far) — the caller keeps the BSP-CSG
// fallback, so there is no regression.
var ErrUnsupportedHalfSpace = errors.New("brep: half-space cut handles only the wired analytic curved solids")

// HalfSpaceCut returns the part of body on the NEGATIVE side of plane — the half-space
// {x : normal·(x − origin) ≤ 0} — capped by a planar lid whose outward normal is +plane.Normal.
// The result is an exact curved B-rep. A plane that misses the body returns the whole body (all
// negative) or an empty body (all positive). ErrUnsupportedHalfSpace for an as-yet-unwired solid.
//
// Example — a sphere cut by z=0 keeps the lower hemisphere (a true spherical cap + a disk lid):
//
//	sphere, _ := brep.SolidSphere(math.P3(0,0,0), 5, "s")
//	plane, _ := geom.NewPlane(math.P3(0,0,0), math.V3(0,0,1))
//	cap, _ := brep.HalfSpaceCut(sphere, plane) // lower hemisphere
func HalfSpaceCut(body *topo.Body, plane geom.Plane) (*topo.Body, error) {
	n := unit(plane.Normal())
	faces := facesOfAny(body)
	if cyl, base, height, ok := cylinderSolidParams(faces); ok && perpendicularToAxis(n, cyl) {
		return cylinderHalfSpace(body, cyl, base, height, plane) // ⟂ cut → shorter cylinder, fast path
	}
	if cone, vMin, vMax, ok := coneSolidParams(faces); ok && perpendicularToConeAxis(n, cone) {
		return coneHalfSpace(body, cone, vMin, vMax, plane) // ⟂ cut → cone/frustum, fast path
	}
	if torus, ok := torusSolidParams(faces); ok && perpendicularToTorusAxis(n, torus) {
		return torusHalfSpace(body, torus, plane) // ⟂ cut → trimmed torus band + annular lid
	}
	if torus, ok := torusSolidParams(faces); ok && torusSingleOvalCap(torus, plane) {
		return torusObliqueHalfSpace(torus, plane) // axis-∥ cut → single-oval cap + planar lid
	}
	return generalHalfSpace(body, plane, n, faces)
}
