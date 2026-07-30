// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// W3b unified canal HOST bite. Every assertion drives the REAL N7 fixture (n7CornerFill →
// extractCurvedCorner → resolveBlend), never a fabricated patch. The load-bearing deliverables:
//   - each corner host's INNER BITE composes from the arms' W2 host rails (collected, shared-edge
//     identity — NOT rebuilt at a shared centre) +/- the host foot-locus, chained by shared endpoints
//     (junctions meet ~1e-14, feet[0]/feet[1] point-identical to the rails' inner ends → residual 0);
//   - a plane host (2 rails, no bridge, rails meeting at a point) CLOSES to a valid retrimmed face;
//   - the risk #1 anchor evidence — the WALL bite's arm-rail OUTER ends vs the wall bitten loop: s_4
//     anchors at ~0, the s_5 TORUS rail's far end (azimuth 0) overshoots the FIXTURE band by ~34u, so
//     canalHostBite HONEST-DECLINES with the measured gap (a fixture host-extent limit, not a mis-centred
//     rail — the junctions meet exactly); and
//   - a MIS-TAGGED foot-locus (feet[0]↔feet[1] swapped) is caught by the chain gate, never papered over.

// n7CanalBiteInputs resolves the real N7 corner into the W3b bite inputs: the per-arm rail bundles, the
// tagged boundary isocurves, and the CanalCorner.Rolls payload (rolls[0]=wall, rolls[1]=s_10 surface).
func n7CanalBiteInputs(t *testing.T, w cornerWeld, arms []edgeFillet, res Resolution) ([]canalArmBundle, canalBoundaries, []geom.Surface) {
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
	bundles, ok := canalArmBundles(arms, w, centres, scale, res)
	if !ok {
		t.Fatalf("canalArmBundles must build the N7 per-arm rail bundles")
	}
	return bundles, boundaries, loop.Canal.Rolls
}

// TestArmRailsOnHost_CollectsPerHost is the W3b collection gate: armRailsOnHost gathers exactly the arms
// that roll on each host (2 on the wall / each plane), reading each arm's ALREADY-built rail from its
// bundle (shared-edge identity), and the wall's two rails' inner ends are point-identical to the wall
// foot-locus feet[0]'s endpoints (the W0/W1 chain junctions, residual 0).
func TestArmRailsOnHost_CollectsPerHost(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	bundles, boundaries, _ := n7CanalBiteInputs(t, w, arms, res)
	wall, fx50, fz10 := arms[0].a, arms[0].b, arms[1].b
	for name, host := range map[string]*topo.Face{"wall": wall, "planeB(fx50)": fx50, "planeA(fz10)": fz10} {
		if n := len(armRailsOnHost(host, bundles)); n != 2 {
			t.Errorf("%s: armRailsOnHost = %d rails, want 2", name, n)
		}
	}
	// The two wall rails' inner (tHost) ends MEET feet[0]'s endpoints W0/W1 — the chain junctions the wall
	// bite closes on (they coincide to ~1e-14; the bit-exact identity is the bridge curve feet[0] itself,
	// asserted in TestCanalInnerBite_WallChainsRailsFootRails).
	f0 := boundaries.feet[0]
	lo, hi := f0.Domain()
	rails := armRailsOnHost(wall, bundles)
	got := []math.Point3{rails[0].to, rails[1].to}
	junction := stdmath.Min(pairResidual(got, f0.PointAt(lo), f0.PointAt(hi)), pairResidual(got, f0.PointAt(hi), f0.PointAt(lo)))
	tol := res.Weld() * w.radius
	t.Logf("wall rails' inner ends ↔ feet[0] endpoints junction gap = %.3e (tol %.3e)", junction, tol)
	if junction > tol {
		t.Fatalf("wall rails' inner ends do not meet feet[0]; junction gap %.3e > tol %.3e", junction, tol)
	}
}

