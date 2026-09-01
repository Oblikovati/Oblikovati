// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
)

// O1-class mixed-sense curved-host corner: the third member of the mixed trihedral family, after M8's
// analytic 2r-torus (fillet_curved_mixed_weld.go) and N4's canal (fillet_curved_mixed_bspline.go). A
// trihedral vertex where TWO CONCAVE arms — a cylinder arm on the boss wall ∧ a flat wall, and a cove torus
// arm (major = R+r) on the boss wall ∧ a flat cap — meet ONE CONVEX planar band (Plane∧Plane, its
// rolling-ball cylinder). Roll-sense regime R2 for both concave arms (corner-rollsense-remap.md §3:
// s = −MAT_in·CVX = +1 on a boss), which is why the cove torus's major radius is R+r and why the corner ball
// rides the boss wall at ρ = R+r rather than R−r.
//
// The role signature is DISJOINT from both prior classes and each other's, which is what lets all three
// share the dispatch ladder with no order dependence (asserted by the class-disjointness tests):
//
//	M8: convex Cyl∧Plane  + cove torus     + planar CONCAVE band
//	N4: concave cyl arm   + CONVEX torus   + planar CONCAVE band
//	O1: concave cyl arm   + cove torus     + planar CONVEX  band
//
// Topologically O1 is N4's structure with the roles PERMUTED: the two concave arms TERMINATE at the corner
// on radius-r cross-section arcs, the convex band runs PAST it (so the patch rails on the band's own
// cylinder), and the patch's fourth side rails on the shared MID host — which here is the boss CYLINDER, not
// a plane. DRAWEXE 8.0.0 on the O1 fixture confirms exactly that ring: the corner patch result_5 borders
// result_3 (the cylinder arm, at z=85), result_12 (the convex band), result_6 (the cove torus band) and
// result_1 (the boss wall, which carries the corner's own inner loop).

// o1MixedArms are the three role-classified arms of an O1-class corner.
type o1MixedArms struct {
	ccyl    edgeFillet // CONCAVE cylinder arm; hosts = mid host cylinder + the shared flat wall
	cove    edgeFillet // CONCAVE cove torus arm (major R+r); hosts = mid host cylinder + the shared flat cap
	lateral edgeFillet // CONVEX planar Plane∧Plane band; hosts = both shared flats; runs LATERAL
}

// o1Corner is the fully solved O1 corner: the four points + two terminating-arm arcs, the two on-host rails
// read off the canal's own boundary isoparms (railBC on the lateral band, railDA on the mid cylinder), the
// certified canal patch, and the mid host face the patch's D→A rail rides.
type o1Corner struct {
	pts     cornerCanalPts
	railBC  geom.Curve3 // on-lateral-band contact rail, oriented B→C
	railDA  geom.Curve3 // on-mid-cylinder contact rail, oriented D→A
	patch   CornerBlendPatch
	midFace *topo.Face
}

// isConvexBandArm reports the O1 convex planar band: a CONVEX Plane∧Plane fillet, whose arm surface is the
// rolling-ball cylinder cornerArms normalized onto it. Distinct from isPlanarBandArm (the CONCAVE band M8
// and N4 carry, ef.flip) and from isConvexCylArm (a convex Cyl∧Plane pivot, whose hosts are not both planes)
// — those three predicates partition the convex/concave × planar/curved cases the three classes need.
func isConvexBandArm(ef edgeFillet) bool {
	if !isConvexCylArm(ef) {
		return false
	}
	_, aPlane := ef.a.Geometry().(geom.Plane)
	_, bPlane := ef.b.Geometry().(geom.Plane)
	return aPlane && bPlane
}

// classifyO1MixedArms partitions the three trihedral arms into the concave-cyl, cove-torus, and convex-band
// roles — or ok=false when the corner is not this 1+1+1 O1 config (any other valence/sense keeps its prior
// path). Requires exactly 3 arms and one arm per role (dup-role guard), mirroring classifyN4MixedArms.
func classifyO1MixedArms(arms []edgeFillet) (o1MixedArms, bool) {
	if len(arms) != 3 {
		return o1MixedArms{}, false
	}
	var out o1MixedArms
	var seen [3]bool // [ccyl, cove, lateral]
	for _, ef := range arms {
		if !assignO1Role(&out, &seen, ef) {
			return o1MixedArms{}, false
		}
	}
	if !seen[0] || !seen[1] || !seen[2] {
		return o1MixedArms{}, false
	}
	return out, true
}

