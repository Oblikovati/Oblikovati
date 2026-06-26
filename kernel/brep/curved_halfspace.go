// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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
	// One model-relative coincidence scale for this cut, derived from the body's extent
	// (Oblikovati/Oblikovati#1399); threaded into the general split so the analytic imprint's
	// clearance/grazing tests scale with the part rather than reading a cm-anchored epsilon.
	res := geom.ResolutionForBox(body.RangeBox())
	if cyl, base, height, ok := cylinderSolidParams(faces); ok && perpendicularToAxis(n, cyl) {
		return cylinderHalfSpace(body, cyl, base, height, plane) // ⟂ cut → shorter cylinder, fast path
	}
	if cone, vMin, vMax, ok := coneSolidParams(faces); ok && perpendicularToConeAxis(n, cone) {
		return coneHalfSpace(body, cone, vMin, vMax, plane) // ⟂ cut → cone/frustum, fast path
	}
	if torus, ok := torusSolidParams(faces); ok {
		if cut, handled, err := torusHalfSpaceCut(body, torus, plane, n); handled {
			return cut, err
		}
	}
	return generalHalfSpace(body, plane, n, faces, res)
}

// torusHalfSpaceCut dispatches a bare torus's analytic half-space cuts by the section the plane carves:
// a perpendicular cut (two concentric circles → a band + annular lid), and the axis-parallel spiric cuts
// (a single oval cap, its genus-1 complement, or a two-oval band through the hole — Oblikovati#1375).
// handled=false leaves an oblique/spiric topology not yet wired to the general CSG fallback.
func torusHalfSpaceCut(body *topo.Body, torus geom.Torus, plane geom.Plane, n math.Vector3) (*topo.Body, bool, error) {
	switch {
	case perpendicularToTorusAxis(n, torus):
		res, err := torusHalfSpace(body, torus, plane) // ⟂ cut → trimmed torus band + annular lid
		return res, true, err
	// The single-oval CAP now routes through the unified (u,v)-arrangement trimmer (torusSideSplit, via
	// generalHalfSpace's handled=false fallthrough below), the first spiric case migrated off the bespoke
	// ladder (#1406). The remaining spiric cases stay analytic here until their migration step.
	case torusSingleOvalComplement(torus, plane):
		res, err := torusComplementHalfSpace(torus, plane) // axis-∥ → genus-1 complement + planar lid
		return res, true, err
	case torusTwoOvalBand(torus, plane):
		res, err := torusTwoOvalHalfSpace(torus, plane) // axis-∥ through the hole → v-wrapping band + 2 lids
		return res, true, err
	case torusObliqueOval(torus, plane):
		res, err := torusObliqueOvalHalfSpace(torus, plane) // tilted single-oval bite → cap or complement
		return res, true, err
	case torusTwoObliqueOval(torus, plane):
		res, err := torusTwoOvalHalfSpace(torus, plane) // tilted cut through the hole → v-wrapping band + 2 lids
		return res, true, err
	}
	return nil, false, nil
}
