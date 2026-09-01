// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// u4CoreFixture returns U4's dets/spans plus the ONE dual-host CORE span (index 1 — the z∈[-6.240,
// +6.240] band both hosts dip into, derivation §1.2/§4-U4-4; indices 0/2 are the sliver spans, U4-3).
func u4CoreFixture(t *testing.T) (edgeFillet, []obstacleDetection, panelSpan, Resolution) {
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
	core := spans[1]
	if !core.hostA || !core.hostB {
		t.Fatalf("fixture precondition: spans[1] = %+v, want the core span (hostA=hostB=true)", core)
	}
	return ef, dets, core, res
}

// coreOracleArea is the DRAWEXE oracle for ONE split core panel (derivation §0, result_13≡result_14);
// the unsplit whole-span panel's oracle is exactly double (the two halves sum to the whole band).
const coreOracleArea = 30.334
const coreOracleAreaUnsplit = 2 * coreOracleArea // 60.668

// corpusAreaTol is corpus.json "U4".deps — the SAME 1% relative tolerance the full-body corpus gate
// uses (mirrors panel_sliver_test.go's corpusDeps).
const corpusAreaTol = 0.01

// TestExtractPanelLoopU4CoreIsValence4AndClosed pins the coons4-fillable precondition (derivation
// §1.5) for BOTH core constructions this slice builds: the whole-span (unsplit) panel and each
// z=0-split half — every one is a Valence-4, Closed RailLoop coons4Provider.Fits accepts.
func TestExtractPanelLoopU4CoreIsValence4AndClosed(t *testing.T) {
	t.Parallel()
	ef, dets, core, res := u4CoreFixture(t)
	assertLoopFits(t, "unsplit", ef, dets, core, res)
	halves, ok := splitCoreSpan(core)
	if !ok {
		t.Fatalf("splitCoreSpan ok=false")
	}
	for _, h := range halves {
		assertLoopFits(t, "split half", ef, dets, h, res)
	}
}

// assertLoopFits builds span's loop and asserts it is a coons4-fillable Valence-4 closed loop.
func assertLoopFits(t *testing.T, label string, ef edgeFillet, dets []obstacleDetection, span panelSpan, res Resolution) {
	t.Helper()
	loop, ok := extractPanelLoop(span, dets, ef, res)
	if !ok {
		t.Fatalf("%s (span %+v): extractPanelLoop ok=false", label, span)
	}
	if loop.Valence() != 4 {
		t.Errorf("%s: Valence() = %d, want 4", label, loop.Valence())
	}
	if !loop.Closed(res.Weld()) {
		t.Errorf("%s: loop not Closed within weld %.3e", label, res.Weld())
	}
	if !(coons4Provider{}).Fits(loop) {
		t.Errorf("%s: coons4Provider.Fits = false, want true (Valence 4)", label)
	}
}

// TestSplitCoreSpanProducesTwoStraddlingHalves pins the z=0 split lever's structural shape (derivation
// §2.3): both halves stay CORE spans (hostA=hostB=true), meet exactly at z=0, and together reconstruct
// the original span's [zLo,zHi] extent with no gap or overlap.
func TestSplitCoreSpanProducesTwoStraddlingHalves(t *testing.T) {
	t.Parallel()
	_, _, core, _ := u4CoreFixture(t)
	halves, ok := splitCoreSpan(core)
	if !ok {
		t.Fatalf("splitCoreSpan ok=false")
	}
	lo, hi := halves[0], halves[1]
	if !lo.hostA || !lo.hostB || !hi.hostA || !hi.hostB {
		t.Errorf("halves = %+v / %+v, want both hostA=hostB=true", lo, hi)
	}
	if lo.zLo != core.zLo || lo.zHi != 0 || hi.zLo != 0 || hi.zHi != core.zHi {
		t.Errorf("halves = %+v / %+v, want [%.4f,0] and [0,%.4f]", lo, hi, core.zLo, core.zHi)
	}
}

