// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// u4SliverFixture returns U4's dets/spans (the do-no-harm precondition every U4-3 test shares, mirroring
// u4ClosureFixture's role for the U4-1 tests) plus the two SLIVER spans (index 0 and 2 — the B-only
// z∈[-6.633,-6.240] and z∈[6.240,6.633] bands, derivation §1.2/§4-U4-3; index 1 is the CORE span, U4-4).
func u4SliverFixture(t *testing.T) (edgeFillet, []obstacleDetection, [2]panelSpan, Resolution) {
	t.Helper()
	ef, res := u4EdgeFillet(t)
	dets, ok := detectObstacles(ef, res)
	if !ok || len(dets) != 2 {
		t.Fatalf("fixture precondition: detectObstacles(U4) = (%d, %v), want (2, true)", len(dets), ok)
	}
	spans := partitionUnionStations(dets, ef)
	if len(spans) != 3 {
		t.Fatalf("fixture precondition: partitionUnionStations(U4) = %d spans, want 3", len(spans))
	}
	return ef, dets, [2]panelSpan{spans[0], spans[2]}, res
}

// TestExtractPanelLoopU4SliverIsValence4AndClosed pins the coons4-fillable precondition (derivation
// §1.5): both sliver spans build a Valence-4, Closed RailLoop coons4Provider.Fits accepts.
func TestExtractPanelLoopU4SliverIsValence4AndClosed(t *testing.T) {
	t.Parallel()
	ef, dets, slivers, res := u4SliverFixture(t)
	for i, span := range slivers {
		loop, ok := extractPanelLoop(span, dets, ef, res)
		if !ok {
			t.Fatalf("sliver[%d] (span %+v): extractPanelLoop ok=false", i, span)
		}
		if loop.Valence() != 4 {
			t.Errorf("sliver[%d]: Valence() = %d, want 4", i, loop.Valence())
		}
		if !loop.Closed(res.Weld()) {
			t.Errorf("sliver[%d]: loop not Closed within weld %.3e", i, res.Weld())
		}
		if !(coons4Provider{}).Fits(loop) {
			t.Errorf("sliver[%d]: coons4Provider.Fits = false, want true (Valence 4)", i)
		}
	}
}

// TestExtractPanelLoopU4SliverAreaMatchesOracle is the U4-3 hard gate (derivation §4-U4-3, DRAWEXE
// result_5/result_9): each sliver's resolved CornerBlendPatch surface area matches the OCCT oracle
// 3.039 within the corpus's own relative tolerance (corpus.json "U4".deps=0.01, the SAME 1% OCCT uses
// body-wide). Area is measured by surfaceArea (corner_blend_obstacle_test.go's fine-quadrature
// integrator, the established test-only convention for a coons/obstacle patch's area, NOT
// TessellateFace/validate.MeshArea): the production tessellator (nurbsPcurveMesh) does not converge for this
// shape's extreme aspect ratio (~0.4 wide × ~7.85 long, a genuine "sliver") — probed empirically,
// validate.MeshArea at ChordTolerance 1e-2/1e-3/1e-4/1e-5 gives 1.69/6.06/6.21/13.98, never converging toward
// the oracle — a pre-existing tessellation-pipeline limitation this gated-off slice does not exercise
// live (assembleDualObstacleSet still honest-rejects, so no dual-host body is ever tessellated for
// real yet) but that MUST be fixed before U4-5 turns the gate on and renders these panels for the live
// MCP screenshot. Flagged as a concern in the U4-3 report; out of THIS slice's scope to fix.
func TestExtractPanelLoopU4SliverAreaMatchesOracle(t *testing.T) {
	t.Parallel()
	ef, dets, slivers, res := u4SliverFixture(t)
	const oracleArea = 3.039
	const corpusDeps = 0.01 // corpus.json "U4".deps
	for i, span := range slivers {
		loop, ok := extractPanelLoop(span, dets, ef, res)
		if !ok {
			t.Fatalf("sliver[%d]: extractPanelLoop ok=false", i)
		}
		patch, ok := resolveBlend(loop, res)
		if !ok {
			t.Fatalf("sliver[%d]: resolveBlend ok=false", i)
		}
		bs, isBS := patch.Surface.(geom.BSplineSurface)
		if !isBS {
			t.Fatalf("sliver[%d]: patch surface = %T, want BSplineSurface", i, patch.Surface)
		}
		area := surfaceArea(bs)
		rel := stdmath.Abs(area-oracleArea) / oracleArea
		t.Logf("sliver[%d] (span %+v): area = %.6f (oracle %.4f, rel %.4f%%)", i, span, area, oracleArea, rel*100)
		if rel > corpusDeps {
			t.Errorf("sliver[%d]: area %.6f vs oracle %.4f (rel %.4f%% > %.1f%%)", i, area, oracleArea, rel*100, corpusDeps*100)
		}
	}
}

