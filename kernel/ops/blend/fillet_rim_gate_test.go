// SPDX-License-Identifier: GPL-2.0-only

package blend_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Regression for the rim-fillet pick gate widening (loneRimPick/resolveRim, fillet_rim.go). The STEP
// importer never emits geom.Circle — every imported full circle is a geom.Arc3d with SweepAngle≈2π —
// so the gate must classify a closed full-sweep Arc3d as a circular rim exactly like a geom.Circle,
// and keep rejecting a genuinely partial arc. isClosedCircularEdge is the single predicate both
// loneRimPick/resolveRim AND fullCircleRimGeom (sphere_zone_mesh.go) call; these cases pin its
// behaviour directly, at the edge-classification level, without needing a full solid body.

// gateEdge builds a standalone edge with the given curve and closure (same start/end vertex when
// closed=true, distinct vertices otherwise) — isClosedCircularEdge only inspects the edge's curve and
// vertex identity, so no faces/body are needed to exercise it.
func gateEdge(t *testing.T, curve geom.Curve3, closed bool) *topo.Edge {
	t.Helper()
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("gate", "body", 0)))
	start := bld.AddVertex(math.P3(1, 0, 0), topo.NewLineage(topo.Tok("gate", "v", 0)))
	end := start
	if !closed {
		end = bld.AddVertex(math.P3(0, 1, 0), topo.NewLineage(topo.Tok("gate", "v", 1)))
	}
	return bld.AddEdge(curve, start, end, topo.NewLineage(topo.Tok("gate", "e", 0)))
}

func TestIsClosedCircularEdgeAcceptsGeomCircle(t *testing.T) {
	t.Parallel()
	c, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatal(err)
	}
	if e := gateEdge(t, c, true); !probe.IsClosedCircularEdge(e) {
		t.Error("a closed geom.Circle edge must be recognized as a circular rim")
	}
}

// TestIsClosedCircularEdgeAcceptsFullSweepArc3d is the widen itself: a closed geom.Arc3d whose sweep
// is a full turn (the shape every STEP-imported circular rim actually has) must count as a circular
// rim, exactly like a geom.Circle — before this fix, loneRimPick/resolveRim rejected every such edge.
func TestIsClosedCircularEdgeAcceptsFullSweepArc3d(t *testing.T) {
	t.Parallel()
	a, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 5, 0, 2*stdmath.Pi)
	if err != nil {
		t.Fatal(err)
	}
	if e := gateEdge(t, a, true); !probe.IsClosedCircularEdge(e) {
		t.Error("a closed full-sweep (2π) Arc3d edge must be recognized as a circular rim")
	}
}

// TestIsClosedCircularEdgeRejectsPartialArc keeps a genuinely partial arc (a real corner arc, half a
// turn, with distinct start/end vertices) declined — the widen must not swallow the ordinary
// loneArcPick territory.
func TestIsClosedCircularEdgeRejectsPartialArc(t *testing.T) {
	t.Parallel()
	a, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 5, 0, stdmath.Pi)
	if err != nil {
		t.Fatal(err)
	}
	if e := gateEdge(t, a, false); probe.IsClosedCircularEdge(e) {
		t.Error("an open half-turn Arc3d edge must NOT be recognized as a circular rim")
	}
}

// TestIsClosedCircularEdgeRejectsClosedShortSweep guards the tolerance boundary: even a SELF-closed
// (start==end) Arc3d must still be rejected if its sweep is far from 2π (a degenerate/malformed edge),
// so the predicate gates on sweep angle, not merely on vertex identity.
func TestIsClosedCircularEdgeRejectsClosedShortSweep(t *testing.T) {
	t.Parallel()
	a, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 5, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatal(err)
	}
	if e := gateEdge(t, a, true); probe.IsClosedCircularEdge(e) {
		t.Error("a closed edge with a quarter-turn sweep must NOT be recognized as a full circular rim")
	}
}