// TestFootLocusForHost_TagsByRolls proves the foot-locus↔host mapping is by CanalCorner.Rolls IDENTITY,
// not geometry-guessing (risk #3): the wall gets feet[0], the s_10 boss gets feet[1], and the two planes
// (on neither roll surface) get NO bridge.
func TestFootLocusForHost_TagsByRolls(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	bundles, boundaries, rolls := n7CanalBiteInputs(t, w, arms, res)
	_ = bundles
	tol := res.Weld() * w.radius
	wall, fx50, fz10 := arms[0].a, arms[0].b, arms[1].b

	assertBridgeIs(t, wall, boundaries, rolls, tol, boundaries.feet[0], "wall→feet[0]")
	assertNoBridge(t, fx50, boundaries, rolls, tol, "planeB(fx50)")
	assertNoBridge(t, fz10, boundaries, rolls, tol, "planeA(fz10)")

	boss := n7BossHost(t)
	assertBridgeIs(t, boss, boundaries, rolls, tol, boundaries.feet[1], "s_10 boss→feet[1]")
}

// TestCanalInnerBite_WallChainsRailsFootRails is the W3b/F3 core gate + the F3 shared-edge vertex-identity
// check: the wall inner bite composes [s_4 wall rail, feet[0] as ringSegSamples sub-chords, s_5 wall rail]
// and chains them by shared endpoints — every junction meeting within tol. F3 replaced the single
// whole-curve feet[0] bridge (which shared only its two endpoints with the corner patch and cracked the 5
// interior vertices) with the SAME sampleCurve3Open sub-chords the corner patch tiles feet[0] into, so the
// retrimmed wall welds point-for-point to the corner patch. This asserts the foot sub-chords are exactly
// the patch's sampling (residual 0, equal count) — the watertightness crux.
func TestCanalInnerBite_WallChainsRailsFootRails(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	bundles, boundaries, rolls := n7CanalBiteInputs(t, w, arms, res)
	tol := res.Weld() * w.radius
	inner, reason := canalInnerBite(arms[0].a, bundles, boundaries, rolls, tol)
	if reason != "" {
		t.Fatalf("wall inner bite declined: %s", reason)
	}
	// 2 wall rails (s_4, s_5) + ringSegSamples foot-locus sub-chords (feet[0]).
	if want := 2 + ringSegSamples; len(inner) != want {
		t.Fatalf("wall inner bite = %d pieces, want %d (2 rails + %d feet[0] sub-chords)", len(inner), want, ringSegSamples)
	}
	assertChainMeets(t, inner, tol, "wall")
	assertFootChordsMatchPatch(t, inner, boundaries.feet[0], boundaries.feetRev[0])
}

// assertFootChordsMatchPatch proves every foot-locus sub-chord vertex in the inner bite is BIT-IDENTICAL to
// the corner patch's own feet[0] sampling (sampleCurve3Open with the patch rev) — the F3 shared-edge
// vertex-sequence identity that welds the retrimmed wall to the corner patch without a crack (residual 0).
func assertFootChordsMatchPatch(t *testing.T, inner []endSeg, foot geom.Curve3, rev bool) {
	t.Helper()
	var verts []math.Point3 // every inner-bite vertex (rails + foot sub-chords); geom.Curve3 is uncomparable
	for _, s := range inner {
		verts = append(verts, s.from, s.to)
	}
	for _, p := range sampleCurve3Open(foot, rev) { // the corner patch's exact feet[0] samples
		best := 1.0
		for _, q := range verts {
			if d := float64(p.DistanceTo(q)); d < best {
				best = d
			}
		}
		if best != 0 {
			t.Fatalf("corner-patch feet[0] sample %v has no bit-identical wall foot-chord vertex (nearest residual %.3e, want 0)", p, best)
		}
	}
}