// assignO1Role files one arm into its O1 role slot, rejecting an unrecognised role or a dup-role.
func assignO1Role(out *o1MixedArms, seen *[3]bool, ef edgeFillet) bool {
	switch {
	case isConcaveCylArm(ef):
		if seen[0] {
			return false
		}
		out.ccyl, seen[0] = ef, true
	case isCoveTorusArm(ef):
		if seen[1] {
			return false
		}
		out.cove, seen[1] = ef, true
	case isConvexBandArm(ef):
		if seen[2] {
			return false
		}
		out.lateral, seen[2] = ef, true
	default:
		return false
	}
	return true
}

// o1CornerHosts are the three host faces an O1 corner needs: the MID cylinder both concave arms roll on, and
// the two shared flats — one shared by the cylinder arm and the band, one by the cove arm and the band. The
// two flats are not used to build the patch (the frame's û falls out of the two axes), but their EXISTENCE is
// what makes the frame's tangency argument true, so they are required rather than assumed.
type o1CornerHosts struct {
	mid     *topo.Face
	midCyl  geom.Cylinder
	wall    *topo.Face // shared by ccyl + lateral
	capFlat *topo.Face // shared by cove + lateral
}

// o1Hosts resolves the three host faces by face identity. ok=false when any is missing.
func o1Hosts(arms o1MixedArms) (o1CornerHosts, bool) {
	mid, midCyl, okM := sharedCylinderHostFace(arms.ccyl, arms.cove)
	wall, okW := sharedPlaneHost(arms.ccyl, arms.lateral)
	capFlat, okC := sharedPlaneHost(arms.cove, arms.lateral)
	if !okM || !okW || !okC {
		return o1CornerHosts{}, false
	}
	return o1CornerHosts{mid: mid, midCyl: midCyl, wall: wall, capFlat: capFlat}, true
}

// o1ArmSurfaces are the three exact analytic arm surfaces, unwrapped once.
type o1ArmSurfaces struct {
	spine geom.Cylinder // the concave cylinder arm's rolling-ball cylinder
	cove  geom.Torus    // the cove torus arm
	band  geom.Cylinder // the convex band's rolling-ball cylinder — the LATERAL tube
}

// o1Surfaces unwraps the three arm surfaces and gates them on the requested radius r: all three tubes must be
// the SAME rolling ball, because the corner patch is the envelope of ONE ball of radius r. A mixed-radius
// corner gets an explicit decline here rather than an opaque loft failure further down.
func o1Surfaces(arms o1MixedArms, r, tol float64) (o1ArmSurfaces, bool) {
	spine, ok1 := arms.ccyl.armSurface.(geom.Cylinder)
	cove, ok2 := arms.cove.armSurface.(geom.Torus)
	band, ok3 := arms.lateral.armSurface.(geom.Cylinder)
	if !ok1 || !ok2 || !ok3 {
		return o1ArmSurfaces{}, false
	}
	if stdmath.Abs(spine.Radius-r) > tol || stdmath.Abs(cove.MinorRadius-r) > tol || stdmath.Abs(band.Radius-r) > tol {
		return o1ArmSurfaces{}, false
	}
	return o1ArmSurfaces{spine: spine, cove: cove, band: band}, true
}

// o1BallRideRadius is ρ, the corner ball's distance from the MID host's axis, read off the CONCAVE CYLINDER
// ARM's own spine rather than assumed as R+r: the arm surface already encodes the roll-sense the case's
// build (boss vs bore, convex vs concave) produced, so deriving ρ from it means the O1 path cannot impose
// N4's or M8's sense on a case that has a different one. The two independent consistency checks are that ρ is
// exactly r off the host radius (the ball is tangent to that host) and that the cove arm's spine circle lies
// on the same ρ-cylinder — the second is what ties the two terminating arms to ONE ball.
func o1BallRideRadius(hosts o1CornerHosts, surf o1ArmSurfaces, r, tol float64) (float64, bool) {
	axis := probe.Unit(hosts.midCyl.AxisDir.AsVector())
	rel := hosts.midCyl.Origin.VectorTo(surf.spine.Origin)
	rho := float64(rel.Sub(axis.Scale(rel.Dot(axis))).Length())
	if stdmath.Abs(stdmath.Abs(rho-hosts.midCyl.Radius)-r) > tol {
		return 0, false
	}
	if stdmath.Abs(surf.cove.MajorRadius-rho) > tol {
		return 0, false
	}
	return rho, true
}

