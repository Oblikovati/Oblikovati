// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// adoptConvexWedgeSetback re-terminates the OBLIQUE run-off ends of every band converging on a CONVEX
// SAME-SENSE trihedral corner (3 convex fillets meeting at 3 PLANAR faces — the material-side sphere
// solvePlanarBlend already builds correctly) and returns the re-trimmed body, ok=true, when at least
// one band end actually moved AND the result certifies a watertight hole-contained solid. Otherwise
// ok=false and the caller keeps the baseline.
//
// The gap it closes (OCCT tests/blend/simple A8 +2.75%): A8 is a wedge (triangular prism) whose three
// convex edges meet at one trihedral vertex. The corner itself is already OCCT-exact — the octant
// sphere (frac Ω/4π, area 195.1) and the three corner-side band rails retract to the sphere's tangent
// points, so the non-orthogonal setback s = r·cot(θ/2) at the corner is ALREADY modelled by the shared
// blend path (geometry-derivation §4). The miss is at the OTHER (far) end of the slant-edge band: it
// runs off the filleted edge onto the un-filleted x=0 face. Because that band's axis is OBLIQUE to the
// run-off plane, the simple-end round overshoots the plane (a tab to x=−9.285), inflating the band, the
// top face and the x=0 face by +742 total. OCCT clips the band's flank rails exactly where they PIERCE
// the run-off plane (band top rail → (0,73.07,100), area 1317.09 not 1459.75). The perpendicular-axis
// bands (band1/band3) already land ON their run-off plane, so they are byte-identical.
//
// The fix reuses the runout rail-slide primitive (setbackTrihedralCorner → railPierce): for each band
// of a convex wedge corner it pierces both flank rails against the far simple end's opposite planar
// face. The corner (blend) end is a no-op (opposFarPlane rejects a blend end); a perpendicular simple
// end is a no-op (railPierce t=0). Only a genuinely oblique run-off end moves, TOWARD OCCT. Assembly +
// adoption ride the same do-no-harm floor as the P2/P3 corner passes (obstacleImprovedSolid), so a
// decline or a non-certifying candidate keeps the baseline byte-identical.
//
// The gate (convexTrihedralCornerBands + allBodyFacesPlanar + endOvershootsRunoff) declines: the
// concave/mixed-sense trihedral (K6/L4 sphere, K9/M2 torus — those carry ef.flip bands), the dihedral
// miter (a 2-edge corner, not a 3-band blend), a valence other than 3, any curved-host body (F7's
// elliptical prism), and a non-overshooting run-off end. It does NOT require the three faces to be
// orthogonal — A8's wedge is non-orthogonal (the slant face), which is the whole point of this slice.
//
// SCOPE OF CHANGE (not byte-identical for the whole corpus — the survivor-arc / N5 lesson): across all
// six grids the changed-body set is exactly {A8 (RED→GREEN), A6 (GREEN→GREEN, a truncated-wedge sibling
// RE-WELDED TOWARD OCCT: rel +0.79%→−0.03%, watertight+fold-free)}, each pinned in
// occtparity/fingerprint_pins_test.go. Every other config leaves it declined → baseline byte-identical.
func adoptConvexWedgeSetback(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) (*topo.Body, bool) {
	if !allBodyFacesPlanar(body) {
		return nil, false // scope: a polyhedral wedge/box (the derivation's planar-host boundary); a
		// curved-host body (e.g. F7's elliptical prism) runs off differently and is left byte-identical
	}
	setFils, fired := setbackConvexWedgeBands(fils, blends)
	if !fired {
		return nil, false
	}
	cand := assembleFilletBody(body, setFils, blends)
	if obstacleImprovedSolid(cand) {
		return cand, true
	}
	return nil, false
}

// setbackConvexWedgeBands returns a fresh fils slice with every convex-wedge band's oblique run-off end
// re-terminated at its far-plane pierce; the caller's slice is never mutated (edgeFillet corners are
// values, so the shallow copy fully isolates the writes). fired=false leaves the caller byte-identical.
func setbackConvexWedgeBands(fils []edgeFillet, blends map[uint64]*cornerBlend) ([]edgeFillet, bool) {
	bandFI := convexWedgeBandIndices(fils, blends)
	if len(bandFI) == 0 {
		return fils, false
	}
	out := append([]edgeFillet(nil), fils...)
	fired := false
	for fi := range bandFI {
		if setbackObliqueRunoffEnds(&out[fi]) {
			fired = true
		}
	}
	return out, fired
}

// convexWedgeBandIndices collects the fils-slot indices of every band converging on a convex same-sense
// trihedral corner, so their far run-off ends can be re-terminated. A body with no such corner yields
// an empty set (the overwhelmingly common case) — the pass then declines byte-identical.
func convexWedgeBandIndices(fils []edgeFillet, blends map[uint64]*cornerBlend) map[int]bool {
	out := map[int]bool{}
	for vid, cb := range blends {
		bands, ok := convexTrihedralCornerBands(vid, cb, fils)
		if !ok {
			continue
		}
		for _, b := range bands {
			out[b.fi] = true
		}
	}
	return out
}

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