// TestCanalInnerBite_PlanesMeetAtPoint proves the two-rail plane bites (no bridge) chain by the two arm
// rails MEETING AT A POINT (P0 on plane B, P1 on plane A) — the (2,-) composition the same assembler
// handles with no special case.
func TestCanalInnerBite_PlanesMeetAtPoint(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	bundles, boundaries, rolls := n7CanalBiteInputs(t, w, arms, res)
	tol := res.Weld() * w.radius
	for name, host := range map[string]*topo.Face{"planeB(fx50)": arms[0].b, "planeA(fz10)": arms[1].b} {
		inner, reason := canalInnerBite(host, bundles, boundaries, rolls, tol)
		if reason != "" {
			t.Fatalf("%s inner bite declined: %s", name, reason)
		}
		if len(inner) != 2 {
			t.Fatalf("%s inner bite = %d pieces, want 2 (two arm rails, no bridge)", name, len(inner))
		}
		gap := float64(inner[0].to.DistanceTo(inner[1].from))
		t.Logf("%s: two arm rails meet at %v (junction gap %.3e)", name, inner[0].to, gap)
		if gap > tol {
			t.Fatalf("%s rails do not meet at a point: gap %.3e > tol %.3e", name, gap, tol)
		}
	}
}

// TestCanalHostBite_PlaneBClosesToValidFace proves the unified assembler CLOSES the plane-B corner host to
// a valid retrimmed face on the fixture: both arm rails' outer ends anchor on the plane rectangle, so the
// far span closes and canalHostBite returns a face whose first loop is the retrimmed bite (inner + far).
func TestCanalHostBite_PlaneBClosesToValidFace(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	bundles, boundaries, rolls := n7CanalBiteInputs(t, w, arms, res)
	ff, reason := canalHostBite(arms[0].b, bundles, boundaries, rolls, w, res)
	if reason != "" {
		t.Fatalf("canalHostBite must CLOSE plane B (both outer ends anchor); declined: %s", reason)
	}
	if len(ff.loops) == 0 || len(ff.loops[0].pts) < 4 {
		t.Fatalf("plane-B retrim must yield a closed bite loop; got %d loops", len(ff.loops))
	}
	if _, isPl := ff.surface.(geom.Plane); !isPl {
		t.Fatalf("plane-B retrim surface is %T, want geom.Plane", ff.surface)
	}
	t.Logf("plane B retrimmed to a valid face: %d loops, outer bite has %d vertices", len(ff.loops), len(ff.loops[0].pts))
}

// TestCanalHostBite_WallAnchorEvidence is the risk #1 escalation gate: on the real N7 wall it MEASURES the
// two wall arm-rails' OUTER-end → bitten-loop anchor gaps (the evidence W4 relies on), asserts s_4 anchors
// (~0) while the s_5 TORUS rail's far end (azimuth 0) overshoots the FIXTURE band by orders of magnitude,
// and asserts canalHostBite HONEST-DECLINES with the measured gap rather than forcing a non-anchoring
// close. If a future fix lands the torus runout on the wall loop (a wider host / a rail terminator), this
// flips and must be revisited — the exact tripwire the flag calls for.
func TestCanalHostBite_WallAnchorEvidence(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	bundles, boundaries, rolls := n7CanalBiteInputs(t, w, arms, res)
	tol := res.Weld() * w.radius
	wall := arms[0].a
	inner, reason := canalInnerBite(wall, bundles, boundaries, rolls, tol)
	if reason != "" {
		t.Fatalf("wall inner bite declined: %s", reason)
	}
	star, ok := bittenLoop(wall, innerBiteKey(inner), tol)
	if !ok {
		t.Fatal("wall host must have an unambiguous bitten loop")
	}
	wsegs := segsFromLoop(star)
	gS4 := distPointToLoopEdges(wsegs, inner[0].from)          // s_4 (cylinder) rail outer end
	gS5 := distPointToLoopEdges(wsegs, inner[len(inner)-1].to) // s_5 (torus) rail outer end
	t.Logf("WALL outer-end anchor gaps (risk #1): s_4=%.4e  s_5(torus)=%.4e  (tol=res.Weld·r=%.3e)", gS4, gS5, tol)
	if gS4 > tol {
		t.Errorf("s_4 wall rail outer end must anchor on the wall loop; gap %.4e > tol %.3e", gS4, tol)
	}
	if gS5 <= tol {
		t.Fatalf("EXPECTED the s_5 torus rail to overshoot the fixture wall band; gap %.4e ≤ tol %.3e — revisit", gS5, tol)
	}
	_, br := canalHostBite(wall, bundles, boundaries, rolls, w, res)
	if br == "" || !strings.Contains(br, "far span will not close") {
		t.Fatalf("canalHostBite must honest-decline the wall with the anchor diagnostic; got %q", br)
	}
	t.Logf("honest wall bite decline: %s", br)
}

