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

// W3 canal-aware HOST retrims. Every assertion drives the REAL N7 fixture (n7CornerFill →
// extractCurvedCorner → resolveBlend), never a fabricated patch. The load-bearing deliverable is the
// WATERTIGHTNESS ANCHOR evidence (architect-flagged): the foot-locus endpoints must lie on the host's
// bitten loop for the splice to anchor. On N7 they do NOT (the corner vertices are the arm-rail junctions,
// ~37 units interior to the wall band), so retrimCanalHost HONEST-DECLINES with the measured gap and the
// weld floors — an honest report, never a loosened splice.

// TestRetrimCanalHost_WallAnchorGap is the escalation-evidence gate: on the real N7 wall host + wall
// foot-locus feet[0], it (1) MEASURES the foot-locus-endpoint→bitten-loop anchor gaps and logs them (the
// evidence W4 relies on), (2) asserts they exceed res.Weld·scale by orders of magnitude (the anchor does
// NOT hold), and (3) asserts retrimCanalHost DECLINES rather than forcing a non-anchoring splice. If a
// future intersection fix lands the foot-locus endpoints on the wall loop, this test flips and must be
// revisited (the anchor would then hold) — it is the exact tripwire the flag calls for.
func TestRetrimCanalHost_WallAnchorGap(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	_, boundaries, _, scale := n7CanalWeldInputs(t, w, arms, res)
	wall := arms[0].a // the cylinder wall host (arm s_4 rolls on it)
	if _, isCyl := wall.Geometry().(geom.Cylinder); !isCyl {
		t.Fatalf("N7 wall host is %T, want geom.Cylinder", wall.Geometry())
	}
	weldScale := res.Weld() * scale

	star, ok := bittenLoop(wall, footLocusMid(boundaries.feet[0]), res.Weld()*w.radius)
	if !ok {
		t.Fatal("wall host must have an unambiguous bitten loop for feet[0]")
	}
	g0, g1 := footLocusAnchorGaps(star, boundaries.feet[0])
	t.Logf("WALL foot-locus feet[0] anchor gaps: endpoint0=%.4e endpoint1=%.4e (res.Weld·scale=%.3e)", g0, g1, weldScale)

	if g0 <= weldScale || g1 <= weldScale {
		t.Fatalf("EXPECTED the foot-locus to NOT anchor on the fixture wall loop; got gaps %.4e/%.4e ≤ tol %.3e — revisit the anchor claim", g0, g1, weldScale)
	}
	if _, ok := retrimCanalHost(wall, boundaries.feet[0], w, res); ok {
		t.Fatal("retrimCanalHost must HONEST-DECLINE a non-anchoring foot-locus (never force a loosened splice)")
	}
}

// TestCanalHostFaces_WallDeclineCarriesAnchorGap proves the honest floor is DIAGNOSTIC: routing the wall
// host through canalHostFace declines with a reason that carries the measured anchor gap (so the
// controller can escalate the exact foot-locus↔loop intersection), not a bare failure.
func TestCanalHostFaces_WallDeclineCarriesAnchorGap(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	_, boundaries, centres, scale := n7CanalWeldInputs(t, w, arms, res)
	bundles, ok := canalArmBundles(arms, w, centres, scale, res)
	if !ok {
		t.Fatal("canalArmBundles must build the N7 far arcs")
	}
	wall := arms[0].a // the wall host; routing it (f == wall) triggers the foot-locus retrim + decline
	_, reason := canalHostFace(wall, wall, boundaries, bundles, w, res, res.Weld()*w.radius)
	if reason == "" {
		t.Fatal("canalHostFace must decline when the wall foot-locus does not anchor")
	}
	if !strings.Contains(reason, "watertightness anchor") {
		t.Fatalf("wall decline must carry the anchor diagnostic; got %q", reason)
	}
	t.Logf("honest wall-retrim floor: %s", reason)
}