// TestExtractPanelLoopU4CoreWeldsToSliverSeam is the ★ z-STATION CONSISTENCY gate (derivation §1.4:
// "shared with the sliver"): the core panel's z=6.240 corner pair (from the UNSPLIT construction, the
// same station a sliver panel ends at) must be BIT-IDENTICAL to setbackSection's own endpoints at that
// exact station — the same reuse-not-rederive discipline TestExtractPanelLoopU4SliverCornerCoincides-
// WithSetbackSection already pins for the sliver side of this same seam, so the two neighbours weld
// with no drift (ADR-0042, the sliver-core weld invariant U4-2's docstring calls out by name).
func TestExtractPanelLoopU4CoreWeldsToSliverSeam(t *testing.T) {
	t.Parallel()
	ef, dets, core, res := u4CoreFixture(t)
	direct, ok := setbackSection(core.zHi, dets, ef, res)
	if !ok {
		t.Fatalf("setbackSection(%.4f) ok=false", core.zHi)
	}
	loop, ok := extractPanelLoop(core, dets, ef, res)
	if !ok {
		t.Fatalf("extractPanelLoop(core) ok=false")
	}
	pA, pB := direct.PointAt(0), direct.PointAt(1)
	if !loopCarriesExactPoint(loop, pA) {
		t.Errorf("core loop has no corner bit-identical to setbackSection's own pA %v", pA)
	}
	if !loopCarriesExactPoint(loop, pB) {
		t.Errorf("core loop has no corner bit-identical to setbackSection's own pB %v", pB)
	}
	// Cross-check against the sliver's OWN corner at the identical station (the actual weld partner):
	// extractPanelLoop on the REAL neighbouring sliver span (partitionUnionStations' own spans[2], not
	// a synthetic literal) must carry the SAME two points.
	spans := partitionUnionStations(dets, ef)
	sliverLoop, ok := extractPanelLoop(spans[2], dets, ef, res)
	if !ok {
		t.Fatalf("extractPanelLoop(neighbour sliver, spans[2]) ok=false")
	}
	if !loopCarriesExactPoint(sliverLoop, pA) || !loopCarriesExactPoint(sliverLoop, pB) {
		t.Errorf("neighbour sliver loop does not carry the SAME seam corners as the core loop — weld would open")
	}
}

// TestExtractPanelLoopU4CoreSplitHalvesShareMidSeam pins the internal z=0 corner (derivation §2.3):
// the two split halves' shared mid corner is bit-identical between them — both are the SAME
// setbackSection(0) call's endpoints, reused verbatim by each half (never re-derived), so the internal
// seam this slice's own split introduces is itself weld-exact.
func TestExtractPanelLoopU4CoreSplitHalvesShareMidSeam(t *testing.T) {
	t.Parallel()
	ef, dets, core, res := u4CoreFixture(t)
	halves, ok := splitCoreSpan(core)
	if !ok {
		t.Fatalf("splitCoreSpan ok=false")
	}
	mid, ok := setbackSection(0, dets, ef, res)
	if !ok {
		t.Fatalf("setbackSection(0) ok=false")
	}
	pA, pB := mid.PointAt(0), mid.PointAt(1)
	for i, h := range halves {
		loop, ok := extractPanelLoop(h, dets, ef, res)
		if !ok {
			t.Fatalf("half[%d]: extractPanelLoop ok=false", i)
		}
		if !loopCarriesExactPoint(loop, pA) || !loopCarriesExactPoint(loop, pB) {
			t.Errorf("half[%d]: loop does not carry setbackSection(0)'s own corners bit-identically", i)
		}
	}
}

// TestExtractPanelLoopU4CoreUnsplitCanalAreaFaithful is the U4-4b receipt that the FAITHFUL fill (the
// exact-station canal loft, not U4-4's coons4) closes the gap even UNSPLIT: the whole-span |z|<6.240
// panel now lands WITHIN corpus.json "U4".deps=1% of the true 60.668 — a decisive improvement over
// coons4's ~76% unsplit balloon (U4-4 report). The core surface is a rolling-ball CANAL (U4-2), so
// skinning it through the exact radius-5 stations is the RIGHT surface; the only error is the
// v-interpolation between exact stations, which converges fast. The SHIPPED construction is still the
// z=0 SPLIT (below) — chosen for the core-core seam weld, not because unsplit misses — but this proves
// the split is no longer a numerical necessity, only a seam-architecture choice.
func TestExtractPanelLoopU4CoreUnsplitCanalAreaFaithful(t *testing.T) {
	t.Parallel()
	ef, dets, core, res := u4CoreFixture(t)
	loop, ok := extractPanelLoop(core, dets, ef, res)
	if !ok {
		t.Fatalf("extractPanelLoop(core, unsplit) ok=false")
	}
	if loop.Stations == nil {
		t.Fatal("unsplit core loop carries no canal station payload — the coons4 fill was not replaced")
	}
	area := measureIntegratorArea(t, loop, res)
	rel := stdmath.Abs(area-coreOracleAreaUnsplit) / coreOracleAreaUnsplit
	t.Logf("unsplit canal-loft core area = %.6f (oracle %.4f, rel %.4f%%)", area, coreOracleAreaUnsplit, rel*100)
	if rel > corpusAreaTol {
		t.Errorf("unsplit canal-loft core area %.6f vs oracle %.4f (rel %.4f%% > corpus tol %.1f%%)",
			area, coreOracleAreaUnsplit, rel*100, corpusAreaTol*100)
	}
}

