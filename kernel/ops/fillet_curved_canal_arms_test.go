// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// W2 per-arm-centre canal arm faces. Every assertion drives the REAL N7 fixture (n7CornerFill →
// extractCurvedCorner → resolveBlend), never a fabricated patch. The load-bearing gate is SHARED-CURVE
// IDENTITY: each arm face's corner-side rail is point-identical to the canal patch's corresponding
// boundary (the watertight self-check). The crux is that the torus arm (s_5) — which DECLINED under the
// single ball (the fixed 2r wall-tangent gap) — now BUILDS at its own reflected centre C″ (z=5).

// n7CanalWeldInputs resolves the real N7 corner into the inputs canalWeldFaces threads (single source):
// the canal patch, its four tagged boundary isocurves, the per-arm reflected centres, and the scale.
func n7CanalWeldInputs(t *testing.T, w cornerWeld, arms []edgeFillet, res Resolution) (CornerBlendPatch, canalBoundaries, []math.Point3, float64) {
	t.Helper()
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Canal == nil {
		t.Fatalf("N7 must extract a Canal-marked loop; ok=%v canal=%v", ok, loop.Canal != nil)
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCanal {
		t.Fatalf("N7 must resolve to the canal tier; ok=%v kind=%q", ok, patch.Kind)
	}
	boundaries, err := canalBoundaryRoles(patch)
	if err != nil {
		t.Fatalf("canalBoundaryRoles declined: %v", err)
	}
	scale := tangentCornerScale(w, arms)
	centres, ok, _ := reflectedArmCentres(w, arms, scale, res)
	if !ok {
		t.Fatalf("reflectedArmCentres unresolved for N7")
	}
	return patch, boundaries, centres, scale
}

// n7MidArmIndex returns the N7 mid (non-wall) arm index (s_10) via the same classification the assembler
// uses, failing the test if the wall/arm topology is not the expected one-cylinder-two-plane mix.
func n7MidArmIndex(t *testing.T, arms []edgeFillet) int {
	t.Helper()
	wallFace, _, ok := tangentCornerWall(arms)
	if !ok {
		t.Fatalf("N7 corner has no single cylinder wall host")
	}
	wa, ok := wallSharingArms(arms, wallFace)
	if !ok {
		t.Fatalf("N7 corner does not have exactly two wall-sharing arms")
	}
	return nonWallArmIndex(arms, wa[0], wa[1])
}

// TestCanalArmFaces_BuildThreeWatertightArms is the W2 core gate: canalArmFaces builds all three N7 arm
// faces (including the torus arm), each with a closed loop whose corner-side rail is POINT-IDENTICAL to
// the canal patch's shared boundary (watertightness), whose two host rails lie on their hosts, and whose
// far arc is a radius-r terminal cross-section on the arm surface.
func TestCanalArmFaces_BuildThreeWatertightArms(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	patch, boundaries, centres, scale := n7CanalWeldInputs(t, w, arms, res)
	faces, reason := canalArmFaces(arms, w, boundaries, centres, scale, res)
	if reason != "" {
		t.Fatalf("canalArmFaces declined the real N7 corner: %s", reason)
	}
	if len(faces) != len(arms) {
		t.Fatalf("want %d arm faces, got %d", len(arms), len(faces))
	}
	mid := n7MidArmIndex(t, arms)
	tol := res.Weld() * scale
	for i := range arms {
		rail, rev, ok := canalArmCornerRail(boundaries, centres[i], i, mid)
		if !ok {
			t.Fatalf("arm %d: no corner-side rail", i)
		}
		assertArmFaceCloses(t, faces[i], rail, rev, tol, i)
		assertCornerRailShared(t, faces[i], rail, rev, patch, i)
		assertArmHostRailsOnHost(t, arms[i], centres[i], w, scale, res, tol, i)
		assertArmFarArcIsCrossSection(t, arms[i], centres[i], w, scale, res, tol, i)
	}
}

// TestCanalArmFace_TorusArmBuildsAtOwnCentre is the crux: the s_5 torus arm — which declined under the
// single ball because the shared corner demands its wall-tangent at z=15 where it physically touches at
// z=5 (a fixed 2r=10 gap) — BUILDS as a valid arm face when solved at its OWN reflected centre C″ (z=5).
func TestCanalArmFace_TorusArmBuildsAtOwnCentre(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	_, boundaries, centres, scale := n7CanalWeldInputs(t, w, arms, res)
	i := torusArmIndex(t, arms)
	if got := centres[i].Z; stdmath.Abs(float64(got)-5) > 1e-9 {
		t.Fatalf("torus arm reflected centre z = %v, want the on-spine-plane z=5 (C″)", got)
	}
	rail, rev, ok := canalArmCornerRail(boundaries, centres[i], i, n7MidArmIndex(t, arms))
	if !ok {
		t.Fatalf("torus arm: no corner-side rail")
	}
	if _, ok := arms[i].armSurface.(geom.Torus); !ok {
		t.Fatalf("arm %d is %T, want geom.Torus (s_5)", i, arms[i].armSurface)
	}
	if _, ok := canalArmFace(arms[i], centres[i], rail, rev, w, scale, res); !ok {
		t.Fatal("the torus arm must BUILD at its own reflected centre C″ (z=5) — the whole reason the canal path exists")
	}
}

// TestCanalArmFaces_WrongEndArcMappingRejects is the mapping mutation test: each wall-sharing arm must
// take the END ARC at its OWN reflected centre (matched by proximity). Feeding a wall arm the OTHER
// wall arm's end arc (the index-swapped mis-map) makes the corner rail's endpoints disagree with the
// arm's host-tangent points, so the loop cannot close and canalArmFace honest-rejects — proof the
// by-proximity mapping is load-bearing, not incidental.
func TestCanalArmFaces_WrongEndArcMappingRejects(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	_, boundaries, centres, scale := n7CanalWeldInputs(t, w, arms, res)
	mid := n7MidArmIndex(t, arms)
	for i := range arms {
		if i == mid {
			continue // the mid arm takes the foot-locus, not an end arc — not part of the swap
		}
		right, rightRev, _ := endArcForCentre(boundaries, centres[i])
		if _, ok := canalArmFace(arms[i], centres[i], right, rightRev, w, scale, res); !ok {
			t.Fatalf("wall arm %d must BUILD with its own (proximity-matched) end arc", i)
		}
		wrong, wrongRev := otherEndArc(boundaries, centres[i])
		if _, ok := canalArmFace(arms[i], centres[i], wrong, wrongRev, w, scale, res); ok {
			t.Fatalf("wall arm %d must REJECT the swapped (other arm's) end arc — the loop cannot close", i)
		}
	}
}

// TestCanalArmLoop_EarlierJunctionGapRejected is the mutation-evidence test for the W2 review's
// closure-gate finding: canalArmLoop's own gate used to check only the FINAL junction (hn.to vs
// corner[0]) — near-definitionally satisfied because orderArmHostRails selects hNear BY matching .to
// against corner[0]. A break at an EARLIER junction (here: the far host rail's target, hFar.to → far.from,
// the same class of break as the W2 reviewer's Mutation C — orientEndSeg reversed into a 75-unit gap)
// used to sail through canalArmLoop unnoticed and only surface downstream (assertArmFaceCloses /
// assembleBody). After the fix, canalArmLoop must decline the break ITSELF, naming the junction and the
// measured gap. Proves it discriminates: (1) the real, uncorrupted geometry closes; (2) the hand-broken
// geometry is declined AT canalArmLoop with ~75 in the reason; (3) restoring the real geometry closes
// again (the break, not the tolerance, caused the decline).
func TestCanalArmLoop_EarlierJunctionGapRejected(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	_, boundaries, centres, scale := n7CanalWeldInputs(t, w, arms, res)
	mid := n7MidArmIndex(t, arms)
	i := torusArmIndex(t, arms)
	arm, centre := arms[i], centres[i]
	tol := res.Weld() * scale

	rail, rev, ok := canalArmCornerRail(boundaries, centres[i], i, mid)
	if !ok {
		t.Fatalf("torus arm: no corner-side rail")
	}
	set, ok := solveArmSetback(arm, centre, w.radius, scale, res)
	if !ok {
		t.Fatalf("torus arm: solveArmSetback declined")
	}
	wi := cornerWeld{center: centre, radius: w.radius, arms: []armSetback{set}}
	h0, h1, ok := canalArmHostRails(arm, set, wi, res)
	if !ok {
		t.Fatalf("torus arm: host rails declined")
	}
	far, ok := farCrossSectionArc(set.arm, w.radius, h0.from, h1.from)
	if !ok {
		t.Fatalf("torus arm: far cross-section arc declined")
	}

	// (1) the real, unbroken geometry closes — canalArmLoop's own gate accepts it.
	if _, reason := canalArmLoop(h0, h1, far, rail, rev, tol); reason != "" {
		t.Fatalf("torus arm: canalArmLoop declined the real N7 geometry: %s", reason)
	}

	// (2) hand-introduce an EARLIER-junction break (junction 2: hFar.to → far.from), NOT the final one
	// (junction 4: hNear.to → corner[0]): translate the far cross-section arc's endpoints off their
	// chain points by 75 units, mirroring the W2 reviewer's Mutation C gap magnitude.
	const gapMag = 75.0
	broken := far
	broken.from = math.P3(broken.from.X+gapMag, broken.from.Y, broken.from.Z)
	broken.to = math.P3(broken.to.X+gapMag, broken.to.Y, broken.to.Z)
	_, brokenReason := canalArmLoop(h0, h1, broken, rail, rev, tol)
	if brokenReason == "" {
		t.Fatal("canalArmLoop must decline an earlier-junction break, not silently build a malformed loop")
	}
	if !strings.Contains(brokenReason, "gap") {
		t.Fatalf("canalArmLoop's decline reason must carry the measured gap, got: %s", brokenReason)
	}
	t.Logf("earlier-junction break correctly caught at canalArmLoop's own gate: %s", brokenReason)

	// (3) restore: the SAME uncorrupted inputs still close — the break, not the tolerance, caused (2).
	if _, reason := canalArmLoop(h0, h1, far, rail, rev, tol); reason != "" {
		t.Fatalf("torus arm: canalArmLoop declined the restored geometry: %s", reason)
	}
}

// TestEndArcForCentre_MatchesByProximity proves the by-proximity mapping: the end arc endArcForCentre
// picks for a wall arm's reflected centre is the cross-section arc ABOUT that centre (its fitted circle
// centre coincides with the arm centre), NOT the other wall arm's — the discriminator against index order.
func TestEndArcForCentre_MatchesByProximity(t *testing.T) {
	t.Parallel()
	w, arms, res := n7CornerFill(t)
	_, boundaries, centres, _ := n7CanalWeldInputs(t, w, arms, res)
	mid := n7MidArmIndex(t, arms)
	for i := range arms {
		if i == mid {
			continue
		}
		rail, _, ok := endArcForCentre(boundaries, centres[i])
		if !ok {
			t.Fatalf("wall arm %d: endArcForCentre declined", i)
		}
		fitted, ok := endArcCentre(rail)
		if !ok {
			t.Fatalf("wall arm %d: could not fit the selected end arc's circle", i)
		}
		if d := float64(fitted.DistanceTo(centres[i])); d > 1e-6 {
			t.Fatalf("wall arm %d: selected end arc centred at %v, %.3e from its reflected centre %v (wrong arc?)", i, fitted, d, centres[i])
		}
	}
}

// torusArmIndex returns the index of the single torus arm (s_5), or fails the test.
func torusArmIndex(t *testing.T, arms []edgeFillet) int {
	t.Helper()
	for i := range arms {
		if _, ok := arms[i].armSurface.(geom.Torus); ok {
			return i
		}
	}
	t.Fatalf("no torus arm among the N7 arms")
	return -1
}

// otherEndArc returns the FARTHER end arc from centre (and its rev) — the mis-mapped choice for the
// swap (the wall arm's OWN arc is the nearer one endArcForCentre picks). Matched by fitted centre, not
// by curve equality (geom.BSplineCurve is uncomparable).
func otherEndArc(b canalBoundaries, centre math.Point3) (geom.Curve3, bool) {
	c0, _ := endArcCentre(b.endArcs[0])
	c1, _ := endArcCentre(b.endArcs[1])
	if float64(centre.DistanceTo(c0)) <= float64(centre.DistanceTo(c1)) {
		return b.endArcs[1], b.endArcsRev[1] // arm is nearer arc 0 → the swap is arc 1
	}
	return b.endArcs[0], b.endArcsRev[0]
}

// assertArmFaceCloses checks the arm face has exactly ONE loop of len(cornerSamples)+3 points whose
// corner-rail portion (the first len(cornerSamples) points) starts the ring, and whose corner rail's FAR
// endpoint meets the first trailing (host-rail) point — so the ring closes with no junction gap.
func assertArmFaceCloses(t *testing.T, face filletFace, rail geom.Curve3, rev bool, tol float64, i int) {
	t.Helper()
	if len(face.loops) != 1 {
		t.Fatalf("arm %d: %d loops, want 1", i, len(face.loops))
	}
	pts := face.loops[0].pts
	corner := sampleCurve3Open(rail, rev)
	if len(pts) != len(corner)+3 {
		t.Fatalf("arm %d: loop has %d points, want %d (corner samples + far host rail + far arc + near host rail)", i, len(pts), len(corner)+3)
	}
	if d := float64(pts[0].DistanceTo(corner[0])); d > tol {
		t.Fatalf("arm %d: loop does not start at the corner rail (off %.3e)", i, d)
	}
	lo, hi := rail.Domain()
	far := rail.PointAt(hi)
	if float64(corner[0].DistanceTo(far)) < tol { // corner[0] is the hi end → far is the lo end
		far = rail.PointAt(lo)
	}
	if d := float64(pts[len(corner)].DistanceTo(far)); d > tol {
		t.Fatalf("arm %d: corner rail's far end does not meet the host rail (junction gap %.3e)", i, d)
	}
}

// assertCornerRailShared is the WATERTIGHT gate: every sampled point of the arm face's corner rail is
// point-identical (within a deterministic 1e-9) to a point on the CANAL PATCH's boundary loop — proof the
// two faces weld along the SAME curve, not two re-derived coincident ones. It logs the max residual.
func assertCornerRailShared(t *testing.T, face filletFace, rail geom.Curve3, rev bool, patch CornerBlendPatch, i int) {
	t.Helper()
	patchPts := patch.Loops[0].pts
	corner := sampleCurve3Open(rail, rev)
	maxDev := 0.0
	for k, p := range corner {
		if float64(p.DistanceTo(face.loops[0].pts[k])) > 1e-12 {
			t.Fatalf("arm %d: corner rail point %d is not the loop's own point (canalArmLoop must sample it first)", i, k)
		}
		d := nearestPointDist(patchPts, p)
		if d > 1e-9 {
			t.Fatalf("arm %d: corner rail point %d = %v is %.3e off every canal-patch boundary point (NOT the shared curve)", i, k, p, d)
		}
		maxDev = stdmath.Max(maxDev, d)
	}
	t.Logf("arm %d: corner rail ↔ canal patch boundary max residual = %.3e (watertight identity)", i, maxDev)
}

// assertArmHostRailsOnHost re-derives the arm's two host contact rails and asserts each lies on its host
// face's surface within tol (a straight ruling stays on a plane / on the cylinder it rules; a torus
// contact arc stays on the wall / cap).
func assertArmHostRailsOnHost(t *testing.T, arm edgeFillet, centre math.Point3, w cornerWeld, scale float64, res Resolution, tol float64, i int) {
	t.Helper()
	set, ok := solveArmSetback(arm, centre, w.radius, scale, res)
	if !ok {
		t.Fatalf("arm %d: solveArmSetback declined at %v", i, centre)
	}
	wi := cornerWeld{center: centre, radius: w.radius, arms: []armSetback{set}}
	h0, h1, ok := canalArmHostRails(arm, set, wi, res)
	if !ok {
		t.Fatalf("arm %d: host rails declined", i)
	}
	assertEndSegOnSurface(t, h0, arm.a.Geometry(), tol, i, "host a")
	assertEndSegOnSurface(t, h1, arm.b.Geometry(), tol, i, "host b")
}

// assertArmFarArcIsCrossSection re-derives the arm's far cross-section arc and asserts it is a radius-r
// arc lying on the arm surface — the arm's terminal cross-section.
func assertArmFarArcIsCrossSection(t *testing.T, arm edgeFillet, centre math.Point3, w cornerWeld, scale float64, res Resolution, tol float64, i int) {
	t.Helper()
	set, _ := solveArmSetback(arm, centre, w.radius, scale, res)
	wi := cornerWeld{center: centre, radius: w.radius, arms: []armSetback{set}}
	h0, h1, _ := canalArmHostRails(arm, set, wi, res)
	far, ok := farCrossSectionArc(set.arm, w.radius, h0.from, h1.from)
	if !ok {
		t.Fatalf("arm %d: far cross-section arc declined", i)
	}
	arc, ok := far.curve.(geom.Arc3d)
	if !ok {
		t.Fatalf("arm %d: far rail is %T, want geom.Arc3d", i, far.curve)
	}
	if stdmath.Abs(arc.Radius-w.radius) > tol {
		t.Fatalf("arm %d: far arc radius %.4f, want r=%.1f", i, arc.Radius, w.radius)
	}
	assertEndSegOnSurface(t, far, arm.armSurface, tol, i, "far cross-section")
}

// assertEndSegOnSurface samples an endSeg (endpoints, midpoint, and — for an arc — interior) and asserts
// every sample lies within tol of surf (nearest-point distance).
func assertEndSegOnSurface(t *testing.T, s endSeg, surf geom.Surface, tol float64, i int, name string) {
	t.Helper()
	samples := []math.Point3{s.from, s.to}
	if arc, ok := s.curve.(geom.Arc3d); ok {
		for k := 1; k < 8; k++ {
			samples = append(samples, arc.PointAt(float64(k)/8)) // sample the ARC, not the chord
		}
	} else {
		samples = append(samples, midpoint3(s.from, s.to)) // straight ruling: chord midpoint is on it
	}
	for _, p := range samples {
		if d := pointOffSurface(surf, p); d > tol {
			t.Fatalf("arm %d: %s rail point %v is %.3e off its host %T (tol %.3e)", i, name, p, d, surf, tol)
		}
	}
}

// pointOffSurface is the nearest-point distance from p to surf.
func pointOffSurface(surf geom.Surface, p math.Point3) float64 {
	_, _, foot := geom.ClosestPointOnSurface(surf, p)
	return float64(foot.DistanceTo(p))
}

// nearestPointDist returns the distance from p to its nearest point in pts.
func nearestPointDist(pts []math.Point3, p math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, q := range pts {
		best = stdmath.Min(best, float64(p.DistanceTo(q)))
	}
	return best
}

// midpoint3 is the midpoint of a,b.
func midpoint3(a, b math.Point3) math.Point3 {
	return math.P3((float64(a.X)+float64(b.X))/2, (float64(a.Y)+float64(b.Y))/2, (float64(a.Z)+float64(b.Z))/2)
}
