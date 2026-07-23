// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"os"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// u4StepPath is the U4 corpus fixture (#2007 Group C): a box with a convex edge x=10,y=-20,z∈[-15,15]
// filleted r=5, pierced by TWO bosses — Cylinder r8 on face y=-20 and an oblique EllipticalCylinder on
// face x=10 — whose footprints both dip into the fillet band (qualifying==2, dual-host). It is the
// SAME STEP file the occtparity corpus scores as "U4"; kernel/ops reads it directly (not through the
// occtparity package, which itself imports kernel/ops — importing it back would cycle) so these unit
// tests can drive detectObstacles/partitionUnionStations/assembleDualObstacleSet at the ops level.
const u4StepPath = "../../model/feature/occtparity/fixtures/simple/U4.step"

// u4EdgeMidpoint is the corpus pick locator for U4's filleted edge (corpus.json "U4" record): midpoint
// (10,-20,0), radius 5.
var u4EdgeMidpoint = math.P3(10, -20, 0)

// importU4 imports the U4 corpus fixture body, mirroring slabWithColumn's STEP-import pattern
// (fillet_obstacle_watertight_test.go) so the topology under test is a real imported B-rep.
func importU4(t *testing.T) *topo.Body {
	t.Helper()
	data, err := os.ReadFile(u4StepPath)
	if err != nil {
		t.Fatalf("read U4 fixture: %v", err)
	}
	bodies, _, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil || len(bodies) != 1 {
		t.Fatalf("import U4: %v (n=%d)", err, len(bodies))
	}
	return bodies[0]
}

// u4Fillet solves U4's real fillet feature end to end (resolveFilletPicks → computeCorners →
// computeFillets), the same sequence FilletEdges runs internally, and returns the imported body
// alongside its resolved edgeFillet list (always len 1 for U4) and the shared Resolution — the common
// precondition every U4 kernel/ops-level test builds on. Returning the body too (not just the
// edgeFillet) matters for U4-1's buildDualClosure tests: filletRebuildMaps keys by *topo.Face pointer
// identity, so maps and the edgeFillet/detections consuming them must come from the SAME body instance,
// not a second independent importU4 call.
func u4Fillet(t *testing.T) (*topo.Body, []edgeFillet, Resolution) {
	t.Helper()
	body := importU4(t)
	edge := edgeAtMidpoint(body, u4EdgeMidpoint)
	if edge == nil {
		t.Fatalf("U4 fixture: filleted edge (midpoint %v) not found", u4EdgeMidpoint)
	}
	picks, err := resolveFilletPicks(body, filletPicksFor([][]byte{edge.ReferenceKey()}, 5))
	if err != nil {
		t.Fatalf("resolveFilletPicks(U4): %v", err)
	}
	blends, miters, err := computeCorners(picks)
	if err != nil {
		t.Fatalf("computeCorners(U4): %v", err)
	}
	fils, err := computeFillets(body, picks, blends, miters, FillConcaveOutward, nil)
	if err != nil {
		t.Fatalf("computeFillets(U4): %v", err)
	}
	if len(fils) != 1 {
		t.Fatalf("U4: want 1 edgeFillet, got %d", len(fils))
	}
	return body, fils, ResolutionForBody(body)
}

// u4EdgeFillet is u4Fillet's single-edgeFillet convenience form — the shape the U4-0 unit tests need
// (they never touch the body directly).
func u4EdgeFillet(t *testing.T) (edgeFillet, Resolution) {
	t.Helper()
	_, fils, res := u4Fillet(t)
	return fils[0], res
}

