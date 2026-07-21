// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The convex same-sense trihedral (oblique run-off) corner treatment (OCCT tests/blend/simple A8/A6) is
// accumulated by the unified pass (fillet_corner_setback_unified.go): classifyBlendCorner tags a corner
// convexRunoff via convexTrihedralCornerBands + allBodyFacesPlanar, and accumulateConvexRunoff clips
// each band's oblique run-off end via setbackObliqueRunoffEnds (the reused rail-slide helper). The gate
// + per-end predicates below (convexTrihedralCornerBands, allBodyFacesPlanar, endOvershootsRunoff,
// setbackObliqueRunoffEnds) are the reused helpers the classifier + accumulate own; the old
// adoptConvexWedgeSetback entrypoint was folded into that pass.

// convexTrihedralCornerBands returns the three bands of the corner at vid and ok=true when it is the
// A8-class corner this pass owns: exactly three CONVEX (not ef.flip) fillet ends meeting three PLANAR
// faces (orthogonal or not). Any other config (concave/mixed sense, non-planar host, valence != 3)
// returns ok=false so the corner is left untouched.
func convexTrihedralCornerBands(vid uint64, cb *cornerBlend, fils []edgeFillet) ([]cornerBand, bool) {
	if cb == nil || cb.vertex == nil {
		return nil, false
	}
	bands := cornerBandsAt(vid, fils)
	if len(bands) != 3 || anyConcaveBand(bands) {
		return nil, false
	}
	faces := mixedCornerFaces(bands[0], bands[1:])
	if len(faces) != 3 || !allFacesPlanar(faces) {
		return nil, false
	}
	return bands, true
}

// anyConcaveBand reports whether any band is a concave (ef.flip) fillet — the same-sense CONVEX corner
// requires all three convex (a concave/mixed corner is a sphere/torus handled by the P2/P3 passes).
func anyConcaveBand(bands []cornerBand) bool {
	for _, b := range bands {
		if b.concave {
			return true
		}
	}
	return false
}

// allFacesPlanar reports whether every face is a plane (the run-off pierce is only defined against a
// planar far face; a curved host is left to the honest-reject baseline).
func allFacesPlanar(faces []*topo.Face) bool {
	for _, f := range faces {
		if _, ok := f.Geometry().(geom.Plane); !ok {
			return false
		}
	}
	return true
}

// allBodyFacesPlanar reports whether every face of the body is a plane — a polyhedral wedge/box, the
// derivation's planar-host scope. A curved-host body (e.g. F7's elliptical prism) is left untouched:
// its oblique band runs off against a curved neighbour so the flat-plane pierce would move it AWAY from
// OCCT (verified: F7 base rel −0.082% → −0.106% when clipped). Keeping the pass to polyhedral bodies is
// the clean scope boundary between A8/A6 (all-planar wedges, clip TOWARD OCCT) and F7.
func allBodyFacesPlanar(body *topo.Body) bool {
	return allFacesPlanar(body.Faces())
}

// setbackObliqueRunoffEnds runs the trihedral rail-setback on BOTH ends of a convex wedge band and
// reports whether either end moved. The corner (blend) end is a no-op (opposFarPlane rejects a blend),
// and a perpendicular simple end is a no-op (railPierce t=0); only an oblique run-off end onto a planar
// unfilleted face actually clips. Both ends are evaluated (no short-circuit) so a band that runs off
// obliquely at both ends is fully re-terminated.
func setbackObliqueRunoffEnds(ef *edgeFillet) bool {
	moved0 := setbackRunoffEnd(ef, &ef.c0)
	moved1 := setbackRunoffEnd(ef, &ef.c1)
	return moved0 || moved1
}

// setbackRunoffEnd clips corner c's flank rails to their far-plane pierce (setbackTrihedralCorner) —
// but ONLY when the rails currently OVERSHOOT that plane (endOvershootsRunoff), i.e. a tab sticks out
// past the un-filleted run-off face. It reports whether a tangent point actually moved past the
// model-relative weld tolerance, so a no-op end (blend corner, perpendicular pierce, or a rail already
// inside the run-off plane) does not spuriously mark the pass as fired.
//
// The overshoot guard is the load-bearing discriminant between A8/A6 (an OBLIQUE band running off onto
// an un-filleted face overshoots the plane by ~r → a tab OUTSIDE the body → clip TOWARD OCCT) and a
// case like F7 (its run-off rails already sit ON/inside the plane → the pierce would REMOVE real
// material, moving AWAY from OCCT). Without it the pass wrongly perturbs green-by-tolerance F7.
func setbackRunoffEnd(ef *edgeFillet, c *corner) bool {
	if !endOvershootsRunoff(c) {
		return false
	}
	ta0, tb0 := c.ta, c.tb
	_ = setbackTrihedralCorner(ef, c) // never errors on a trihedral end; a non-simple/parallel end is a no-op
	tol := ResolutionForPoints([]math.Point3{ta0, tb0}).Weld()
	return c.ta.DistanceTo(ta0) > tol || c.tb.DistanceTo(tb0) > tol
}

// endOvershootsRunoff reports whether corner c is a simple PLANAR end whose flank rails run PAST the
// run-off plane — a tab OUTSIDE the body: n·(ta−q) or n·(tb−q) > weld, with n the outward normal of the
// end plane and q the end vertex (which lies on the plane). A perpendicular or interior end does not
// overshoot (n·(·−q) ≤ 0), so the clip never fires there; only the oblique tab does. Blend/miter/curved
// ends are rejected upstream by opposFarPlane.
func endOvershootsRunoff(c *corner) bool {
	n, q, ok := opposFarPlane(c)
	if !ok {
		return false
	}
	tol := ResolutionForPoints([]math.Point3{c.ta, c.tb}).Weld()
	return n.Dot(q.VectorTo(c.ta)) > tol || n.Dot(q.VectorTo(c.tb)) > tol
}