// TestExtractPanelLoopU4SliverCornerCoincidesWithSetbackSection is the ★ Z-STATION CONSISTENCY gate
// (U4-2 review): each sliver's z=±6.240 corner pair must be BIT-IDENTICAL to setbackSection's own
// endpoints at that exact station — not merely close — so the sliver's seam rail welds exactly to the
// future core panel's own z=±6.240 rail (U4-4/U4-5). extractPanelLoop achieves this by REUSING
// setbackSection's return value directly as the loop's 4th side (buildSliverLoop), never re-deriving
// the corner independently, so this test is a receipt of that reuse, not a tolerance check.
func TestExtractPanelLoopU4SliverCornerCoincidesWithSetbackSection(t *testing.T) {
	t.Parallel()
	ef, dets, slivers, res := u4SliverFixture(t)
	for i, span := range slivers {
		activeIsB, ok := sliverActiveHost(span)
		if !ok {
			t.Fatalf("sliver[%d]: sliverActiveHost ok=false", i)
		}
		detA, detB, ok := hostDetections(dets)
		if !ok {
			t.Fatalf("sliver[%d]: hostDetections ok=false", i)
		}
		active := detA
		if activeIsB {
			active = detB
		}
		_, seamEnd, ok := sliverStations(span, active, ef)
		if !ok {
			t.Fatalf("sliver[%d]: sliverStations ok=false", i)
		}
		direct, ok := setbackSection(seamEnd, dets, ef, res)
		if !ok {
			t.Fatalf("sliver[%d]: setbackSection(%.4f) ok=false", i, seamEnd)
		}
		loop, ok := extractPanelLoop(span, dets, ef, res)
		if !ok {
			t.Fatalf("sliver[%d]: extractPanelLoop ok=false", i)
		}
		pinInactive, pinActive := direct.PointAt(0), direct.PointAt(1)
		if !loopCarriesExactPoint(loop, pinInactive) {
			t.Errorf("sliver[%d]: loop has no corner bit-identical to setbackSection's own pA %v", i, pinInactive)
		}
		if !loopCarriesExactPoint(loop, pinActive) {
			t.Errorf("sliver[%d]: loop has no corner bit-identical to setbackSection's own pB %v", i, pinActive)
		}
	}
}

// loopCarriesExactPoint reports whether some side of loop starts (or, by the loop's own Closed
// invariant, equivalently ends) at EXACTLY p — distance zero, not a weld tolerance — the bit-identity
// check the corner-coincidence gate needs.
func loopCarriesExactPoint(loop RailLoop, p math.Point3) bool {
	for _, s := range loop.Sides {
		if curveStart(s.Curve).DistanceTo(p) == 0 {
			return true
		}
	}
	return false
}

// TestExtractPanelLoopU4SliverAWeldsToNotchTangent pins the "A-side welds to the notch tangent" gate
// AT ITS WING END: the sliver's A-tangent LINE side reaches the SAME fillet-cylinder A-tangent ray
// U4-1's notch leaves plain (un-notched) beyond the union's outer stations — the exact ray
// cornerRadials derives from ef.c0 (the constant host-A radial), which is also what U4-1's
// TestBuildDualClosureU4WingsWeldNoOrphanSeam anchors its own "far-tangent" weld check to
// (fillet_obstacle_dual_test.go). Only the WING end is checked against the pure tangent formula: the
// SEAM end is setbackSection's own pinned point (already proven bit-identical to it, the previous
// test), and — verified against the OCCT oracle itself (occtSectionZNode[0] = (5.006254,-20,...), NOT
// (5,-20,...)) — sectionEndA's branch legitimately recedes off the pure tangent by a few mm exactly AT
// the node station (a genuine, oracle-matching property of the EXISTING U4-2 setbackSection, not a
// defect introduced here): the "line reaches exactly x=5 at the node" is the derivation's idealization,
// the true (and OCCT's own) section is already a hair off it there.
func TestExtractPanelLoopU4SliverAWeldsToNotchTangent(t *testing.T) {
	t.Parallel()
	ef, dets, slivers, res := u4SliverFixture(t)
	hostRadial, _, _ := cornerRadials(ef, true) // host A's own tangent radial (ef.c0.cen -> c0.ta)
	for i, span := range slivers {
		loop, ok := extractPanelLoop(span, dets, ef, res)
		if !ok {
			t.Fatalf("sliver[%d]: extractPanelLoop ok=false", i)
		}
		aSide := loop.Sides[0]
		if aSide.Cont != G1 {
			t.Errorf("sliver[%d]: A-side Cont = %v, want G1", i, aSide.Cont)
		}
		wingEnd := curveEnd(aSide.Curve) // s0 (A->B): B is the wing end
		z := axisParam(ef, wingEnd)
		want := filletAxisAt(ef, z).TranslateBy(hostRadial)
		if d := wingEnd.DistanceTo(want); d > res.Weld() {
			t.Errorf("sliver[%d]: A-side wing-end %v is %.3e off the A-tangent ray (want %v)", i, wingEnd, d, want)
		}
	}
}