// TestDetectObstaclesU4ReturnsBothHosts pins the U4-0 extractor (derivation §3.1, item 1): U4's
// filleted edge carries TWO qualifying obstacles — Host A (Cylinder r8 on y=-20) and Host B (the
// oblique EllipticalCylinder on x=10) — and detectObstacles must keep BOTH, where the pre-existing
// detectObstacle keeps neither (qualifying==2 honest-rejects to obstacleDetection{}, false). This is
// the do-no-harm precondition every later U4 slice builds on: if this ever returns only 1 or 0, the
// dual-host rebuild has nothing to work with.
func TestDetectObstaclesU4ReturnsBothHosts(t *testing.T) {
	ef, res := u4EdgeFillet(t)
	dets, ok := detectObstacles(ef, res)
	if !ok || len(dets) != 2 {
		t.Fatalf("detectObstacles(U4) = (%d dets, ok=%v), want (2, true)", len(dets), ok)
	}
	if dets[0].hostIsA == dets[1].hostIsA {
		t.Fatalf("detectObstacles(U4): both detections report hostIsA=%v, want one A one B", dets[0].hostIsA)
	}
	// detectObstacle (the single-host entry) must still honest-reject this same edge: qualifying==2 is
	// exactly the case it is not scoped to handle (byte-identical do-no-harm, derivation §3.3).
	if _, ok := detectObstacle(ef, res); ok {
		t.Errorf("detectObstacle(U4) must still honest-reject the dual-host edge (qualifying==2)")
	}
}

// TestPartitionUnionStationsU4 pins the U4-0 station partition (derivation §3.1 item 2, §1.1/§1.2): the
// DRAWEXE ground truth places the A-nodes (Cylinder r8 crossing the A-tangent) at z=±6.240 and the
// B-nodes (the oblique ellipse crossing the B-tangent) at z=±6.633, so the union band splits into 3
// spans — B-only sliver [-6.633,-6.240], A+B core [-6.240,+6.240], B-only sliver [+6.240,+6.633] — with
// no panel geometry built yet (that is U4-3/U4-4). Asserted to 1e-3, which is tight against the
// station spacing (~0.39 between an A-node and its neighbouring B-node) yet loose enough for the
// sampled-rim crossing solve's own tolerance.
func TestPartitionUnionStationsU4(t *testing.T) {
	ef, res := u4EdgeFillet(t)
	dets, ok := detectObstacles(ef, res)
	if !ok || len(dets) != 2 {
		t.Fatalf("fixture precondition: detectObstacles(U4) = (%d, %v), want (2, true)", len(dets), ok)
	}
	spans := partitionUnionStations(dets, ef)
	if len(spans) != 3 {
		t.Fatalf("partitionUnionStations(U4) = %d spans, want 3 (sliver, core, sliver); got %+v", len(spans), spans)
	}
	// 1e-3 RELATIVE (model-relative, not a bare absolute epsilon — ADR-0042): the B-node stations come
	// from lerpAtZero's chord interpolation across the sampled elliptical rim (obstacleRimSamples=64),
	// whose sagitta error at this model's scale is a few thousandths of a unit — well inside 1e-3 of the
	// ~6.6 station magnitude, but tighter than a fixed absolute 1e-3 would allow.
	wantStations := [4]float64{-6.633, -6.240, 6.240, 6.633}
	gotStations := [4]float64{spans[0].zLo, spans[0].zHi, spans[1].zHi, spans[2].zHi}
	for i, want := range wantStations {
		if got, tol := gotStations[i], 1e-3*absDiff(want, 0); absDiff(got, want) > tol {
			t.Errorf("station[%d] = %.4f, want %.4f (±%g)", i, got, want, tol)
		}
	}
	// span[1].zLo must weld to span[0].zHi (no gap/overlap between consecutive panels) — exact by
	// construction (partitionUnionStations builds spans from one shared sorted station list), so a bare
	// weld-scale epsilon suffices here.
	const chainTol = 1e-9
	if absDiff(spans[1].zLo, spans[0].zHi) > chainTol || absDiff(spans[2].zLo, spans[1].zHi) > chainTol {
		t.Errorf("spans are not chained end-to-end: %+v", spans)
	}
	// core span (both hosts dipping) is the middle one; both slivers are B-only (A still on-cylinder).
	if !spans[1].hostA || !spans[1].hostB {
		t.Errorf("core span[1] = %+v, want hostA=true hostB=true", spans[1])
	}
	for i, sliver := range []panelSpan{spans[0], spans[2]} {
		if sliver.hostA || !sliver.hostB {
			t.Errorf("sliver span[%d] = %+v, want hostA=false hostB=true", i, sliver)
		}
	}
}