// TestExtractPanelLoopU4CoreSplitAreaFaithful is the U4-4b hard-gate area check for the SHIPPED
// construction (the z=0 split): each half's resolved CornerBlendPatch — now the exact-station canal
// loft (BlendKindCanalStation), NOT coons4 — is measured via surfaceArea (the U4-3 test-only
// integrator convention) and pinned WITHIN corpus.json "U4".deps=1% of the oracle (30.334). The
// measured residual is ~0.006% (K=9 stations, past the convergence knee), ~700x inside the gate and a
// decisive improvement over coons4's 4.5% all-G0 miss (the whole reason U4-4b replaced the fill).
func TestExtractPanelLoopU4CoreSplitAreaFaithful(t *testing.T) {
	t.Parallel()
	ef, dets, core, res := u4CoreFixture(t)
	halves, ok := splitCoreSpan(core)
	if !ok {
		t.Fatalf("splitCoreSpan ok=false")
	}
	for i, h := range halves {
		loop, ok := extractPanelLoop(h, dets, ef, res)
		if !ok {
			t.Fatalf("half[%d]: extractPanelLoop ok=false", i)
		}
		if loop.Stations == nil {
			t.Fatalf("half[%d]: core loop carries no canal station payload — the coons4 fill was not replaced", i)
		}
		area := measureIntegratorArea(t, loop, res)
		rel := stdmath.Abs(area-coreOracleArea) / coreOracleArea
		t.Logf("split half[%d] (span %+v): canal-loft area = %.6f (oracle %.4f, rel %.4f%%)", i, h, area, coreOracleArea, rel*100)
		if rel > corpusAreaTol {
			t.Errorf("half[%d]: canal-loft area %.6f vs oracle %.4f (rel %.4f%% > corpus tol %.1f%%)",
				i, area, coreOracleArea, rel*100, corpusAreaTol*100)
		}
	}
}

// TestExtractPanelLoopU4CoreResolvesToCanalStation pins that the CORE fill is the exact-station canal
// loft, not coons4: resolveBlend routes the Stations-marked loop to canalStationProvider (ahead of
// coons4 in blendTiers), so the emitted patch carries BlendKindCanalStation. The sliver loops (no
// Stations payload) still route to coons4 — the do-no-harm invariant that keeps the corpus untouched.
func TestExtractPanelLoopU4CoreResolvesToCanalStation(t *testing.T) {
	t.Parallel()
	ef, dets, core, res := u4CoreFixture(t)
	halves, ok := splitCoreSpan(core)
	if !ok {
		t.Fatalf("splitCoreSpan ok=false")
	}
	loop, ok := extractPanelLoop(halves[0], dets, ef, res)
	if !ok {
		t.Fatalf("extractPanelLoop(half[0]) ok=false")
	}
	patch, ok := resolveBlend(loop, res)
	if !ok {
		t.Fatalf("resolveBlend ok=false")
	}
	if patch.Kind != BlendKindCanalStation {
		t.Errorf("core patch Kind = %q, want %q (the faithful canal fill, not coons4)", patch.Kind, BlendKindCanalStation)
	}
	// A sliver span (single host) must still resolve to coons4 — U4-4b touches only the core fill.
	spans := partitionUnionStations(dets, ef)
	sliverLoop, ok := extractPanelLoop(spans[0], dets, ef, res)
	if !ok {
		t.Fatalf("extractPanelLoop(sliver spans[0]) ok=false")
	}
	if sliverLoop.Stations != nil {
		t.Error("sliver loop carries a canal station payload — U4-4b must leave the sliver fill unchanged")
	}
	sliverPatch, ok := resolveBlend(sliverLoop, res)
	if !ok {
		t.Fatalf("resolveBlend(sliver) ok=false")
	}
	if sliverPatch.Kind != BlendKindCoons4 {
		t.Errorf("sliver patch Kind = %q, want %q (unchanged coons4 fill)", sliverPatch.Kind, BlendKindCoons4)
	}
}

