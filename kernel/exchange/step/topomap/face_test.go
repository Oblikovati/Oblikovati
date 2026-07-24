// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// bulgingArcLoop builds a single-edge bound loop whose edge is a semicircular arc from (1,0,0) to
// (-1,0,0) bulging through (0,1,0): both endpoints sit at y=0 but the curve interior peaks at y=1.
// It is the adversarial fixture for review Finding 1 — a curved trim edge non-monotone in the sweep
// direction, whose d-extremum lives at the curve INTERIOR, invisible to an endpoint-only bound.
func bulgingArcLoop(t *testing.T) []boundLoop {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("test", "body", 0)))
	start := bld.AddVertex(math.P3(1, 0, 0), topo.NewLineage(topo.Tok("test", "vertex", 0)))
	end := bld.AddVertex(math.P3(-1, 0, 0), topo.NewLineage(topo.Tok("test", "vertex", 1)))
	arc, err := geom.Arc3dByThreePoints(start.Point(), math.P3(0, 1, 0), end.Point())
	if err != nil {
		t.Fatalf("build bulging arc: %v", err)
	}
	edge := bld.AddEdge(arc, start, end, topo.NewLineage(topo.Tok("test", "edge", 0)))
	return []boundLoop{{outer: true, uses: []topo.Use{topo.Fwd(edge)}}}
}

// TestExtrusionSweepRangeCoversCurvedEdgeInterior locks review Finding 1: the sweep bound must cover a
// curved trim edge's interior excursion along the sweep direction, not merely its endpoints. The profile
// is a single control point at the origin (so its d-term cancels: lo=pMin, hi=pMax), the sweep direction
// is +Y, and the boundary is the bulging arc whose endpoints project to y=0 but whose interior reaches
// y=1. The vertex-only bound this fix replaced returned hi=0 — under-covering the swept patch so the
// trimming loop referenced v outside [lo,hi] and the shell silently re-opened. The RangeBox-corner bound
// must report hi≈1. This FAILS against the pre-fix endpoint-only loopPoints.
func TestExtrusionSweepRangeCoversCurvedEdgeInterior(t *testing.T) {
	bounds := bulgingArcLoop(t)
	ctrl := []math.Point3{math.P3(0, 0, 0)} // profile term cancels
	d := math.V3(0, 1, 0)

	lo, hi := extrusionSweepRange(bounds, ctrl, d)

	const peak = 1.0 // the arc interior peaks at y=1, beyond both y=0 endpoints
	if hi < peak-1e-6 {
		t.Fatalf("sweep hi = %.6g, want >= %.6g — bound under-covers the curved edge's interior (Finding 1)", hi, peak)
	}
	if lo > 1e-6 {
		t.Errorf("sweep lo = %.6g, want ~0 (arc stays at y>=0)", lo)
	}
}