// u4OracleAreas are the DRAWEXE per-face oracles for the FOUR B-spline fillet panels (derivation §0):
// two slivers at 3.039 (result_5/result_9) and two split cores at 30.334 (result_13/result_14).
var u4SliverOracle, u4CoreOracle = 3.039, 30.334

// TestFilletU4DualHostWatertight is the U4-5 payoff gate (#2007 Group C): the whole dual-host pipeline ON.
// FilletEdges on U4's convex edge (two bosses dipping into one r=5 fillet, qualifying==2) must now build
// the full welded body — 2 notched hosts + 2 split walls + 2 wings + 4 corner-blend panels — that clears
// the FULL hardened watertight bar (Valid && Closed && Manifold && HolesContained && IsSolid), matches the
// DRAWEXE 16-FACE / every-face-WIRE:1 topology, and lands the whole-body surface area within the corpus's
// 1% of the oracle 6583.29. This is the outcome the U4-0..U4-4b slices built toward, replacing the old
// HolesContained=false do-no-harm baseline (the dual-host defect held in quarantine until this slice).
func TestFilletU4DualHostWatertight(t *testing.T) {
	res := filletU4Body(t)
	rep := Validate(res)
	watertight := rep.Valid && rep.Closed && rep.Manifold && rep.HolesContained && res.IsSolid()
	if !watertight {
		t.Fatalf("U4 must be watertight: Valid=%v Closed=%v Manifold=%v HolesContained=%v IsSolid=%v issues=%v",
			rep.Valid, rep.Closed, rep.Manifold, rep.HolesContained, res.IsSolid(), rep.Issues)
	}
	if n := len(res.Faces()); n != 16 {
		t.Errorf("U4: %d result faces, want 16 (DRAWEXE nbshapes)", n)
	}
	total := 0.0
	for _, f := range res.Faces() {
		if len(f.Loops()) != 1 {
			t.Errorf("U4 face has %d loops, want 1 (every result face WIRE:1)", len(f.Loops()))
		}
		total += MeshArea(TessellateFace(f, Quality{ChordTolerance: 1e-3}))
	}
	const wholeBodyOracle, corpusDeps = 6583.29, 0.01
	if rel := absDiff(total, wholeBodyOracle) / wholeBodyOracle; rel > corpusDeps {
		t.Errorf("U4 whole-body area %.4f vs oracle %.4f (rel %.4f%% > %.1f%%)", total, wholeBodyOracle, rel*100, corpusDeps*100)
	}
}

// TestFilletU4PerFaceProductionTessellation is the ★★ carried #2009/U4-4 gate: each of the FOUR B-spline
// fillet panels must tessellate through the PRODUCTION path (TessellateFace → nurbsPcurveMesh) to its own
// DRAWEXE oracle area — FOLD-FREE — not merely sum to the right whole-body total (which masks a bad panel
// against a compensating one). The two cylinder wings (analytic, exact) round out the six fillet faces.
// This is the real-U4 validation the #2009 aspect-aware starved-rail fix deferred: if a panel folds or
// misses its oracle in production, the fix did NOT cover the real faces and the slice is BLOCKED.
func TestFilletU4PerFaceProductionTessellation(t *testing.T) {
	res := filletU4Body(t)
	var slivers, cores int
	for _, f := range res.Faces() {
		if _, isBS := f.Geometry().(geom.BSplineSurface); !isBS {
			continue
		}
		m := TessellateFace(f, Quality{ChordTolerance: 1e-3})
		area := MeshArea(m)
		if folds := FoldEdgeCount(m); folds != 0 {
			t.Errorf("U4 fillet panel (prodArea %.4f): %d fold edges, want 0 — production tessellation folds", area, folds)
		}
		switch {
		case within(area, u4SliverOracle, 0.01):
			slivers++
			t.Logf("sliver panel prodArea=%.4f (oracle %.3f, rel %.4f%%)", area, u4SliverOracle, absDiff(area, u4SliverOracle)/u4SliverOracle*100)
		case within(area, u4CoreOracle, 0.01):
			cores++
			t.Logf("core panel prodArea=%.4f (oracle %.3f, rel %.4f%%)", area, u4CoreOracle, absDiff(area, u4CoreOracle)/u4CoreOracle*100)
		default:
			t.Errorf("U4 fillet panel prodArea %.4f matches neither sliver oracle %.3f nor core oracle %.3f within 1%%",
				area, u4SliverOracle, u4CoreOracle)
		}
	}
	if slivers != 2 || cores != 2 {
		t.Errorf("U4 per-face: %d slivers + %d cores tessellated to oracle, want 2 + 2", slivers, cores)
	}
}