// TestChainBiteSegs_MistaggedFootLocusDeclines is the discrimination mutation: swapping feet[0]↔feet[1]
// (a mis-tagged foot-locus) makes the wall bridge feet[1], whose endpoints do NOT meet the wall rails'
// inner ends W0/W1 — so the chain gate DECLINES at the non-meeting junction instead of welding a corrupt
// loop. A passing bite must survive this; the real tag survives, the mutated one does not.
func TestChainBiteSegs_MistaggedFootLocusDeclines(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	bundles, boundaries, rolls := n7CanalBiteInputs(t, w, arms, res)
	tol := res.Weld() * w.radius

	if _, reason := canalInnerBite(arms[0].a, bundles, boundaries, rolls, tol); reason != "" {
		t.Fatalf("the CORRECT wall foot-locus tag must chain; got decline %q", reason)
	}
	swapped := canalBoundaries{
		endArcs: boundaries.endArcs, endArcsRev: boundaries.endArcsRev,
		feet:    [2]geom.Curve3{boundaries.feet[1], boundaries.feet[0]},
		feetRev: [2]bool{boundaries.feetRev[1], boundaries.feetRev[0]},
	}
	_, reason := canalInnerBite(arms[0].a, bundles, swapped, rolls, tol)
	if reason == "" {
		t.Fatal("a MIS-TAGGED wall foot-locus (feet[1] in feet[0]'s slot) must be caught by the chain gate")
	}
	if !strings.Contains(reason, "does not chain") {
		t.Fatalf("mis-tag decline must name the non-meeting junction; got %q", reason)
	}
	t.Logf("mis-tag caught by the chain gate: %s", reason)
}

// TestFootLocusBite_S10SharedEdgeIdentity is the s_10 shared-edge gate (unchanged intent from W3): the
// foot-locus feet[1] the s_10 boss bite splices is the SAME curve object the mid (s_10) arm face closes
// on (canalArmCornerRail), so the two seams are point-identical BY CONSTRUCTION. It asserts the bite's
// endpoints ARE the mid-arm corner-rail endpoints, and the mid-arm face's corner-rail samples are
// byte-identical (residual 0.0) to sampling feet[1] the same way.
func TestFootLocusBite_S10SharedEdgeIdentity(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	patch, boundaries, centres, scale := n7CanalWeldInputs(t, w, arms, res)
	_ = patch
	mid := n7MidArmIndex(t, arms)

	rail, rev, ok := canalArmCornerRail(boundaries, centres[mid], mid, mid)
	if !ok {
		t.Fatal("mid arm must take the u=1 foot-locus corner rail")
	}
	bite := footLocusBite(boundaries.feet[1])
	railLo, railHi := rail.Domain()
	if bite.from != rail.PointAt(railLo) || bite.to != rail.PointAt(railHi) {
		t.Fatalf("s_10 bite endpoints (%v→%v) are not the mid arm's corner-rail endpoints (%v→%v)", bite.from, bite.to, rail.PointAt(railLo), rail.PointAt(railHi))
	}
	face, ok := canalArmFace(arms[mid], centres[mid], rail, rev, w, scale, res)
	if !ok {
		t.Fatal("mid arm face must build")
	}
	corner := sampleCurve3Open(rail, rev)
	maxDev := 0.0
	for k, p := range corner {
		maxDev = stdmath.Max(maxDev, float64(p.DistanceTo(face.loops[0].pts[k])))
	}
	if maxDev != 0 {
		t.Fatalf("s_10 bite↔mid-arm shared-edge residual = %.3e, want exactly 0 (same curve, same sampling)", maxDev)
	}
	t.Logf("s_10 shared-edge (feet[1]) identity residual = %.3e", maxDev)
}

