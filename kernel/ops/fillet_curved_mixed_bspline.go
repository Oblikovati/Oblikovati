// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// N4-class mixed-sense curved-host corner: the BSpline (coons4) sibling of the M8 2r-torus corner
// (fillet_curved_mixed_weld.go). A trihedral vertex where a CONCAVE Cylinder∧{Plane,boss-wall} arm + a
// CONVEX Torus arm (the boss cap-rim) + a planar Plane∧Plane band meet. Unlike M8 (whose three arms all
// TERMINATE at the corner via radius-r arcs bounding a 2r-torus patch), N4's torus arm runs LATERALLY
// past the corner: only the concave-cyl and planar-band arms terminate (radius-r arcs), while the corner
// patch's other two sides are on-host contact rails — one on the shared vertical plane, ONE ON THE TORUS
// ARM ITSELF. OCCT fills this 4-sided loop with a rational BSplineSurface, and a pole dump proves that
// surface is the exact rolling-ball CANAL over a ball-centre curve through the corner; we build it as
// such (fillet_curved_mixed_canal.go), which also DERIVES the two on-host rails as the ball's contact
// loci instead of guessing them. The four corner points are derived from OUR arm surfaces
// (DRAWEXE-validated on N4 to ≤0.02): A(113.39,67.19,55) B(118.31,66.32,50) C(116.38,61.12,45)
// D(111.59,56.99,45):
//
//	C = concave-cyl boss-wall contact ruling ∩ torus outer-equator (v=0) contact circle;
//	B = planar-band top-plane contact ruling ∩ torus top (v=π/2) contact circle;
//	D,A = the concave-cyl / planar-band feet on the shared vertical plane at those stations.
//
// The 4-cycle A→B→C→D→A: A→B (band arc), B→C (rail on torus), C→D (ccyl arc), D→A (rail on vplane).

// n4MixedArms are the three role-classified arms of an N4-class corner. The roles are DISJOINT from M8's
// (convex-cyl + concave-cove-torus + planar), so classifyN4MixedArms and classifyMixedRoleArms never both
// accept the same corner — the two weld paths are mutually exclusive (do-no-harm).
type n4MixedArms struct {
	ccyl  edgeFillet // CONCAVE cylinder arm; hosts = shared vertical plane + boss wall
	torus edgeFillet // CONVEX torus arm (boss cap-rim); hosts = shared top plane + boss wall; runs LATERAL
	band  edgeFillet // planar Plane∧Plane band; hosts = shared vertical plane + top plane
}

// n4CornerPts are the four corner points, the two terminating-arm cross-section arcs (band A→B, ccyl
// C→D), and the two arms' rolling-ball centres at those stations (the canal centre curve's endpoints).
// Built ONCE so every welded neighbour reads the byte-identical curve.
type n4CornerPts struct {
	a, b, c, d         math.Point3
	arcAB, arcCD       geom.Arc3d
	ballBand, ballCcyl math.Point3
}

// n4Corner is the fully solved N4 corner: the points, the two arcs, the two on-host rails (torus B→C,
// vplane D→A) read off the canal's own boundary isoparms, and the certified canal patch.
type n4Corner struct {
	pts    n4CornerPts
	railBC geom.Curve3 // on-torus contact rail (the ball's torus-arm contact locus), oriented B→C
	railDA geom.Curve3 // on-vplane contact rail (the ball's vplane contact locus), oriented D→A
	patch  CornerBlendPatch
	vplane geom.Plane
	boss   geom.Cylinder
}

// isConcaveCylArm reports the N4 concave cylinder arm: an exact geom.Cylinder arm rolling in a reentrant
// void (armConcave). Distinct from M8's isConvexCylArm (a convex, non-flipped cylinder pivot).
func isConcaveCylArm(ef edgeFillet) bool {
	_, ok := ef.armSurface.(geom.Cylinder)
	return ok && ef.armConcave
}

// isConvexTorusArm reports the N4 convex torus arm (the boss cap-rim fillet). Distinct from M8's
// isCoveTorusArm (a CONCAVE cove torus).
func isConvexTorusArm(ef edgeFillet) bool {
	_, ok := ef.armSurface.(geom.Torus)
	return ok && !ef.armConcave
}