// within reports whether a is within rel (relative) of want.
func within(a, want, rel float64) bool { return absDiff(a, want)/want <= rel }

// filletU4Body drives FilletEdges over U4's dual-host convex edge (the real feature path end to end).
func filletU4Body(t *testing.T) *topo.Body {
	t.Helper()
	body := importU4(t)
	edge := edgeAtMidpoint(body, u4EdgeMidpoint)
	if edge == nil {
		t.Fatalf("U4 fixture: filleted edge (midpoint %v) not found", u4EdgeMidpoint)
	}
	res, err := FilletEdges(body, [][]byte{edge.ReferenceKey()}, 5)
	if err != nil {
		t.Fatalf("FilletEdges(U4): %v", err)
	}
	return res
}

// absDiff avoids importing math.Abs's stdmath alias for a single call site.
func absDiff(a, b float64) float64 {
	if a < b {
		return b - a
	}
	return a - b
}

// u4ClosureFixture extends u4Fillet with U4's dual-host detections, their union spans, and the
// filletRebuildMaps buildNotchedHost needs — the full precondition buildDualClosure's U4-1 tests below
// drive it with. All returned values are built from the SAME body/edgeFillet (see u4Fillet's docstring
// on why that identity matters for the maps lookup).
func u4ClosureFixture(t *testing.T) (edgeFillet, []obstacleDetection, []panelSpan, filletRebuildMaps) {
	t.Helper()
	body, fils, res := u4Fillet(t)
	ef := fils[0]
	dets, ok := detectObstacles(ef, res)
	if !ok || len(dets) != 2 {
		t.Fatalf("fixture precondition: detectObstacles(U4) = (%d, %v), want (2, true)", len(dets), ok)
	}
	spans := partitionUnionStations(dets, ef)
	maps, _ := filletBuildMaps(body, fils)
	return ef, dets, spans, maps
}

// filletFaceArea tessellates a single filletFace and sums its triangle areas — the empirical way to
// prove buildSplitObstacleWall's rim split does not change the wall's true surface area: assembleBody
// welds one face's loop into real topo geometry (it does not require the input set to be watertight, so
// a lone wall face builds fine), and TessellateFace/MeshArea then measure it exactly the way the corpus
// oracle's own area gates do elsewhere in this package.
func filletFaceArea(t *testing.T, f filletFace) float64 {
	t.Helper()
	faces := assembleBody([]filletFace{f}).Faces()
	if len(faces) != 1 {
		t.Fatalf("filletFaceArea: assembleBody produced %d faces, want 1", len(faces))
	}
	mesh := TessellateFace(faces[0], Quality{ChordTolerance: 1e-3})
	return MeshArea(mesh)
}

// pointOnStraightLoopSegment reports whether p lies within tol of one of loop's STRAIGHT (curve==nil)
// segments — the check U4-1's "no orphan seam" gate needs (derivation §3.2 caveat): a wing's far-
// tangent point must land on the OTHER host's plain (un-notched) tangent-line edge, not merely
// somewhere near the loop in aggregate, else a future edge-insert has no straight edge left to split.
func pointOnStraightLoopSegment(loop filletLoop, p math.Point3, tol float64) bool {
	n := len(loop.pts)
	for i := 0; i < n; i++ {
		if loop.curves[i] != nil {
			continue
		}
		seg := geom.NewLineSegment(loop.pts[i], loop.pts[(i+1)%n])
		if geom.DistancePointToSegment(seg, p) <= tol {
			return true
		}
	}
	return false
}

// otherNotch returns the closure notch that does NOT belong to outer's own host — the notch the wing's
// far-tangent point (built on outer's OTHER face) must weld to.
func otherNotch(dets []obstacleDetection, closure dualClosure, outer obstacleDetection) filletFace {
	if outer.host == dets[0].host {
		return closure.notchB
	}
	return closure.notchA
}