// solveO1Corner derives the full O1 corner from the classified arms, or ok=false at any class check — the
// do-no-harm floor (the corner keeps its prior declined path, and nothing is mutated).
func solveO1Corner(arms o1MixedArms, r float64, res Resolution) (o1Corner, bool) {
	tol := res.Weld() * r
	hosts, okH := o1Hosts(arms)
	if !okH {
		return o1Corner{}, false
	}
	surf, okS := o1Surfaces(arms, r, tol)
	if !okS {
		return o1Corner{}, false
	}
	rho, okR := o1BallRideRadius(hosts, surf, r, tol)
	if !okR {
		return o1Corner{}, false
	}
	frame, ok := o1SolvedFrame(hosts, surf, rho, r, tol)
	if !ok {
		return o1Corner{}, false
	}
	return assembleO1Corner(frame, arms, hosts, surf, r, res)
}

// o1SolvedFrame builds the ball-centre frame, pins its quarter-turn span, and self-checks both end stations
// against the two terminating arms' own spines.
func o1SolvedFrame(hosts o1CornerHosts, surf o1ArmSurfaces, rho, r, tol float64) (o1BallFrame, bool) {
	frame, ok := newO1BallFrame(hosts.midCyl.Origin, hosts.midCyl.AxisDir.AsVector(), surf.band, rho, 2*r)
	if !ok {
		return o1BallFrame{}, false
	}
	if !frame.o1CanalSpan(surf.spine.Origin, surf.cove.Center, tol) {
		return o1BallFrame{}, false
	}
	if !frame.holdsStations(surf.spine, surf.cove, hosts.midCyl.Origin, tol) {
		return o1BallFrame{}, false
	}
	return frame, true
}

// o1CornerPoints solves the four corner points and the two terminating-arm cross-section arcs from the frame's
// two end stations: each station's ball touches the MID host at one point and the LATERAL band at the other,
// and its radius-r cross-section arc between them is the curve the terminating arm's face trims to.
func o1CornerPoints(frame o1BallFrame, hosts o1CornerHosts, surf o1ArmSurfaces, r float64) (cornerCanalPts, bool) {
	m0, ok0 := frame.centerAt(frame.a0)
	m1, ok1 := frame.centerAt(frame.a1)
	if !ok0 || !ok1 {
		return cornerCanalPts{}, false
	}
	_, _, a := geom.ClosestPointOnSurface(hosts.midCyl, m0)
	_, _, b := geom.ClosestPointOnSurface(surf.band, m0)
	_, _, c := geom.ClosestPointOnSurface(surf.band, m1)
	_, _, d := geom.ClosestPointOnSurface(hosts.midCyl, m1)
	arcAB, okA := arcThrough(m0, r, a, b)
	arcCD, okC := arcThrough(m1, r, c, d)
	if !okA || !okC {
		return cornerCanalPts{}, false
	}
	return cornerCanalPts{a: a, b: b, c: c, d: d, arcAB: arcAB, arcCD: arcCD, ballAB: m0, ballCD: m1}, true
}

// assembleO1Corner lofts the rolling-ball path into the canal patch, reads its two on-host rails off the
// canal's own boundary isoparms, and certifies it against the 4-cycle A→B→C→D→A through the SHARED
// cornerCanalPatch — the same certificate N4 goes through, with the ring's four adjacent surfaces permuted to
// O1's roles. ok=false when the loft, the isoparms or the certificate fail.
func assembleO1Corner(frame o1BallFrame, arms o1MixedArms, hosts o1CornerHosts, surf o1ArmSurfaces, r float64, res Resolution) (o1Corner, bool) {
	pts, ok := o1CornerPoints(frame, hosts, surf, r)
	if !ok {
		return o1Corner{}, false
	}
	path, ok := o1CornerBallPath(frame, hosts.midCyl, surf.band)
	if !ok {
		return o1Corner{}, false
	}
	canal, ok := cornerCanalSurface(path, pts, r, res.Weld())
	if !ok {
		return o1Corner{}, false
	}
	railBC, railDA, ok := cornerCanalRails(canal)
	if !ok {
		return o1Corner{}, false
	}
	ring := cornerCanalRing{
		armAB: arms.ccyl.armSurface, lateral: surf.band,
		armCD: arms.cove.armSurface, mid: hosts.midCyl,
	}
	patch, ok := cornerCanalPatch(canal, pts, railBC, railDA, ring, res)
	if !ok {
		return o1Corner{}, false
	}
	return o1Corner{pts: pts, railBC: railBC, railDA: railDA, patch: patch, midFace: hosts.mid}, true
}