// TestExtractPanelLoopU4CoreCanalSeamBitIdentical is the ★ U4-4b seam-consistency gate: the canal
// loft's v-boundary corners must be BIT-IDENTICAL to setbackSection's own endpoints at those stations
// — the sliver-core weld at z=zLo (=-6.240) and the core-core weld at z=0 (each split half's zHi). The
// loft's four corners are the exact station feet (coreSectionStation reads setbackSection's own
// PointAt(0)/PointAt(1)), and a clamped interpolating spline pins its endpoint controls to those feet,
// so surf.PointAt at each domain corner reproduces the setbackSection endpoint to the last bit. This is
// what makes the canal core panel weld to its sliver/core neighbours with zero drift (ADR-0042).
func TestExtractPanelLoopU4CoreCanalSeamBitIdentical(t *testing.T) {
	t.Parallel()
	ef, dets, core, res := u4CoreFixture(t)
	halves, ok := splitCoreSpan(core)
	if !ok {
		t.Fatalf("splitCoreSpan ok=false")
	}
	loop, ok := extractPanelLoop(halves[0], dets, ef, res) // span [zLo, 0]: v0->z=zLo seam, v1->z=0 seam
	if !ok {
		t.Fatalf("extractPanelLoop(half[0]) ok=false")
	}
	patch, ok := resolveBlend(loop, res)
	if !ok {
		t.Fatalf("resolveBlend ok=false")
	}
	surf, isBS := patch.Surface.(geom.BSplineSurface)
	if !isBS {
		t.Fatalf("patch surface = %T, want BSplineSurface", patch.Surface)
	}
	u0, u1 := surf.UDomain()
	v0, v1 := surf.VDomain()
	assertSeamCorner(t, "z=zLo footA", surf.PointAt(u0, v0), sectionEndpoint(t, halves[0].zLo, dets, ef, res, 0))
	assertSeamCorner(t, "z=zLo footB", surf.PointAt(u1, v0), sectionEndpoint(t, halves[0].zLo, dets, ef, res, 1))
	assertSeamCorner(t, "z=0 footA", surf.PointAt(u0, v1), sectionEndpoint(t, 0, dets, ef, res, 0))
	assertSeamCorner(t, "z=0 footB", surf.PointAt(u1, v1), sectionEndpoint(t, 0, dets, ef, res, 1))
}

// sectionEndpoint returns setbackSection(z)'s own endpoint (end=0 → PointAt(0), end=1 → PointAt(1)) —
// the exact rim foot the canal loft's boundary must reproduce bit-for-bit.
func sectionEndpoint(t *testing.T, z float64, dets []obstacleDetection, ef edgeFillet, res Resolution, end float64) math.Point3 {
	t.Helper()
	sec, ok := setbackSection(z, dets, ef, res)
	if !ok {
		t.Fatalf("setbackSection(%.4f) ok=false", z)
	}
	return sec.PointAt(end)
}

// assertSeamCorner asserts the canal-loft corner is bit-identical to the setbackSection endpoint.
func assertSeamCorner(t *testing.T, label string, got, want math.Point3) {
	t.Helper()
	if got != want {
		t.Errorf("%s: loft corner %v is not bit-identical to setbackSection endpoint %v (Δ=%.3e)",
			label, got, want, float64(got.DistanceTo(want)))
	}
}

// TestExtractPanelLoopU4CoreCanalFootAtRadius witnesses LoftCanalStations' own fidelity gate: every
// supplied station foot sits exactly ef.cyl.Radius (=5) from its centre. The loft asserts this
// internally (a mis-supplied foot is declined, not lofted), so a successful resolveBlend already proves
// it — this test re-measures the payload directly to make the rolling-ball invariant an explicit,
// diagnosable receipt (the ball touches both boss rims at every station).
func TestExtractPanelLoopU4CoreCanalFootAtRadius(t *testing.T) {
	t.Parallel()
	ef, dets, core, res := u4CoreFixture(t)
	loop, ok := extractPanelLoop(core, dets, ef, res)
	if !ok {
		t.Fatalf("extractPanelLoop(core) ok=false")
	}
	sf := loop.Stations
	if sf == nil {
		t.Fatal("core loop carries no canal station payload")
	}
	for i := range sf.Centers {
		dA := stdmath.Abs(float64(sf.FeetA[i].DistanceTo(sf.Centers[i])) - sf.Radius)
		dB := stdmath.Abs(float64(sf.FeetB[i].DistanceTo(sf.Centers[i])) - sf.Radius)
		if dA > res.Weld() || dB > res.Weld() {
			t.Errorf("station %d: footA %.3e / footB %.3e off radius %.4f (weld %.3e)", i, dA, dB, sf.Radius, res.Weld())
		}
	}
}

// measureIntegratorArea resolves loop through coons4 and returns its area via surfaceArea (the
// fine-quadrature test-only integrator, corner_blend_obstacle_test.go — the SAME convention U4-3 uses,
// not TessellateFace/validate.MeshArea).
func measureIntegratorArea(t *testing.T, loop RailLoop, res Resolution) float64 {
	t.Helper()
	patch, ok := resolveBlend(loop, res)
	if !ok {
		t.Fatalf("resolveBlend ok=false")
	}
	bs, isBS := patch.Surface.(geom.BSplineSurface)
	if !isBS {
		t.Fatalf("patch surface = %T, want BSplineSurface", patch.Surface)
	}
	return surfaceArea(bs)
}