// TestBuildDualClosureU4BothNotchesAbsorbFootprint pins U4-1 item 1 (derivation §3.2): calling
// buildNotchedHost per host (REUSED unchanged) absorbs EACH host's own footprint into a single WIRE:1
// loop. The precondition that makes "absorbed" a meaningful claim (not vacuously true): each host's
// ORIGINAL face carries a separate hole loop before the rebuild (>=2 loops, outer+hole) — mergeHoleInto-
// Notch's whole job is collapsing that down to exactly 1.
func TestBuildDualClosureU4BothNotchesAbsorbFootprint(t *testing.T) {
	ef, dets, spans, maps := u4ClosureFixture(t)
	if len(dets[0].host.Loops()) < 2 || len(dets[1].host.Loops()) < 2 {
		t.Fatalf("fixture precondition: U4 hosts must carry a separate hole loop pre-rebuild, got %d/%d loops",
			len(dets[0].host.Loops()), len(dets[1].host.Loops()))
	}
	closure, ok := buildDualClosure(ef, dets, spans, maps)
	if !ok {
		t.Fatalf("buildDualClosure(U4) = ok=false, want ok=true (both notches/walls/wings should build)")
	}
	for name, notch := range map[string]filletFace{"notchA": closure.notchA, "notchB": closure.notchB} {
		if len(notch.loops) != 1 {
			t.Errorf("%s: %d loops, want 1 (WIRE:1, footprint absorbed)", name, len(notch.loops))
		}
	}
	weld := geom.ResolutionForPoints([]math.Point3{dets[0].pMinus, dets[0].pPlus}).Weld()
	if !loopHasPointNear(closure.notchA.loops[0], dets[0].pMinus, weld) || !loopHasPointNear(closure.notchA.loops[0], dets[0].pPlus, weld) {
		t.Errorf("notchA loop does not carry host A's own split nodes (P-=%v, P+=%v)", dets[0].pMinus, dets[0].pPlus)
	}
	if !loopHasPointNear(closure.notchB.loops[0], dets[1].pMinus, weld) || !loopHasPointNear(closure.notchB.loops[0], dets[1].pPlus, weld) {
		t.Errorf("notchB loop does not carry host B's own split nodes (P-=%v, P+=%v)", dets[1].pMinus, dets[1].pPlus)
	}
}

// TestBuildDualClosureU4BothWallsSplitIntact pins U4-1 item 2 (derivation §3.2): calling
// buildSplitObstacleWall per boss (REUSED unchanged) splits EACH wall's rim at its own two nodes while
// keeping the whole wall intact — its true surface area unchanged from the DRAWEXE oracle (A=502.655=
// 2π·8·10, B=799.1, derivation §0 "Survivors intact (WIRE:1)").
func TestBuildDualClosureU4BothWallsSplitIntact(t *testing.T) {
	ef, dets, spans, maps := u4ClosureFixture(t)
	closure, ok := buildDualClosure(ef, dets, spans, maps)
	if !ok {
		t.Fatalf("buildDualClosure(U4) = ok=false, want ok=true")
	}
	weldA := geom.ResolutionForPoints([]math.Point3{dets[0].pMinus, dets[0].pPlus}).Weld()
	if !loopHasPointNear(closure.wallA.loops[0], dets[0].pMinus, weldA) || !loopHasPointNear(closure.wallA.loops[0], dets[0].pPlus, weldA) {
		t.Errorf("wallA loop does not carry host A's own split nodes")
	}
	weldB := geom.ResolutionForPoints([]math.Point3{dets[1].pMinus, dets[1].pPlus}).Weld()
	if !loopHasPointNear(closure.wallB.loops[0], dets[1].pMinus, weldB) || !loopHasPointNear(closure.wallB.loops[0], dets[1].pPlus, weldB) {
		t.Errorf("wallB loop does not carry host B's own split nodes")
	}
	// 0.5% relative tolerance: TessellateFace's Quality{ChordTolerance:1e-3} facets the curved wall
	// rather than integrating its exact analytic surface, so a small faceting deficit is expected —
	// the corpus's own fingerprint gate uses a comparable relative band for tessellated areas.
	const wantA, wantB = 502.655, 799.1
	gotA := filletFaceArea(t, closure.wallA)
	t.Logf("wallA area = %.4f (want %.4f)", gotA, wantA)
	if absDiff(gotA, wantA) > 0.005*wantA {
		t.Errorf("wallA area = %.3f, want %.3f (±0.5%%)", gotA, wantA)
	}
	gotB := filletFaceArea(t, closure.wallB)
	t.Logf("wallB area = %.4f (want %.4f)", gotB, wantB)
	if absDiff(gotB, wantB) > 0.005*wantB {
		t.Errorf("wallB area = %.3f, want %.3f (±0.5%%)", gotB, wantB)
	}
}