// classifyN4MixedArms partitions the three trihedral arms into the concave-cyl, convex-torus, and planar
// roles — or ok=false when the corner is not this 1+1+1 N4 config (any other valence/sense keeps its prior
// path). Requires exactly 3 arms and one arm per role (dup-role guard), mirroring classifyMixedRoleArms.
func classifyN4MixedArms(arms []edgeFillet) (n4MixedArms, bool) {
	if len(arms) != 3 {
		return n4MixedArms{}, false
	}
	var out n4MixedArms
	var seen [3]bool // [ccyl, torus, band]
	for _, ef := range arms {
		if !assignN4Role(&out, &seen, ef) {
			return n4MixedArms{}, false
		}
	}
	if !seen[0] || !seen[1] || !seen[2] {
		return n4MixedArms{}, false
	}
	return out, true
}

// assignN4Role files one arm into its N4 role slot, rejecting an unrecognised role or a dup-role.
func assignN4Role(out *n4MixedArms, seen *[3]bool, ef edgeFillet) bool {
	switch {
	case isConcaveCylArm(ef):
		if seen[0] {
			return false
		}
		out.ccyl, seen[0] = ef, true
	case isConvexTorusArm(ef):
		if seen[1] {
			return false
		}
		out.torus, seen[1] = ef, true
	case isPlanarBandArm(ef):
		if seen[2] {
			return false
		}
		out.band, seen[2] = ef, true
	default:
		return false
	}
	return true
}

// sharedCylinderHostFace returns the cylinder host surface both arms share by face identity — the boss
// wall for the ccyl+torus pair. ok=false when they share no cylinder host.
func sharedCylinderHostFace(x, y edgeFillet) (geom.Cylinder, bool) {
	for _, fx := range [2]*topo.Face{x.a, x.b} {
		cyl, ok := fx.Geometry().(geom.Cylinder)
		if !ok {
			continue
		}
		if fx == y.a || fx == y.b {
			return cyl, true
		}
	}
	return geom.Cylinder{}, false
}

// solveN4Corner derives the full N4 corner (points, arcs, contact-locus rails, canal patch) from the
// classified arms, or ok=false when a host is missing, a station has no real intersection, or the canal
// patch fails to build/certify — the do-no-harm floor (the corner keeps its prior declined path).
func solveN4Corner(arms n4MixedArms, r float64, res Resolution) (n4Corner, bool) {
	boss, okH := sharedCylinderHostFace(arms.ccyl, arms.torus)
	vFace, okV := sharedPlaneHost(arms.ccyl, arms.band)
	tFace, okT := sharedPlaneHost(arms.torus, arms.band)
	if !okH || !okV || !okT {
		return n4Corner{}, false
	}
	torus, ok1 := arms.torus.armSurface.(geom.Torus)
	ccylBall, ok2 := arms.ccyl.armSurface.(geom.Cylinder)
	bandBall, ok3 := arms.band.armSurface.(geom.Cylinder)
	if !ok1 || !ok2 || !ok3 {
		return n4Corner{}, false
	}
	vplane := vFace.Geometry().(geom.Plane)
	tplane := tFace.Geometry().(geom.Plane)
	pts, ok := n4CornerPoints(boss, torus, ccylBall, bandBall, vplane, tplane, r)
	if !ok {
		return n4Corner{}, false
	}
	return assembleN4Corner(pts, arms, vplane, boss, res)
}

// n4CornerPoints solves the four corner points and the two terminating-arm cross-section arcs from the
// arm/host geometry (the file comment's construction). ok=false on any degeneracy.
func n4CornerPoints(boss geom.Cylinder, torus geom.Torus, ccylBall, bandBall geom.Cylinder, vplane, tplane geom.Plane, r float64) (n4CornerPts, bool) {
	c, ballCcyl, ok := n4PointCAndStation(boss, torus, ccylBall)
	if !ok {
		return n4CornerPts{}, false
	}
	_, _, d := geom.ClosestPointOnSurface(vplane, ballCcyl)
	arcCD, ok := arcThrough(ballCcyl, r, c, d)
	if !ok {
		return n4CornerPts{}, false
	}
	b, ballBand, ok := n4PointBAndStation(bandBall, tplane, torus, c)
	if !ok {
		return n4CornerPts{}, false
	}
	_, _, a := geom.ClosestPointOnSurface(vplane, ballBand)
	arcAB, ok := arcThrough(ballBand, r, a, b)
	if !ok {
		return n4CornerPts{}, false
	}
	return n4CornerPts{
		a: a, b: b, c: c, d: d, arcAB: arcAB, arcCD: arcCD,
		ballBand: ballBand, ballCcyl: ballCcyl,
	}, true
}

