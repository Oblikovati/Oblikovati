// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Torus∧torus miter arm pair (family D, simple/O9's second corner): two parallel-axis fillet arms of
// the SAME radius r built off the SAME shared cap plane. geometry-math-advisor consult (torus-torus-
// miter-derivation, accepted, then re-verified against O9's REAL fixture — see torusTorusHostSign's
// doc for what that re-check found). Because both tori share the exact same axis DIRECTION and hence
// the same major-circle-plane height, the implicit tube equations
//
//	(ρ_A(P) − R_A′)² + z(P)² = r²   and   (ρ_B(P) − R_B′)² + z(P)² = r²
//
// (ρ_i = perpendicular distance to torus i's axis, z = signed height above the shared major-circle
// plane, R_i′ = major radius, shared minor radius r) differ ONLY in ρ — so subtracting cancels the
// z²−r² term IDENTICALLY, a difference-of-squares factorization with NO analogue in the general
// torus∧torus SSI:
//
//	(ρ_A−R_A′)² − (ρ_B−R_B′)² = 0  ⟺  [(ρ_A−R_A′)−(ρ_B−R_B′)]·[(ρ_A−R_A′)+(ρ_B−R_B′)] = 0
//
// Both factors are candidate branches. Solving EITHER for ρ_A,ρ_B at a given z reduces to
// ρ_i = R_i′ + sign_i·D(z), D(z)=√(r²−z²) — branch 1 (physically the norm: a difference-preserving
// pairing) is signA=signB; branch 2 (a sum-preserving pairing) is signA=−signB. The FIRST derivation
// (before this was checked against O9's real fixture) assumed both arms are convex bosses and that
// branch 1 (signA=signB=+1) is always physical. O9's actual construction is `cut s1 s2` (boss ∧
// notch, see torusTorusHostSign) — a MIXED boss/bore pair whose physical branch is signA=+1,
// signB=−1 (branch 2), not signA=signB. So signA and signB are certified INDEPENDENTLY per arm
// (torusTorusHostSign) rather than assumed equal — the R−r vs R+r lesson this project keeps
// re-learning (R3's edgeFillet.armSurface bore-sign fix) applies here too. torusTorusStation already
// takes ρ_A, ρ_B independently via intersectCoplanarCircles' native (r1≠r2) two-radius solve, so no
// separate "hyperbola" code path is needed — same formula, independently-certified signs, hard-gated
// sBot (torusTorusSeamBottomCertified) as the final decline-rather-than-ship backstop.

// curvedMiterTorusPair is a miter corner whose two rolling-ball arms are BOTH exact tori: two parallel-
// axis, equal-r fillet arms sharing one cap plane. The sibling TRIHEDRAL corner where both host
// cylinders and that plane meet (fillet_twocyl_corner.go) is a separate, already-solved problem; this
// is the arm-pair SEAM between the two torus arms away from that corner. ok=false for anything else —
// buildCurvedMiterArms (families B/C) keeps its own scope untouched.
type curvedMiterTorusPair struct {
	torA, torB   geom.Torus
	edgeA, edgeB *topo.Edge
}

// curvedMiterTorusAxisTol is the dimensionless floor on â_A·â_B below which the two arm tori's axes
// are not identical enough for the coaxial-direction seam reduction (package doc: the z²−r²
// cancellation needs the SAME axis direction, not merely parallel — an anti-parallel pair mirrors the
// two major circles to different heights and the derivation does not apply). A scale-free angular
// floor, sibling of twoCylAxisParallelTol.
const curvedMiterTorusAxisTol = 1e-9