// TestCanalHostFaces_FarRunoutVerbatim proves the far-runout branch is VERBATIM reuse: canalHostFaces
// routes a far-runout-bitten host (on no roll surface, carrying no arm rails) through farArcsBiting/
// farRunoutFace, producing a face byte-identical to calling farRunoutFace directly, and passes an
// untouched face through transformFace unchanged.
func TestCanalHostFaces_FarRunoutVerbatim(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	_, boundaries, rolls := n7CanalBiteInputs(t, w, arms, res)
	tol := res.Weld() * w.radius

	body, host, bite := farRunoutHostAndBite(t)
	bundles := []canalArmBundle{{far: bite}}

	wantBites := farArcsBiting(host, farBundles(bundles), tol)
	if len(wantBites) != 1 {
		t.Fatalf("far arc must bite the synthetic host; got %d bites", len(wantBites))
	}
	want, ok := farRunoutFace(host, wantBites, nil, tol)
	if !ok {
		t.Fatal("farRunoutFace must splice the synthetic bite")
	}
	got, reason := canalHostFaces(body, w, boundaries, bundles, rolls, res)
	if reason != "" {
		t.Fatalf("canalHostFaces declined the far-runout host: %s", reason)
	}
	if len(got) != 1 {
		t.Fatalf("want one routed host face, got %d", len(got))
	}
	assertSameLoopPoints(t, got[0], want, "far-runout")

	plainBody, plain := farRunoutPlainHost(t)
	gotPlain, reason := canalHostFaces(plainBody, w, boundaries, nil, rolls, res)
	if reason != "" {
		t.Fatalf("canalHostFaces declined the untouched host: %s", reason)
	}
	assertSameLoopPoints(t, gotPlain[0], transformFace(plain, nil, nil, nil, nil, nil, 0), "pass-through")
}

// --- helpers ---------------------------------------------------------------

// pairResidual is the max distance of got[0]→a and got[1]→b — a point-pair identity residual (0 iff the
// two points coincide with (a,b) in that order).
func pairResidual(got []math.Point3, a, b math.Point3) float64 {
	return stdmath.Max(float64(got[0].DistanceTo(a)), float64(got[1].DistanceTo(b)))
}

// assertChainMeets asserts every consecutive junction of a chain meets within tol, logging the max gap.
func assertChainMeets(t *testing.T, chain []endSeg, tol float64, name string) {
	t.Helper()
	maxGap := 0.0
	for i := 0; i+1 < len(chain); i++ {
		g := float64(chain[i].to.DistanceTo(chain[i+1].from))
		maxGap = stdmath.Max(maxGap, g)
	}
	t.Logf("%s inner-bite chain max junction gap = %.3e (tol %.3e)", name, maxGap, tol)
	if maxGap > tol {
		t.Fatalf("%s inner-bite chain does not meet: max junction gap %.3e > tol %.3e", name, maxGap, tol)
	}
}

// assertBridgeIs asserts footLocusForHost(host) returns a bridge whose curve is the wanted foot-locus.
func assertBridgeIs(t *testing.T, host *topo.Face, b canalBoundaries, rolls []geom.Surface, tol float64, want geom.Curve3, name string) {
	t.Helper()
	bridge, ok := footLocusForHost(host, b, rolls, tol)
	if !ok {
		t.Fatalf("%s: footLocusForHost returned no bridge", name)
	}
	wlo, whi := want.Domain()
	if bridge.from != want.PointAt(wlo) || bridge.to != want.PointAt(whi) {
		t.Fatalf("%s: bridge endpoints (%v→%v) are not the wanted foot-locus (%v→%v)", name, bridge.from, bridge.to, want.PointAt(wlo), want.PointAt(whi))
	}
}

// assertNoBridge asserts footLocusForHost(host) returns no bridge (a plane host, on no roll surface).
func assertNoBridge(t *testing.T, host *topo.Face, b canalBoundaries, rolls []geom.Surface, tol float64, name string) {
	t.Helper()
	if _, ok := footLocusForHost(host, b, rolls, tol); ok {
		t.Fatalf("%s: footLocusForHost must return NO bridge (host is on neither roll surface)", name)
	}
}