// arcThrough is the terminating arm's cross-section arc from `from` to `to`: the radius-r circle centred
// at the arm ball centre, through the on-circle midpoint between the two feet (never the chord).
func arcThrough(center math.Point3, r float64, from, to math.Point3) (geom.Arc3d, bool) {
	mid := arcMidBetween(center, r, from, to)
	arc, err := geom.Arc3dByThreePoints(from, mid, to)
	return arc, err == nil
}

// n4PointCAndStation is corner point C — where the concave-cyl arm's boss-wall contact ruling meets the
// torus outer-equator contact circle (both on the boss wall) — and the ccyl ball centre at that station
// (z = the torus centre height). C = boss axis (at the torus height) + R_boss·(radial toward the ccyl ball).
func n4PointCAndStation(boss geom.Cylinder, torus geom.Torus, ccylBall geom.Cylinder) (math.Point3, math.Point3, bool) {
	axis := boss.AxisDir.AsVector()
	obAtTorus := footOnLine(torus.Center, boss.Origin, axis)
	foot := footOnLine(ccylBall.Origin, boss.Origin, axis)
	dir, err := math.UnitVector3FromVector(foot.VectorTo(ccylBall.Origin))
	if err != nil {
		return math.Point3{}, math.Point3{}, false
	}
	c := obAtTorus.TranslateBy(dir.AsVector().Scale(math.Scalar(boss.Radius)))
	ballCcyl := footOnLine(torus.Center, ccylBall.Origin, ccylBall.AxisDir.AsVector()) // ccyl ball at the torus height
	return c, ballCcyl, true
}

// n4PointBAndStation is corner point B — where the planar-band arm's top-plane contact ruling meets the
// torus top (v=π/2) contact circle of radius Major — and the band ball centre at that station. It solves
// the quadratic for the band spine parameter whose top-plane contact lands on the torus top circle, picking
// the root nearest `near` (corner point C). ok=false when the ruling misses the circle.
func n4PointBAndStation(bandBall geom.Cylinder, tplane geom.Plane, torus geom.Torus, near math.Point3) (math.Point3, math.Point3, bool) {
	n := unit(tplane.Normal())
	d := bandBall.AxisDir.AsVector()
	_, _, f0 := geom.ClosestPointOnSurface(tplane, bandBall.Origin) // top-plane contact at t=0
	fd := d.Sub(n.Scale(d.Dot(n)))                                  // contact-line direction
	axis := torus.AxisDir.AsVector()
	a0 := torus.Center.VectorTo(f0)
	qa := f64(fd.Dot(fd)) - sq(f64(fd.Dot(axis)))
	qb := 2 * (f64(a0.Dot(fd)) - f64(a0.Dot(axis))*f64(fd.Dot(axis)))
	qc := f64(a0.Dot(a0)) - sq(f64(a0.Dot(axis))) - sq(torus.MajorRadius)
	t, ok := nearestQuadraticRoot(qa, qb, qc, f0, fd, near)
	if !ok {
		return math.Point3{}, math.Point3{}, false
	}
	b := f0.TranslateBy(fd.Scale(math.Scalar(t)))
	ballBand := bandBall.Origin.TranslateBy(d.Scale(math.Scalar(t)))
	return b, ballBand, true
}

// nearestQuadraticRoot solves qa·t²+qb·t+qc=0 and returns the root whose contact point f0+t·fd is nearest
// `near`. ok=false when there is no real root.
func nearestQuadraticRoot(qa, qb, qc float64, f0 math.Point3, fd math.Vector3, near math.Point3) (float64, bool) {
	t0, t1, ok := solveQuadratic(qa, qb, qc)
	if !ok {
		return 0, false
	}
	p0 := f0.TranslateBy(fd.Scale(math.Scalar(t0)))
	p1 := f0.TranslateBy(fd.Scale(math.Scalar(t1)))
	if p0.DistanceTo(near) <= p1.DistanceTo(near) {
		return t0, true
	}
	return t1, true
}

// solveQuadratic returns the two real roots of a·t²+b·t+c (equal when linear/double), or ok=false when the
// discriminant is negative or the equation is degenerate.
func solveQuadratic(a, b, c float64) (float64, float64, bool) {
	if stdmath.Abs(a) < 1e-12 {
		if stdmath.Abs(b) < 1e-12 {
			return 0, 0, false
		}
		r := -c / b
		return r, r, true
	}
	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, 0, false
	}
	s := stdmath.Sqrt(disc)
	return (-b + s) / (2 * a), (-b - s) / (2 * a), true
}