// TestBuildDualClosureU4WingsWeldNoOrphanSeam pins U4-1 item 3 AND resolves the derivation §3.2 caveat:
// dual-host has no clean single filletWall (each host's "other" face IS the other host, notched by ITS
// OWN footprint — packDetection sets filletWall=ef.b for host A, ef.a for host B, and both ef.a/ef.b ARE
// the two hosts here), so the single-host set.wall/wallInserts seam mechanism cannot apply as-is. This
// test empirically confirms the alternative U4-1 actually builds still welds with no orphan seam: each
// wing's far-tangent point (on the OTHER host's plane) lands within weld tolerance on that other host's
// OWN notch — specifically on one of its plain (un-notched) straight tangent-line segments, since the
// wing's outer stations sit OUTSIDE the other host's own dip interval (B ⊃ A, derivation §1.1) — so a
// future edge-insert has a real straight edge there to split, not a mid-air point.
func TestBuildDualClosureU4WingsWeldNoOrphanSeam(t *testing.T) {
	ef, dets, spans, maps := u4ClosureFixture(t)
	closure, ok := buildDualClosure(ef, dets, spans, maps)
	if !ok {
		t.Fatalf("buildDualClosure(U4) = ok=false, want ok=true")
	}
	if len(closure.wings) != 2 {
		t.Fatalf("buildDualClosure(U4): %d wings, want 2", len(closure.wings))
	}
	outer, ok := outerDetection(ef, dets, spans)
	if !ok {
		t.Fatalf("outerDetection(U4): ok=false — no detection's nodes match the union's outer stations")
	}
	og, ok := computeObstacleGeom(ef, outer)
	if !ok {
		t.Fatalf("computeObstacleGeom(outer detection): ok=false")
	}
	other := otherNotch(dets, closure, outer)
	weld := geom.ResolutionForPoints([]math.Point3{outer.pMinus, outer.pPlus}).Weld()
	for idx, wantOwnNode := range [2]math.Point3{outer.pMinus, outer.pPlus} {
		cut := nodeSection(outer, og, idx)
		ownPt, farPt := cut.nodeTa, cut.nodeTb
		if wantOwnNode.DistanceTo(cut.nodeTa) > weld {
			ownPt, farPt = cut.nodeTb, cut.nodeTa // whichever endpoint IS the outer detection's own node
		}
		if d := ownPt.DistanceTo(wantOwnNode); d > weld {
			t.Errorf("node %d: own-side wing endpoint %v is %.6g from the outer detection's node %v, want <= %.6g",
				idx, ownPt, d, wantOwnNode, weld)
		}
		if !anyWingHasPointNear(closure.wings, ownPt, weld) {
			t.Errorf("node %d: no wing carries the own-side endpoint %v", idx, ownPt)
		}
		if !anyWingHasPointNear(closure.wings, farPt, weld) {
			t.Errorf("node %d: no wing carries the far-tangent endpoint %v", idx, farPt)
		}
		if !pointOnStraightLoopSegment(other.loops[0], farPt, weld) {
			t.Errorf("node %d: far-tangent endpoint %v is an ORPHAN SEAM — no straight segment of the "+
				"other host's notch passes within %.6g of it", idx, farPt, weld)
		}
	}
}

// anyWingHasPointNear reports whether p appears (within tol) in any of the wing faces' loops.
func anyWingHasPointNear(wings []filletFace, p math.Point3, tol float64) bool {
	for _, w := range wings {
		for _, l := range w.loops {
			if loopHasPointNear(l, p, tol) {
				return true
			}
		}
	}
	return false
}
