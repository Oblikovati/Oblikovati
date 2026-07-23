// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"os"
	"testing"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step"
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

// u4EdgeFillet solves the edgeFillet for U4's filleted edge (r=5) the same way FilletEdges does
// internally (resolveFilletPicks → computeCorners → computeFillets), so the returned ef/Resolution
// pair is exactly what obstacleFacesFor would see mid-rebuild — the qualifying==2 dual-host case this
// slice wires up (derivation §0/§1.1).
func u4EdgeFillet(t *testing.T) (edgeFillet, Resolution) {
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
	return fils[0], ResolutionForBody(body)
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

// TestObstacleFacesForU4DualHostStubRejectsToBaseline pins the U4-0 wiring end to end (derivation §3.3,
// item 3): a qualifying==2 edge now ROUTES through dualObstacleRoute/assembleDualObstacleSet instead of
// silently vanishing at detectObstacle's own gate, but the U4-0 stub still answers ok=false, so the
// fillet feature falls to the SAME do-no-harm baseline it did before this change — U4 stays at
// HolesContained=false (the #2007 dual-host defect, unresolved until U4-1..U4-5), not a new outcome.
func TestObstacleFacesForU4DualHostStubRejectsToBaseline(t *testing.T) {
	body := importU4(t)
	edge := edgeAtMidpoint(body, u4EdgeMidpoint)
	if edge == nil {
		t.Fatalf("U4 fixture: filleted edge (midpoint %v) not found", u4EdgeMidpoint)
	}
	res, err := FilletEdges(body, [][]byte{edge.ReferenceKey()}, 5)
	if err != nil {
		t.Fatalf("FilletEdges(U4): %v (the do-no-harm baseline must still build a body)", err)
	}
	rep := Validate(res)
	if rep.HolesContained {
		t.Fatalf("U4: HolesContained=true — the dual-host stub must still land on the pre-existing " +
			"do-no-harm baseline (#2007); did assembleDualObstacleSet start admitting the edge?")
	}
}

// absDiff avoids importing math.Abs's stdmath alias for a single call site.
func absDiff(a, b float64) float64 {
	if a < b {
		return b - a
	}
	return a - b
}