// buildCurvedMiterTorusPair recognizes a torus∧torus arm pair: builds both edges' arm surfaces via
// miterEdgeArmSurface (the SAME per-edge builder the torus+cylinder family uses) and accepts only when
// both are geom.Torus passing torusTorusCoaxial. ok=false for any other pairing — do-no-harm, never
// double-built (buildCurvedMiterArms structurally cannot also match two tori, see its own doc).
func buildCurvedMiterTorusPair(ps []filletPick, r float64, res Resolution) (curvedMiterTorusPair, bool) {
	if len(ps) != 2 {
		return curvedMiterTorusPair{}, false
	}
	sA, okA := miterEdgeArmSurface(ps[0].edge, r, res)
	sB, okB := miterEdgeArmSurface(ps[1].edge, r, res)
	if !okA || !okB {
		return curvedMiterTorusPair{}, false
	}
	torA, isA := sA.(geom.Torus)
	torB, isB := sB.(geom.Torus)
	if !isA || !isB || !torusTorusCoaxial(torA, torB, res) {
		return curvedMiterTorusPair{}, false
	}
	return curvedMiterTorusPair{torA: torA, torB: torB, edgeA: ps[0].edge, edgeB: ps[1].edge}, true
}

// torusTorusCoaxial gates the two preconditions the closed-form station formula needs: axis directions
// IDENTICAL (not just parallel), and major circles at the SAME height along that axis (coplanar) — the
// z²−r² cancellation's only real requirements. Unequal major radii do NOT need a separate gate: the
// station formula solves ρ_A, ρ_B independently (package doc), so a boss/bore mix is handled by
// certifying signA, signB per arm rather than by restricting this recognizer.
func torusTorusCoaxial(torA, torB geom.Torus, res Resolution) bool {
	if stdmath.Abs(float64(torA.AxisDir.Dot(torB.AxisDir))-1) > curvedMiterTorusAxisTol {
		return false // not identical-direction axes — anti-parallel/skew breaks the z² cancellation
	}
	axis := torA.AxisDir.AsVector()
	scale := torA.MajorRadius + torA.MinorRadius
	heightGap := stdmath.Abs(float64(torA.Center.VectorTo(torB.Center).Dot(axis)))
	return heightGap <= res.Weld()*scale
}

// torusTorusStation is the seam point at signed axial offset z from the tori's own (shared) major-
// circle-plane height: D(z)=√(r²−z²) is each tube's radial half-width at that height (real only for
// |z|≤r, the tube's own axial span — the existence gate). ρ_A=R_A′+signA·D(z), ρ_B=R_B′+signB·D(z)
// (independently signed per package doc — a mixed boss/bore pair needs signA≠signB), and the in-plane
// point at THOSE two perpendicular distances from EACH torus centre is exactly intersectCoplanarCircles'
// own two-radius circle∩circle solve (identical to the shared trihedral corner-ball's spine crossing,
// just evaluated at the offset ρ instead of the bare major radius — reused verbatim, never re-derived).
// signA=signB=0 gives the bare spine crossing (the corner-ball centre) as a degenerate case of this
// same formula. root selects whichever of the two circle∩circle roots lands nearer vp — THIS miter's
// own vertex, not the sibling trihedral corner's (which owns the other root of the same equation).
// ok=false when z is out of range or the offset circles miss.
func torusTorusStation(pair curvedMiterTorusPair, r, z, signA, signB float64, vp math.Point3, res Resolution) (math.Point3, bool) {
	disc := r*r - z*z
	if disc < 0 {
		return math.Point3{}, false // out of the tube's axial span
	}
	d := stdmath.Sqrt(disc)
	rhoA := pair.torA.MajorRadius + signA*d
	rhoB := pair.torB.MajorRadius + signB*d
	axis := pair.torA.AxisDir.AsVector()
	p1, p2, ok := intersectCoplanarCircles(pair.torA.Center, rhoA, pair.torB.Center, rhoB, axis, res)
	if !ok {
		return math.Point3{}, false
	}
	base := nearerTorusTorusRoot(vp, p1, p2)
	return base.TranslateBy(axis.Scale(math.Scalar(z))), true
}

// nearerTorusTorusRoot returns whichever of the circle∩circle roots p1, p2 lies nearer vp — the
// physical-branch tie-break every two-root circle∩circle solve in this corpus needs (mirrors
// fillet_twocyl_corner.go's own nearerAtAxialHeight, kept local since that file is a different family).
func nearerTorusTorusRoot(vp, p1, p2 math.Point3) math.Point3 {
	if p1.DistanceTo(vp) <= p2.DistanceTo(vp) {
		return p1
	}
	return p2
}

