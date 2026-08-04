// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The canal tier (M6' C1-C3): Fits classifies by the RailLoop.Canal payload pointer; Build composes
// the offset-SSI spine + cross-section loft (geom.CanalCornerFill) into the rolling-ball corner patch
// and certifies it, or honest-rejects (→ coons4). Assertions are against the DRAWEXE oracle (area
// 90.194), never our own output. Supersedes corner_provider_plate_test.go (deleted with the plate
// stub, ADR-C3): canalProvider took the plate's old tier slot.

// TestCanalProviderFits pins the Fits truth table: true iff the loop carries a non-nil Canal payload
// AND is valence-4. A default (Canal==nil) valence-4 loop and a Canal-populated valence-3 loop must
// both decline — the payload pointer is the ONLY classification signal (ADR-C2; canal-corner-seam-
// architecture.md §2), not loop shape alone.
func TestCanalProviderFits(t *testing.T) {
	p := canalProvider{}

	unmarkedV4 := quarterCylLoop(t, 8) // Canal defaults to nil
	if p.Fits(unmarkedV4) {
		t.Fatal("canalProvider.Fits must be false for an unmarked (Canal==nil) valence-4 loop")
	}

	markedV3 := sphereTriLoop(t, 4)
	markedV3.Canal = &CanalCorner{Radius: 4}
	if p.Fits(markedV3) {
		t.Fatal("canalProvider.Fits must be false for a Canal-populated valence-3 loop (needs valence 4 too)")
	}

	markedV4 := quarterCylLoop(t, 8)
	markedV4.Canal = &CanalCorner{Radius: 5}
	if !p.Fits(markedV4) {
		t.Fatal("canalProvider.Fits must be true for a Canal-populated valence-4 loop")
	}
}

// TestCanalProviderBuildDeclinesOnIncompletePayload pins the do-no-harm floor: a loop marked Canal but
// WITHOUT the roll hosts / ends (an incomplete payload) makes geom.CanalCornerFill error (host count
// != 2), so Build honest-rejects (ok=false) → resolveBlend falls through to coons4. A solver that
// cannot build the surface never fabricates a patch (ADR-0051 ADR-3).
func TestCanalProviderBuildDeclinesOnIncompletePayload(t *testing.T) {
	loop := quarterCylLoop(t, 8)
	loop.Canal = &CanalCorner{Radius: 5} // no Rolls, no Ends
	p := canalProvider{}
	if !p.Fits(loop) {
		t.Fatal("fixture must Fit for this test to be meaningful")
	}
	if _, _, ok := p.Build(loop, blendScale()); ok {
		t.Fatal("canalProvider.Build must decline on an incomplete Canal payload (no roll hosts)")
	}
}

// The tier-order assertion (analyticSphere, canal, coons4, tri3) lives in TestResolveBlendTiersOrder;
// the B3/octant → BlendKindSphere and non-canal-valence-4 → BlendKindCoons4 cases live in
// TestResolveBlendSphereWins / TestResolveBlendCoons4 (all corner_resolve_test.go) — not duplicated.

// TestN7LoopResolvesCanal is the C3 gate at the resolveBlend seam: the real N7 fixture extracts a
// Canal-populated valence-4 loop that the canal tier now BUILDS and certifies (BlendKindCanal), with
// the emergent DRAWEXE-oracle area 90.194 within the C2-justified 0.05% and the Loops emitted on the
// canal's OWN four boundary isoparms (watertight — every boundary lies ON the surface, unlike the
// received amid rail which sits 0.28 off it). The certificate is independently Valid at the model scale.
func TestN7LoopResolvesCanal(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	loop, ok := extractCurvedCorner(w, arms, res)
	if !ok || loop.Valence() != 4 || loop.Canal == nil {
		t.Fatalf("extractCurvedCorner: want a Canal-marked valence-4 N7 loop; ok=%v valence=%d canal=%v", ok, loop.Valence(), loop.Canal != nil)
	}
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCanal {
		t.Fatalf("N7 loop must resolve to the canal tier; ok=%v kind=%q", ok, patch.Kind)
	}
	if area := geom.SurfaceArea(patch.Surface); stdmath.Abs(area-90.194)/90.194 > 5e-4 {
		t.Fatalf("N7 canal area = %.5f, want 90.194 within 0.05%%", area)
	}
	assertLoopsAreCanalBoundary(t, patch, loop, res)
}

