// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Regression for the single-arm curved-runout retrim over a host whose bite lands on an INNER (hole) loop —
// OCCT blend/simple/M7 (curved-runout-r2a-m7-brief.md). M7's host B (plane x=60) flush-cuts the arm cylinder
// through its axis; its z=100 cap carries one unrelated footprint hole. Before the fix, singleRunoutHostFace
// retrimmed only the OUTER loop, so the cap's bite (which lands on the footprint hole) declined
// ("host geom.Plane retrim declined"). hostBittenLoop now routes the bite to the loop carrying the picked
// vertex — the inner footprint loop — while every prior single-arm green (single-loop hosts) keeps retrimming
// its outer loop unchanged. These tests pin both the whole-body weld and the hostBittenLoop routing directly.

// m7FixtureBody is the imported M7 primitive (box + flush-cut cylinder) before filleting.
func m7FixtureBody(t *testing.T) *topo.Body {
	t.Helper()
	return corpusFixture(t, "simple/M7.step")
}

// TestM7SingleArmRunoutWeldsWatertight drives the real M7 body through the single-arm runout assembly and
// asserts a certified watertight 11-face solid — the flush-cut-cap inner-loop retrim's crux. A declined or
// mis-classified retrim leaves reason non-empty or the body uncertified.
func TestM7SingleArmRunoutWeldsWatertight(t *testing.T) {
	body := m7FixtureBody(t)
	ef := m7SingleArmFillet(t, body)
	b, reason := singleArmRunoutBody(body, ef, ResolutionForBody(body))
	if reason != "" || b == nil {
		t.Fatalf("M7 single-arm runout declined: reason=%q body=%v (want a watertight weld)", reason, b != nil)
	}
	rep := Validate(b)
	if !rep.Valid || !rep.HolesContained || !b.IsSolid() || len(b.Faces()) != 11 {
		t.Fatalf("M7 weld not a certified 11-face solid: valid=%v holes=%v solid=%v faces=%d issues=%v",
			rep.Valid, rep.HolesContained, b.IsSolid(), len(b.Faces()), rep.Issues)
	}
}

// TestHostBittenLoopRoutesToInnerFootprint pins hostBittenLoop's routing on the real M7 flush-cut cap: the
// picked-edge vertex (60,25,100) sits on the cap's INNER footprint loop, so hostBittenLoop must return that
// inner loop (not the outer box square) — while a vertex ON the outer box square routes to the outer loop.
// This is the do-no-harm crux: outer-boundary bites (every prior single-arm green) keep their outer retrim.
func TestHostBittenLoopRoutesToInnerFootprint(t *testing.T) {
	cap0 := m7FlushCutCap(t, m7FixtureBody(t))
	outer, inner := m7CapLoops(t, cap0)
	tol := 1e-6 * boundingDiagOf(cap0)
	if got := hostBittenLoop(cap0, math.P3(60, 25, 100), tol); got != inner {
		t.Fatalf("hostBittenLoop for the picked vertex (60,25,100) returned %v, want the inner footprint loop", got == outer)
	}
	if got := hostBittenLoop(cap0, math.P3(0, 0, 100), tol); got != outer {
		t.Fatalf("hostBittenLoop for an outer-box vertex (0,0,100) returned the inner loop, want the outer box loop")
	}
}

// m7SingleArmFillet resolves M7's single r=10 pick and returns its solved curved-arm edgeFillet, asserting
// the dispatch classifier (isSingleArmRunout) admits it — the runout path this regression exercises.
func m7SingleArmFillet(t *testing.T, body *topo.Body) edgeFillet {
	t.Helper()
	edge := edgeNearestMidpoint(t, body, math.P3(60, 25, 125))
	edges, err := resolveFilletPicks(body, []EdgeFilletRadii{{Key: edge.ReferenceKey(), R0: 10, R1: 10}})
	if err != nil {
		t.Fatalf("resolve M7 pick: %v", err)
	}
	blends, miters, err := computeCorners(body, edges)
	if err != nil {
		t.Fatalf("compute M7 corners: %v", err)
	}
	fils, err := computeFillets(body, edges, blends, miters, FillConcaveOutward, nil)
	if err != nil {
		t.Fatalf("compute M7 fillets: %v", err)
	}
	if !isSingleArmRunout(fils) {
		t.Fatalf("M7 pick is not classified as a single-arm runout (dispatch regression)")
	}
	return fils[0]
}

// edgeNearestMidpoint returns the body edge whose parametric midpoint is closest to m (the corpus locator).
func edgeNearestMidpoint(t *testing.T, body *topo.Body, m math.Point3) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := stdmath.Inf(1)
	for _, e := range body.Edges() {
		if d := float64(e.Geometry().PointAt(0.5).DistanceTo(m)); d < bestD {
			bestD, best = d, e
		}
	}
	if best == nil || bestD > 1 {
		t.Fatalf("no M7 edge near locator midpoint %v (nearest at %.3e)", m, bestD)
	}
	return best
}

// m7FlushCutCap returns the z=100 cap plane that carries the footprint hole — the unique planar face with an
// inner loop whose vertices sit at z=100.
func m7FlushCutCap(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range body.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok || len(f.Loops()) < 2 || stdmath.Abs(pl.Origin.Z-100) > 1e-6 {
			continue
		}
		return f
	}
	t.Fatalf("M7 body carries no z=100 cap plane with a footprint hole")
	return nil
}

// m7CapLoops returns the cap's outer box loop and its inner footprint loop.
func m7CapLoops(t *testing.T, cap0 *topo.Face) (outer, inner *topo.Loop) {
	t.Helper()
	for _, l := range cap0.Loops() {
		if l.IsOuter() {
			outer = l
		} else {
			inner = l
		}
	}
	if outer == nil || inner == nil {
		t.Fatalf("M7 cap is not a 2-wire face (outer=%v inner=%v)", outer != nil, inner != nil)
	}
	return outer, inner
}

// boundingDiagOf is the diagonal of the face's vertex bounding box — the characteristic size for a
// model-relative loop-vertex tolerance (ADR-0042).
func boundingDiagOf(f *topo.Face) float64 {
	var pts []math.Point3
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			pts = append(pts, useFromVertex(u).Point())
		}
	}
	if len(pts) == 0 {
		return 1
	}
	lo, hi := pts[0], pts[0]
	for _, p := range pts[1:] {
		lo = math.P3(stdmath.Min(lo.X, p.X), stdmath.Min(lo.Y, p.Y), stdmath.Min(lo.Z, p.Z))
		hi = math.P3(stdmath.Max(hi.X, p.X), stdmath.Max(hi.Y, p.Y), stdmath.Max(hi.Z, p.Z))
	}
	return float64(hi.DistanceTo(lo))
}