// TestExtractPanelLoopU4SliverBWeldsToWingArc pins the "z=6.633(or -6.633) end welds to the wing"
// gate: the sliver's active-host node (its far end, opposite the setbackSection seam) is bit-identical
// to that host's own detection node — the SAME point buildObstacleWings' node section anchors to, so a
// future weld (U4-5) has no T-junction there.
func TestExtractPanelLoopU4SliverBWeldsToWingArc(t *testing.T) {
	t.Parallel()
	ef, dets, slivers, res := u4SliverFixture(t)
	detA, detB, ok := hostDetections(dets)
	if !ok {
		t.Fatalf("hostDetections ok=false")
	}
	for i, span := range slivers {
		activeIsB, ok := sliverActiveHost(span)
		if !ok {
			t.Fatalf("sliver[%d]: sliverActiveHost ok=false", i)
		}
		active := detA
		if activeIsB {
			active = detB
		}
		loop, ok := extractPanelLoop(span, dets, ef, res)
		if !ok {
			t.Fatalf("sliver[%d]: extractPanelLoop ok=false", i)
		}
		wantNode := active.pMinus
		if active.pMinus.DistanceTo(curveEnd(loop.Sides[1].Curve)) > active.pPlus.DistanceTo(curveEnd(loop.Sides[1].Curve)) {
			wantNode = active.pPlus
		}
		got := curveEnd(loop.Sides[1].Curve) // s1 (wing arc, B->C): C is the active host's own node
		if d := got.DistanceTo(wantNode); d > res.Weld() {
			t.Errorf("sliver[%d]: wing-arc end %v is %.3e off the active host's own node %v", i, got, d, wantNode)
		}
	}
}

// TestExtractPanelLoopBuildsCoreSpan pins the U4-3/U4-4 scope handoff (derivation §4-U4-3 vs §4-U4-4):
// a CORE span (both hosts active) now builds a valid Valence-4 loop — that shape landed in U4-4
// (fillet_obstacle_panel_core.go); the U4-4 report covers its area/fidelity gates in detail.
func TestExtractPanelLoopBuildsCoreSpan(t *testing.T) {
	t.Parallel()
	ef, dets, _, res := u4SliverFixture(t)
	spans := partitionUnionStations(dets, ef)
	core := spans[1]
	if !core.hostA || !core.hostB {
		t.Fatalf("fixture precondition: spans[1] = %+v, want the core span (hostA=hostB=true)", core)
	}
	loop, ok := extractPanelLoop(core, dets, ef, res)
	if !ok {
		t.Fatalf("extractPanelLoop(core span) ok=false, want ok=true (U4-4)")
	}
	if loop.Valence() != 4 {
		t.Errorf("extractPanelLoop(core span): Valence() = %d, want 4", loop.Valence())
	}
}

