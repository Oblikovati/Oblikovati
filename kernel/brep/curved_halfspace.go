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

// Since ADR-0062 the cut is not an operation of its own: it is a DIFFERENCE against the plane's
// positive side, bounded to the target's box. OCCT says the same thing in code —
// BRepPrimAPI_MakeHalfSpace builds exactly that solid and hands it to the ordinary BOP — and
// solvespace's SShell::MakeFromBoolean trims BOTH shells against the same shared intersection curves.
// Neither kernel synthesises one side's boundary out of the other side's leftovers, which is what the
// pipeline below did: it split only the target, asked each wall for the section arcs it had trimmed on,
// and built a lid from them.

// ErrUnsupportedHalfSpace reports a body the half-space cut does not handle. It is the difference's own
// decline, surfaced under the name the callers (ops/boolean's convex and flat-subtract compositions,
// ops/surface's directed sculpt) already switch on.
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
	cut, err := Boolean(Difference, body, BoundedHalfSpace(plane, body.RangeBox()))
	if err != nil {
		return nil, errors.Join(ErrUnsupportedHalfSpace, err)
	}
	if cut == nil {
		// The plane's positive side covers the body: nothing is left. The contract is an EMPTY body, not
		// a nil one — callers compose the cut (one piece per face of a convex tool) and read Faces() on
		// every piece.
		return topo.NewBuilder(true, topo.NewLineage(topo.Tok("halfspace", "body", 0))).Build(), nil
	}
	return cut, nil
}

// BoundedHalfSpace is the plane's POSITIVE side as an ordinary solid: a prism whose base lies in the
// plane and which extends a box-diagonal past every corner of box, so within box it IS the half-space.
// It is what a half-space cut subtracts, and what OCCT's BRepPrimAPI_MakeHalfSpace builds.
//
// The extent comes from the box's own diagonal, so the tool scales with the target and carries no
// absolute size. box must be the target's range box.
//
// Example — the +z half-space over a body:
//
//	rest, _ := brep.Boolean(brep.Difference, body, brep.BoundedHalfSpace(plane, body.RangeBox()))
func BoundedHalfSpace(plane geom.Plane, box math.Box) *topo.Body {
	d := math.Scalar(box.Diagonal().Length())
	n := unit(plane.Normal())
	// The base is the box centre projected into the plane, so the prism covers the box whichever way
	// the plane's own origin sits relative to it.
	base := box.Center().TranslateBy(n.Scale(-math.Scalar(float64(plane.Origin.VectorTo(box.Center()).Dot(n)))))
	u, v := plane.UAxis.AsVector(), plane.VAxis.AsVector()
	var corners [8]math.Point3
	for i := range 8 {
		p := base.TranslateBy(u.Scale(pick(i&1 != 0, d, -d))).TranslateBy(v.Scale(pick(i&2 != 0, d, -d)))
		if i&4 != 0 {
			p = p.TranslateBy(n.Scale(d))
		}
		corners[i] = p
	}
	// (UAxis, VAxis, Normal) is right-handed — Normal IS UAxis×VAxis — so the block corner convention
	// gives the same outward winding here that it gives an axis-aligned box.
	return hexahedronBody(corners, "halfspace")
}