// n7BossHost builds the s_10 boss host face (cylinder R=5 about (55,·,15) axis y) as a band — used only to
// check footLocusForHost tags it to feet[1] by roll-host identity (its loop content is not load-bearing).
func n7BossHost(t *testing.T) *topo.Face {
	t.Helper()
	bld := topo.NewBuilder(true, topo.Lineage{})
	lin := topo.Lineage{}
	axis, ref := math.V3(0, 1, 0), math.V3(1, 0, 0)
	deg := stdmath.Pi / 180
	bot := mustArcRef(t, math.P3(55, -20, 15), axis, ref, 5, -190*deg, 100*deg)
	top := mustArcRef(t, math.P3(55, 30, 15), axis, ref, 5, -90*deg, -100*deg)
	a0, a1 := bld.AddVertex(bot.PointAt(0), lin), bld.AddVertex(bot.PointAt(1), lin)
	a2, a3 := bld.AddVertex(top.PointAt(0), lin), bld.AddVertex(top.PointAt(1), lin)
	be := bld.AddEdge(bot, a0, a1, lin)
	right := bld.AddEdge(geom.NewLineSegment(a1.Point(), a2.Point()), a1, a2, lin)
	te := bld.AddEdge(top, a2, a3, lin)
	left := bld.AddEdge(geom.NewLineSegment(a3.Point(), a0.Point()), a3, a0, lin)
	cyl := mustCylinder(t, math.P3(55, 0, 15), math.V3(0, 1, 0), 5)
	return bld.AddFace(cyl, lin, topo.OuterLoop(topo.Fwd(be), topo.Fwd(right), topo.Fwd(te), topo.Fwd(left)))
}

// farRunoutHostAndBite builds a z=0 plane rectangle (in its own body) and a quarter-circle far arc that
// bites its corner at the origin — the two arc endpoints lie on the rectangle's two edges, so
// farRunoutFace can splice it. Returns the body, the host face (in body.Faces()), and the bite.
func farRunoutHostAndBite(t *testing.T) (*topo.Body, *topo.Face, endSeg) {
	t.Helper()
	bld := topo.NewBuilder(true, topo.Lineage{})
	host := n7PlaneHost(t, bld, mustPlane(t, math.P3(0, 0, 0), math.V3(0, 0, 1)),
		[]math.Point3{math.P3(0, 0, 0), math.P3(100, 0, 0), math.P3(100, 100, 0), math.P3(0, 100, 0)})
	arc := mustArcRef(t, math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 10, 0, stdmath.Pi/2)
	return bld.Build(), host, endSeg{from: arc.PointAt(0), to: arc.PointAt(1), curve: arc, mid: arc.PointAt(0.5), arc: true}
}

// farRunoutPlainHost builds a plane rectangle (in its own body) no bite touches — the pass-through case.
func farRunoutPlainHost(t *testing.T) (*topo.Body, *topo.Face) {
	t.Helper()
	bld := topo.NewBuilder(true, topo.Lineage{})
	host := n7PlaneHost(t, bld, mustPlane(t, math.P3(0, 0, 50), math.V3(0, 0, 1)),
		[]math.Point3{math.P3(0, 0, 50), math.P3(10, 0, 50), math.P3(10, 10, 50), math.P3(0, 10, 50)})
	return bld.Build(), host
}

// assertSameLoopPoints asserts two filletFaces have exactly-equal loop points (byte identity of the
// produced geometry) — the verbatim-reuse discriminator.
func assertSameLoopPoints(t *testing.T, got, want filletFace, name string) {
	t.Helper()
	if len(got.loops) != len(want.loops) {
		t.Fatalf("%s: got %d loops, want %d", name, len(got.loops), len(want.loops))
	}
	for i := range want.loops {
		gp, wp := got.loops[i].pts, want.loops[i].pts
		if len(gp) != len(wp) {
			t.Fatalf("%s loop %d: got %d pts, want %d", name, i, len(gp), len(wp))
		}
		for k := range wp {
			if gp[k] != wp[k] {
				t.Fatalf("%s loop %d pt %d: got %v, want %v (not verbatim)", name, i, k, gp[k], wp[k])
			}
		}
	}
}