// TestExtractPanelLoopU4CoreSplitProductionTessConverges is the ★ partial real-U4 validation of the
// #2009 starved-rail tessellator fix the brief asks for: UNLIKE the sliver panel (U4-3's own docstring:
// nurbsPcurveMesh never converges for its ~0.4×7.85 extreme-aspect shape), a split core panel's aspect
// ratio is not extreme, so it IS buildable as a standalone topo.Face and nurbsPcurveMesh converges,
// fold-free, to the SAME area the test-only integrator measures (cross-validating both measurement
// paths independently) — evidence the #2009 fix generalizes beyond its own synthetic bunched-cylinder
// regression to a real corner-blend panel shape, even though this whole dual-host body is still
// gated off (never tessellated live until U4-5).
func TestExtractPanelLoopU4CoreSplitProductionTessConverges(t *testing.T) {
	t.Parallel()
	ef, dets, core, res := u4CoreFixture(t)
	halves, ok := splitCoreSpan(core)
	if !ok {
		t.Fatalf("splitCoreSpan ok=false")
	}
	loop, ok := extractPanelLoop(halves[0], dets, ef, res)
	if !ok {
		t.Fatalf("extractPanelLoop(half[0]) ok=false")
	}
	patch, ok := resolveBlend(loop, res)
	if !ok {
		t.Fatalf("resolveBlend ok=false")
	}
	integratorArea := measureIntegratorArea(t, loop, res)
	face := railLoopToTopoFace(t, loop, patch.Surface)

	prevErr := stdmath.Inf(1)
	for _, tol := range []float64{0.05, 0.01, 0.001, 0.0001} {
		q := Quality{ChordTolerance: tol, AngleTolerance: DefaultQuality().AngleTolerance}
		m := nurbsPcurveMesh(face, q)
		if m == nil {
			t.Fatalf("nurbsPcurveMesh declined the core split-half panel at tol=%g", tol)
		}
		area := validate.MeshArea(m)
		rel := stdmath.Abs(area-integratorArea) / integratorArea
		t.Logf("tol=%g: nurbsPcurveMesh area=%.6f (integrator=%.6f, rel=%.4f%%) folds=%d",
			tol, area, integratorArea, rel*100, validate.FoldEdgeCount(m))
		if folds := validate.FoldEdgeCount(m); folds != 0 {
			t.Errorf("tol=%g: %d fold edges, want 0", tol, folds)
		}
		if rel > 0.02 {
			t.Errorf("tol=%g: nurbsPcurveMesh area %.6f vs integrator %.6f (rel %.4f%% > 2%%) — "+
				"production tess should converge to the same fill the integrator measures", tol, area, integratorArea, rel*100)
		}
		if rel > prevErr+0.005 {
			t.Errorf("tol=%g: rel error %.5f increased from previous %.5f by more than FP slack — not converging", tol, rel, prevErr)
		}
		prevErr = rel
	}
}

// railLoopToTopoFace wraps a resolved RailLoop's boundary directly into a standalone single-face
// topo.Body (mirroring starved_rail_test.go's cylindricalStripFace pattern for a synthetic panel):
// each side's own Curve becomes an edge between its own corner vertices, in loop order (RailLoop.Closed
// already guarantees side i's end meets side i+1's start), so the loop needs no re-derivation — this
// is a test-only adapter for driving the PRODUCTION tessellator (nurbsPcurveMesh), not a production
// assembly path (the real one, buildPatchFace, is single-host-obstacle-specific and does not generalize
// to a bare RailLoop; the dual-host assembler lands in U4-5).
func railLoopToTopoFace(t *testing.T, loop RailLoop, surf geom.Surface) *topo.Face {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("test", "core-probe", 0)))
	n := loop.Valence()
	verts := make([]*topo.Vertex, n)
	for i, s := range loop.Sides {
		verts[i] = bld.AddVertex(curveStart(s.Curve), topo.NewLineage(topo.Tok("test", "v", i)))
	}
	uses := make([]topo.Use, n)
	for i, s := range loop.Sides {
		e := bld.AddEdge(s.Curve, verts[i], verts[(i+1)%n], topo.NewLineage(topo.Tok("test", "e", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok("test", "face", 0)), topo.OuterLoop(uses...))
	return bld.Build().Faces()[0]
}