// TestCanalPatchLoopsOnSurface is the C3-review regression: the emitted patch Loops must lie ON the
// canal surface within the model weld. This holds for the canal's OWN boundary isocurves (by
// construction) and FAILS at ~0.28 for the pre-fix received-rails emission (the mid-arm cross-section
// amid, centred at C′ which is not a canal boundary, is 0.28 off the surface). Proving it here makes
// the on-surface property TESTED, so re-introducing the received rails is a red test, not a silently
// malformed trim edge (a face with an edge 0.28 off its own surface).
func TestCanalPatchLoopsOnSurface(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	loop, _ := extractCurvedCorner(w, arms, res)
	patch, ok := resolveBlend(loop, res)
	if !ok || patch.Kind != BlendKindCanal {
		t.Fatalf("N7 loop must resolve to the canal tier; ok=%v kind=%q", ok, patch.Kind)
	}
	surf, isBSpline := patch.Surface.(geom.BSplineSurface)
	if !isBSpline {
		t.Fatalf("canal patch surface is %T, want geom.BSplineSurface", patch.Surface)
	}
	maxDev := maxLoopToSurface(surf, patch.Loops)
	if maxDev > res.Weld() {
		t.Fatalf("max loop-to-surface distance = %.3e exceeds weld %.3e — emitted Loops are not on the canal surface", maxDev, res.Weld())
	}
}

// maxLoopToSurface is the max nearest-point distance from any loop point to surf (the on-surface
// witness assertLoopsOnCanal gates Build with; recomputed in the test to prove the property directly).
func maxLoopToSurface(surf geom.BSplineSurface, loops []filletLoop) float64 {
	maxDev := 0.0
	for _, l := range loops {
		for _, p := range l.pts {
			_, _, foot := geom.ClosestPointOnSurface(surf, p)
			maxDev = stdmath.Max(maxDev, float64(foot.DistanceTo(p)))
		}
	}
	return maxDev
}

// TestN7CanalCertificateValid witnesses the canal patch's certificate directly: the surface welds G1
// to its two fillet arms (the end cross-sections) AND is tangent to both roll hosts (the foot-loci),
// so all five fields pass at the model scale.
func TestN7CanalCertificateValid(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	loop, _ := extractCurvedCorner(w, arms, res)
	_, cert, ok := canalProvider{}.Build(loop, res)
	if !ok {
		t.Fatal("canalProvider.Build must succeed on the real N7 loop")
	}
	if !cert.Valid(res) {
		t.Fatalf("canal certificate not Valid: Closed=%v WeldsArms=%v NoFold=%v MaxDev=%.3e MaxAngleDev=%.3e",
			cert.Closed, cert.WeldsArms, cert.NoFold, cert.MaxDev, cert.MaxAngleDev)
	}
}

// TestCurvedCornerFaceAdmitsCanal proves the weld Kind-gate admits BlendKindCanal (mirrors the coons4
// arm of the whitelist switch): driving curvedCornerFace on the real N7 corner returns a fillet face
// whose surface is the CANAL BSpline patch — NOT the do-no-harm sphere fallback — so the canal patch
// is not silently dropped by the whitelist's default arm.
func TestCurvedCornerFaceAdmitsCanal(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	dummy, err := geom.NewSphere(w.center, 5) // the fallback surface, distinct from the canal BSpline
	if err != nil {
		t.Fatalf("dummy sphere: %v", err)
	}
	face, ok := curvedCornerFace(w, dummy, arms, res)
	if !ok {
		t.Fatal("curvedCornerFace declined the N7 canal corner")
	}
	if _, isBSpline := face.surface.(geom.BSplineSurface); !isBSpline {
		t.Fatalf("weld dropped the canal patch: face surface is %T, want geom.BSplineSurface (the canal)", face.surface)
	}
}

// assertLoopsAreCanalBoundary proves the emitted patch Loops are the canal's OWN four boundary
// isoparms (the C3-review fix), NOT the received rails: (1) one loop of 4·ringSegSamples points, every
// point ON the surface within weld; (2) the ring is CLOSED and correctly ordered — its four sides
// start at the four patch corners (u0,v0)→(u1,v0)→(u1,v1)→(u0,v1); (3) the two v-boundaries still equal
// the received a0/a1 end cross-section arcs (each v-boundary point lies on a received end-arc rail's
// circle), so the fix preserves the GOOD rails and only replaces the two off-surface u-boundaries.
func assertLoopsAreCanalBoundary(t *testing.T, patch CornerBlendPatch, loop RailLoop, res Resolution) {
	t.Helper()
	surf := patch.Surface.(geom.BSplineSurface)
	weld := res.Weld()
	n := ringSegSamples
	if len(patch.Loops) != 1 || len(patch.Loops[0].pts) != 4*n {
		t.Fatalf("canal Loops = %d loops, %d pts; want 1 loop of %d (4·ringSegSamples)", len(patch.Loops), loopPtCount(patch.Loops), 4*n)
	}
	if dev := maxLoopToSurface(surf, patch.Loops); dev > weld {
		t.Fatalf("canal boundary loop is %.3e off the surface, want <= weld %.3e", dev, weld)
	}
	assertRingClosedAtCorners(t, surf, patch.Loops[0].pts, n, weld)
	assertVBoundariesAreEndArcs(t, patch.Loops[0].pts, loop, n, weld)
}