// TestRetrimCanalHost_S10SharedEdgeIdentity is the s_10 shared-edge gate: the foot-locus the s_10 host
// retrim would splice (feet[1]) is the SAME curve object the mid (s_10) arm face closes on
// (canalArmCornerRail), so the two seams are point-identical BY CONSTRUCTION. It asserts the retrim's
// bite curve IS boundaries.feet[1], and that the mid-arm face's corner-rail samples are byte-identical
// (residual 0.0) to sampling feet[1] the same way — the watertight s_10 seam identity.
func TestRetrimCanalHost_S10SharedEdgeIdentity(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	_, boundaries, centres, scale := n7CanalWeldInputs(t, w, arms, res)
	mid := n7MidArmIndex(t, arms)

	rail, rev, ok := canalArmCornerRail(boundaries, centres[mid], mid, mid)
	if !ok {
		t.Fatal("mid arm must take the u=1 foot-locus corner rail")
	}
	// The retrim's bite is built from the SAME shared foot-locus feet[1] the mid arm closes on; its
	// endpoints are the foot-locus endpoints, so the s_10 host seam and the s_10 arm face share the curve.
	// (geom.BSplineCurve is uncomparable, so identity is proved by point equality, not ==.)
	bite := footLocusBite(boundaries.feet[1])
	railLo, railHi := rail.Domain()
	if bite.from != rail.PointAt(railLo) || bite.to != rail.PointAt(railHi) {
		t.Fatalf("retrim bite endpoints (%v→%v) are not the mid arm's corner-rail endpoints (%v→%v)", bite.from, bite.to, rail.PointAt(railLo), rail.PointAt(railHi))
	}
	// Build the mid arm face and assert its corner-rail portion equals sampling feet[1] the same way (0.0).
	face, ok := canalArmFace(arms[mid], centres[mid], rail, rev, w, scale, res)
	if !ok {
		t.Fatal("mid arm face must build")
	}
	corner := sampleCurve3Open(rail, rev)
	maxDev := 0.0
	for k, p := range corner {
		d := float64(p.DistanceTo(face.loops[0].pts[k]))
		maxDev = stdmath.Max(maxDev, d)
	}
	if maxDev != 0 {
		t.Fatalf("s_10 retrim↔mid-arm shared-edge residual = %.3e, want exactly 0 (same curve, same sampling)", maxDev)
	}
	t.Logf("s_10 shared-edge (feet[1]) identity residual = %.3e (mid-arm corner rail is the retrim's bite curve)", maxDev)
}

// TestCanalHostFaces_FarRunoutVerbatim proves the far-runout branch is VERBATIM reuse: canalHostFaces
// routes a far-runout-bitten host through farArcsBiting/farRunoutFace, producing a face byte-identical
// (loop points exactly equal) to calling farRunoutFace directly, and passes an untouched face through
// transformFace unchanged. Proof it is the same leaf machinery, not a reimplementation.
func TestCanalHostFaces_FarRunoutVerbatim(t *testing.T) {
	w, arms, res := n7CornerFill(t)
	_, boundaries, _, _ := n7CanalWeldInputs(t, w, arms, res)
	tol := res.Weld() * w.radius

	body, host, bite := farRunoutHostAndBite(t) // a plane rectangle + a corner-biting far arc
	bundles := []armRails{{far: bite}}

	// direct single-ball leaf calls (the oracle for "verbatim")
	wantBites := farArcsBiting(host, bundles, tol)
	if len(wantBites) != 1 {
		t.Fatalf("far arc must bite the synthetic host; got %d bites", len(wantBites))
	}
	want, ok := farRunoutFace(host, wantBites, tol)
	if !ok {
		t.Fatal("farRunoutFace must splice the synthetic bite")
	}

	got, reason := canalHostFaces(body, arms, w, boundaries, bundles, res)
	if reason != "" {
		t.Fatalf("canalHostFaces declined the far-runout host: %s", reason)
	}
	if len(got) != 1 {
		t.Fatalf("want one routed host face, got %d", len(got))
	}
	assertSameLoopPoints(t, got[0], want, "far-runout")

	// an untouched face passes through transformFace verbatim
	plainBody, plain := farRunoutPlainHost(t)
	gotPlain, reason := canalHostFaces(plainBody, arms, w, boundaries, nil, res)
	if reason != "" {
		t.Fatalf("canalHostFaces declined the untouched host: %s", reason)
	}
	assertSameLoopPoints(t, gotPlain[0], transformFace(plain, nil, nil, nil, nil), "pass-through")
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