// torusTorusPhysicalSign certifies EACH arm's own D(z) sign independently — rather than assuming both
// are convex bosses (signA=signB=+1) — by comparing the two candidate sBot radii (R′±r) against that
// edge's OWN original (un-offset) host cylinder radius. ok=false when either edge's host radius does
// not certify EITHER sign — the R−r vs R+r lesson this project keeps re-learning (R3's
// edgeFillet.armSurface bore-sign fix), now caught BEFORE it can silently mismatch (unlike an earlier
// version of this function, which additionally required signA==signB and so wrongly rejected O9's own
// real fixture: O9 is `cut s1 s2`, a boss+notch mix, signA=+1 signB=−1 — see package doc).
func torusTorusPhysicalSign(pair curvedMiterTorusPair, res Resolution) (signA, signB float64, ok bool) {
	signA, okA := torusTorusHostSign(pair.edgeA, pair.torA, res)
	signB, okB := torusTorusHostSign(pair.edgeB, pair.torB, res)
	return signA, signB, okA && okB
}

// torusTorusHostSign is the sign s∈{+1,−1} that makes tor.MajorRadius+s·r equal e's ORIGINAL (un-
// filleted) host cylinder radius — the runtime certificate against a hardcoded convex-only assumption.
// ok=false when e is not a Cylinder∧Plane edge, or neither sign matches within tolerance.
func torusTorusHostSign(e *topo.Edge, tor geom.Torus, res Resolution) (float64, bool) {
	cyl, _, ok := cylinderPlaneEdge(e)
	if !ok {
		return 0, false
	}
	tol := res.Weld() * cyl.Radius
	if stdmath.Abs(tor.MajorRadius+tor.MinorRadius-cyl.Radius) <= tol {
		return 1, true
	}
	if stdmath.Abs(tor.MajorRadius-tor.MinorRadius-cyl.Radius) <= tol {
		return -1, true
	}
	return 0, false
}

// torusTorusSharedPlaneHeight is zTop, the shared cap plane's signed axial offset from torA's own
// major-circle-plane height — the sTop station's z (torusTorusStation). ok=false unless the shared
// face is a plane at (near-)exactly ±r from that height (anything else means `shared` is not actually
// the tori's own tangent cap plane, an inconsistent corner this closed form does not model).
func torusTorusSharedPlaneHeight(shared *topo.Face, torA geom.Torus, res Resolution) (float64, bool) {
	pl, ok := shared.Geometry().(geom.Plane)
	if !ok {
		return 0, false
	}
	axis := torA.AxisDir.AsVector()
	z := float64(torA.Center.VectorTo(pl.Origin).Dot(axis))
	tol := res.Weld() * (torA.MajorRadius + torA.MinorRadius)
	if stdmath.Abs(stdmath.Abs(z)-torA.MinorRadius) > tol {
		return 0, false
	}
	return z, true
}

// walkTorusTorusSeam emits k+1 seam points from sTop (z=zTop, the shared-plane tangency, where D=0
// and both circle∩circle roots coincide — hence sign-independent) to sBot (z=0, the tube's own
// equatorial touch with its original host — torusTorusSeamBottomCertified proves this below). z walks
// LINEARLY between the two (the closed-form station is exact at every z, so no adaptive refinement is
// needed); k is the SAME wedge-based chord budget the torus∩cylinder sampler uses
// (curvedSeamChordCount, arm-shape-agnostic).
func walkTorusTorusSeam(pair curvedMiterTorusPair, r, zTop, signA, signB float64, vp math.Point3, res Resolution) ([]math.Point3, bool) {
	center, okC := torusTorusStation(pair, r, 0, 0, 0, vp, res)
	sTop, okT := torusTorusStation(pair, r, zTop, signA, signB, vp, res)
	sBot, okB := torusTorusStation(pair, r, 0, signA, signB, vp, res)
	if !okC || !okT || !okB {
		return nil, false
	}
	k := curvedSeamChordCount(center, sTop, sBot)
	out := make([]math.Point3, k+1)
	out[0], out[k] = sTop, sBot
	for j := 1; j < k; j++ {
		z := zTop * (1 - float64(j)/float64(k))
		p, ok := torusTorusStation(pair, r, z, signA, signB, vp, res)
		if !ok {
			return nil, false
		}
		out[j] = p
	}
	return out, true
}