// loopPtCount totals the points across loops (for the diagnostic in the count assertion).
func loopPtCount(loops []filletLoop) int {
	n := 0
	for _, l := range loops {
		n += len(l.pts)
	}
	return n
}

// assertRingClosedAtCorners proves the ring is closed and correctly ordered: each of the four sides is
// sampled OPEN including its NEAR endpoint, so the four patch corners (u0,v0)→(u1,v0)→(u1,v1)→(u0,v1)
// appear as the sides' start points at indices 0, n, 2n, 3n — visiting every corner in ring order.
func assertRingClosedAtCorners(t *testing.T, surf geom.BSplineSurface, pts []math.Point3, n int, weld float64) {
	t.Helper()
	u0, u1 := surf.UDomain()
	v0, v1 := surf.VDomain()
	corners := [4]math.Point3{surf.PointAt(u0, v0), surf.PointAt(u1, v0), surf.PointAt(u1, v1), surf.PointAt(u0, v1)}
	for i, c := range corners {
		if d := float64(pts[i*n].DistanceTo(c)); d > weld {
			t.Fatalf("ring side %d start = %v is %.3e from patch corner %v (want <= weld %.3e) — loop not closed in order", i, pts[i*n], d, c, weld)
		}
	}
}

// assertVBoundariesAreEndArcs proves the two v-boundary runs (sides 0 and 2, at pts[0:n] and
// pts[2n:3n]) still coincide with the received a0/a1 end cross-section arcs — each v-boundary point
// lies on a received end-arc rail's circle (radius r about the arc centre, in the arc plane) within
// weld. This is the invariant the fix must preserve: the GOOD end rails are unchanged; only the two
// off-surface u-boundaries switched to the true foot-loci.
func assertVBoundariesAreEndArcs(t *testing.T, pts []math.Point3, loop RailLoop, n int, weld float64) {
	t.Helper()
	arcs := receivedEndArcs(loop, weld)
	if len(arcs) != 2 {
		t.Fatalf("want exactly 2 received end-arc rails (arc centre = a spine end), got %d", len(arcs))
	}
	for _, run := range [2][]math.Point3{pts[0:n], pts[2*n : 3*n]} {
		if d := minCircleDistOverArcs(run, arcs); d > weld {
			t.Fatalf("a v-boundary run is %.3e off both received end-arc circles (want <= weld %.3e) — the fix altered a good end rail", d, weld)
		}
	}
}

// receivedEndArcs returns the received rails that are radius-r Arc3d whose centre is a spine end (a0,
// a1) — the two end cross-section rails, excluding the mid-arm amid arc (centred 10 away at C′).
func receivedEndArcs(loop RailLoop, weld float64) []geom.Arc3d {
	var arcs []geom.Arc3d
	for _, s := range loop.Sides {
		if arc, ok := s.Curve.(geom.Arc3d); ok && centreIsSpineEnd(arc.Center, loop.Canal.Ends, weld) {
			arcs = append(arcs, arc)
		}
	}
	return arcs
}

// minCircleDistOverArcs is the smallest (over the candidate arcs) worst-point circle distance for a
// run: for each arc, the max over run points of |dist(p,centre) − r| + |axial offset from the arc
// plane| (zero iff every point is exactly on that arc's circle).
func minCircleDistOverArcs(run []math.Point3, arcs []geom.Arc3d) float64 {
	best := stdmath.Inf(1)
	for _, arc := range arcs {
		worst := 0.0
		for _, p := range run {
			radial := arc.Center.VectorTo(p)
			axial := radial.Dot(arc.Normal.AsVector())
			d := stdmath.Abs(float64(p.DistanceTo(arc.Center))-arc.Radius) + stdmath.Abs(float64(axial))
			worst = stdmath.Max(worst, d)
		}
		best = stdmath.Min(best, worst)
	}
	return best
}