// TestPanelSideInactiveHostIsG1TangentLine pins panelSide's inactive branch (derivation §3.1): a
// straight G1 line to the host plane, exactly connecting the two points it is given.
func TestPanelSideInactiveHostIsG1TangentLine(t *testing.T) {
	t.Parallel()
	ef, dets, _, _ := u4SliverFixture(t)
	detA, _, ok := hostDetections(dets)
	if !ok {
		t.Fatalf("hostDetections ok=false")
	}
	from, to := math.P3(5, -20, 0), math.P3(5, -20, 1)
	side, ok := panelSide(ef, detA, false, from, to)
	if !ok {
		t.Fatalf("panelSide(inactive) ok=false")
	}
	if side.Cont != G1 {
		t.Errorf("inactive panelSide.Cont = %v, want G1", side.Cont)
	}
	if _, isLine := side.Curve.(geom.LineSegment); !isLine {
		t.Errorf("inactive panelSide.Curve = %T, want geom.LineSegment", side.Curve)
	}
	if _, isPlane := side.Adjacent.(geom.Plane); !isPlane {
		t.Errorf("inactive panelSide.Adjacent = %T, want geom.Plane (the host plane)", side.Adjacent)
	}
	if d := side.Curve.PointAt(0).DistanceTo(from); d != 0 {
		t.Errorf("inactive panelSide start drifted %.3e from the pinned endpoint", d)
	}
	if d := side.Curve.PointAt(1).DistanceTo(to); d != 0 {
		t.Errorf("inactive panelSide end drifted %.3e from the pinned endpoint", d)
	}
}

// TestPanelSideActiveHostIsG0RimSubArc pins panelSide's active branch: a G0 rim sub-arc (never G1 —
// the base-rim crease, the T6 fold lesson obstacleSides already documents) pinned exactly to from/to.
func TestPanelSideActiveHostIsG0RimSubArc(t *testing.T) {
	t.Parallel()
	ef, dets, _, _ := u4SliverFixture(t)
	_, detB, ok := hostDetections(dets)
	if !ok {
		t.Fatalf("hostDetections ok=false")
	}
	from := detB.pPlus // host B's own node (z=6.633)
	to, ok := dipRimPointAtStation(detB, ef, axisParam(ef, detB.pPlus)-0.1)
	if !ok {
		t.Fatalf("dipRimPointAtStation precondition failed")
	}
	side, ok := panelSide(ef, detB, true, from, to)
	if !ok {
		t.Fatalf("panelSide(active) ok=false")
	}
	if side.Cont != G0 {
		t.Errorf("active panelSide.Cont = %v, want G0", side.Cont)
	}
	bs, isBS := side.Curve.(geom.BSplineCurve)
	if !isBS {
		t.Errorf("active panelSide.Curve = %T, want geom.BSplineCurve (a fitted rim sub-arc)", side.Curve)
		return
	}
	// The pin lives on the CLAMPED END POLES, and it is exact: panelRimSubArc hands from/to to the
	// least-squares fit as its first and last data points and geom.NewApproximatedBSplineCurve clamps
	// them, so Ctrl[0]/Ctrl[n−1] ARE those points bit-for-bit. Asserting through PointAt instead would
	// gate the fitted pin on the RATIONAL BASIS RECONSTRUCTION at the domain ends — Σ N_i(t)·w_i·P_i
	// renormalised — which is a weighted sum whose rounding depends on the coordinates' magnitudes, not
	// on whether the curve is pinned. It read exactly 0 while host B's node sat on the 64-chord sampled
	// rim and reads 1.986e-15 (one ulp at |P| ≈ 19) now that the node is solved on the rim CURVE
	// (analyticNode); the pin itself never moved. So the pole equality below is the invariant, asserted
	// exactly, and the evaluation is gated separately at panelPinEvalFloor.
	last := len(bs.Ctrl) - 1
	if d := bs.Ctrl[0].DistanceTo(from); d != 0 {
		t.Errorf("active panelSide start pole drifted %.3e from the pinned endpoint", d)
	}
	if d := bs.Ctrl[last].DistanceTo(to); d != 0 {
		t.Errorf("active panelSide end pole drifted %.3e from the pinned endpoint", d)
	}
	_, hi := side.Curve.Domain()
	if d := side.Curve.PointAt(0).DistanceTo(from); d > panelPinEvalFloor {
		t.Errorf("active panelSide start evaluates %.3e from its own clamped pole, above the reconstruction floor", d)
	}
	if d := side.Curve.PointAt(hi).DistanceTo(to); d > panelPinEvalFloor {
		t.Errorf("active panelSide end evaluates %.3e from its own clamped pole, above the reconstruction floor", d)
	}
}

// panelPinEvalFloor bounds the rational-basis reconstruction error when a clamped B-spline is
// EVALUATED at a domain end, as distinct from the pin itself (asserted exactly on the poles above).
// Measured 1.986e-15 on this fixture, whose points are ~19 from the origin — one ulp; 1e-12 leaves
// ~500x margin and is still ~1e5 tighter than the model weld, so a real endpoint drift cannot hide here.
const panelPinEvalFloor = 1e-12