// assembleN4Corner builds the rolling-ball canal patch and reads its two on-host rails off the canal's own
// boundary isoparms, once the four points + two arcs are solved. ok=false when the ball path does not hold
// (the corner is not really rolling on the lateral torus arm), when the loft or the isoparms fail, or when
// the patch does not certify (do-no-harm floor — the corner keeps its prior declined path).
func assembleN4Corner(pts n4CornerPts, arms n4MixedArms, vplane geom.Plane, boss geom.Cylinder, res Resolution) (n4Corner, bool) {
	torus := arms.torus.armSurface.(geom.Torus)
	r := torus.MinorRadius
	tol := res.Weld() * r
	path, ok := n4CornerBallPath(torus, vplane, pts.ballBand, pts.ballCcyl, tol)
	if !ok {
		return n4Corner{}, false
	}
	surf, ok := n4CanalSurface(path, pts, r, res.Weld())
	if !ok {
		return n4Corner{}, false
	}
	railBC, railDA, ok := n4CanalRails(surf)
	if !ok {
		return n4Corner{}, false
	}
	patch, ok := n4CornerPatch(surf, pts, railBC, railDA, arms, vplane, res)
	if !ok {
		return n4Corner{}, false
	}
	return n4Corner{pts: pts, railBC: railBC, railDA: railDA, patch: patch, vplane: vplane, boss: boss}, true
}

// n4CornerPatch certifies the canal surface as the corner patch against the 4-cycle A→B→C→D→A it must
// weld into: the two arm arcs (G1 to their arm surfaces) and the two contact-locus rails (G1 to the torus
// arm / the vertical plane). It emits the canal's OWN boundary isoparms as the patch loops
// (canalPatchLoops, shared with canalProvider) so nothing off-surface can reach assembly, and rejects a
// patch whose loops, folding, closure or G1 residual fail — the same certificate contract coons4's build
// satisfied, just measured on a surface that is the true envelope rather than a transfinite fill.
func n4CornerPatch(surf geom.BSplineSurface, pts n4CornerPts, railBC, railDA geom.Curve3, arms n4MixedArms, vplane geom.Plane, res Resolution) (CornerBlendPatch, bool) {
	loop := RailLoop{Sides: []Side{
		{Curve: pts.arcAB, Adjacent: arms.band.armSurface, Cont: G1}, // s0: A→B
		{Curve: railBC, Adjacent: arms.torus.armSurface, Cont: G1},   // s1: B→C
		{Curve: pts.arcCD, Adjacent: arms.ccyl.armSurface, Cont: G1}, // s2: C→D
		{Curve: railDA, Adjacent: vplane, Cont: G1},                  // s3: D→A
	}}
	loops, err := canalPatchLoops(surf)
	if err != nil {
		return CornerBlendPatch{}, false
	}
	cert := certifyN4CanalPatch(surf, loop, maxLoopSurfaceDev(surf, loops), res)
	if !cert.Valid(res) {
		return CornerBlendPatch{}, false
	}
	return CornerBlendPatch{Surface: surf, Loops: loops, Kind: BlendKindCanal}, true
}

// certifyN4CanalPatch proves the corner canal (ADR-3): Closed from the received 4-cycle, WeldsArms
// structural, NoFold via the shared column sweep, MaxDev the measured loop-to-surface G0 residual, and
// MaxAngleDev the worst G1 crease against all four neighbours (canalSideCrease, shared with canalProvider).
// Unlike canalStationProvider's CORE panel — which has no analytic Adjacent and therefore reports
// MaxAngleDev 0 — every N4 side DOES have one, so the tangency the canal construction guarantees is
// measured rather than asserted.
func certifyN4CanalPatch(surf geom.BSplineSurface, loop RailLoop, dev float64, res Resolution) Certificate {
	crease := 0.0
	for _, s := range loop.Sides {
		crease = stdmath.Max(crease, canalSideCrease(surf, s))
	}
	return Certificate{
		Closed:      loop.Closed(res.Weld()),
		WeldsArms:   true,
		NoFold:      obstacleNoFold(surf, res),
		MaxDev:      dev,
		MaxAngleDev: crease,
	}
}

// f64 narrows a math.Scalar to float64 for the quadratic-solve arithmetic.
func f64(s math.Scalar) float64 { return float64(s) }

// sq is x².
func sq(x float64) float64 { return x * x }