// torusTorusSeamBottomCertified is the hard gate against a wrong branch/sign selection (package doc):
// it re-derives sBot from the TRUE, un-filleted cylA∩cylB intersection (intersectCoplanarCircles on
// the ORIGINAL host radii, not the offset tori) and requires the closed-form sBot to match it within
// weld tolerance. A mismatch means the sign/branch certificate was wrong somewhere upstream — decline
// rather than ship (never launder a mismatch into a tolerance widening).
func torusTorusSeamBottomCertified(pair curvedMiterTorusPair, sBot, vp math.Point3, res Resolution) bool {
	cylA, _, okA := cylinderPlaneEdge(pair.edgeA)
	cylB, _, okB := cylinderPlaneEdge(pair.edgeB)
	if !okA || !okB {
		return false
	}
	axis := pair.torA.AxisDir.AsVector()
	p1, p2, ok := intersectCoplanarCircles(pair.torA.Center, cylA.Radius, pair.torB.Center, cylB.Radius, axis, res)
	if !ok {
		return false
	}
	truth := nearerTorusTorusRoot(vp, p1, p2)
	return float64(truth.DistanceTo(sBot)) <= res.Weld()*cylA.Radius
}

// solveCurvedMiterTorusPair builds a torus∧torus miter corner (family D): certifies each arm's own
// physical material sign independently, locates the shared cap plane's height, walks the certified
// closed-form seam, and solves the corner-ball centre (torusTorusStation's own signA=signB=0
// degenerate case — the bare spine crossing). Every step is an honest reject (do-no-harm, never a
// partial body) named with the vertex and radius.
func solveCurvedMiterTorusPair(v *topo.Vertex, shared *topo.Face, pair curvedMiterTorusPair, r float64, res Resolution) (*cornerMiter, error) {
	signA, signB, ok := torusTorusPhysicalSign(pair, res)
	if !ok {
		return nil, fmt.Errorf("fillet: torus∧torus miter arm at vertex %d does not certify against its own host cylinder radius (radius %g)", v.ID(), r)
	}
	zTop, ok := torusTorusSharedPlaneHeight(shared, pair.torA, res)
	if !ok {
		return nil, fmt.Errorf("fillet: torus∧torus miter's shared face is not the tori's own cap plane at vertex %d", v.ID())
	}
	seam, err := torusTorusCertifiedSeam(pair, r, zTop, signA, signB, v, res)
	if err != nil {
		return nil, err
	}
	center, ok := torusTorusStation(pair, r, 0, 0, 0, v.Point(), res)
	if !ok {
		return nil, fmt.Errorf("fillet: torus∧torus miter corner-ball centre undefined at vertex %d", v.ID())
	}
	curved := &curvedMiterCorner{torusPair: &pair, torEdge: pair.edgeA, cylEdge: pair.edgeB, shared: shared, center: center}
	return &cornerMiter{vertex: v, shared: shared, sBot: seam[len(seam)-1], seam: seam, curved: curved}, nil
}

// torusTorusCertifiedSeam walks the closed-form seam and hard-gates its sBot against the true
// un-filleted host crossing (torusTorusSeamBottomCertified, package doc §4: decline rather than ship
// on a mismatch) — split out of solveCurvedMiterTorusPair to stay within funlen.
func torusTorusCertifiedSeam(pair curvedMiterTorusPair, r, zTop, signA, signB float64, v *topo.Vertex, res Resolution) ([]math.Point3, error) {
	seam, ok := walkTorusTorusSeam(pair, r, zTop, signA, signB, v.Point(), res)
	if !ok {
		return nil, fmt.Errorf("fillet: torus∧torus miter seam did not close at vertex %d (radius %g)", v.ID(), r)
	}
	sBot := seam[len(seam)-1]
	if !torusTorusSeamBottomCertified(pair, sBot, v.Point(), res) {
		return nil, fmt.Errorf("fillet: torus∧torus miter seam bottom at vertex %d does not certify against the un-filleted host crossing", v.ID())
	}
	return seam, nil
}
