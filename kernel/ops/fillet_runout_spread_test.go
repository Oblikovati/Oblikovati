// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// A synthetic fan: axis along +x through the origin, radius 2. A far edge from the apex (0,0,0)
// straight along +y crosses the cylinder (distance-2 tube about the x-axis) at y=2.
func TestSplitOnFarEdgeAnalytic(t *testing.T) {
	fan := endCornerFan{
		radius: 2,
		center: math.P3(0, 0, 0),
		axis:   math.V3(1, 0, 0),
		apex:   math.P3(0, 0, 0),
	}
	fe := fanEdge{from: math.P3(0, 0, 0), to: math.P3(0, 10, 0)}
	p, ok := splitOnFarEdge(fan, fe)
	if !ok {
		t.Fatal("expected a crossing")
	}
	if d := p.DistanceTo(math.P3(0, 2, 0)); d > 1e-9 {
		t.Errorf("split at %v, want (0,2,0) (dist %.3g)", p, d)
	}
}

// An edge oblique to the axis (both x and y components) still crosses the tube at exactly one
// point in (0,1); the expected crossing (-1.8,2,0) was hand-derived from the quadratic
// (A=100, B=0, C=-4 -> t=0.2) and independently checked against distance-to-axis-line == r.
func TestSplitOnFarEdgeAnalyticOblique(t *testing.T) {
	fan := endCornerFan{
		radius: 2,
		center: math.P3(0, 0, 0),
		axis:   math.V3(1, 0, 0),
	}
	fe := fanEdge{from: math.P3(-3, 0, 0), to: math.P3(3, 10, 0)}
	p, ok := splitOnFarEdge(fan, fe)
	if !ok {
		t.Fatal("expected a crossing")
	}
	if d := p.DistanceTo(math.P3(-1.8, 2, 0)); d > 1e-9 {
		t.Errorf("split at %v, want (-1.8,2,0) (dist %.3g)", p, d)
	}
}

// A far edge that never comes within the fillet radius of the axis (at least distance 5 from the
// axis, always outside the r=2 tube) must report no crossing.
func TestSplitOnFarEdgeAnalyticMiss(t *testing.T) {
	fan := endCornerFan{
		radius: 2,
		center: math.P3(0, 0, 0),
		axis:   math.V3(1, 0, 0),
	}
	fe := fanEdge{from: math.P3(5, 5, 0), to: math.P3(5, 10, 0)}
	if _, ok := splitOnFarEdge(fan, fe); ok {
		t.Fatal("expected no crossing for an edge that never nears the tube")
	}
}

// Far edge exactly parallel to the +x axis at radial distance 3 from it; with r=2 the tube
// is never reached, and the quadratic degenerates (a=0,b=0) -> the axis-parallel fallback
// in smallestRootIn01 must return ok=false rather than divide by zero.
func TestSplitOnFarEdgeParallelMiss(t *testing.T) {
	fan := endCornerFan{radius: 2, center: math.P3(0, 0, 0), axis: math.V3(1, 0, 0), apex: math.P3(0, 0, 0)}
	fe := fanEdge{from: math.P3(0, 3, 0), to: math.P3(10, 3, 0)}
	if _, ok := splitOnFarEdge(fan, fe); ok {
		t.Error("parallel non-crossing edge must not report a split")
	}
}

// TestSolveRunoutSpreadChainCloses is the weld-twice invariant gate: a hand-built 3-far-face fan
// (axis +x, r=2, apex origin) must assemble into a closed tA -> split(201) -> split(202) -> tB
// chain, with every interior far edge split at exactly the point its two bordering faces share.
func TestSolveRunoutSpreadChainCloses(t *testing.T) {
	fan := endCornerFan{
		radius: 2, center: math.P3(0, 0, 0), axis: math.V3(1, 0, 0), apex: math.P3(0, 0, 0),
		ta: math.P3(0, 2, 0), tb: math.P3(0, -2, 0),
		fan: []fanFace{
			{face: 101, normal: math.V3(0, 1, 0), entryEdge: 0, exitEdge: 201},
			{face: 102, normal: math.V3(0, 0, 1), entryEdge: 201, exitEdge: 202},
			{face: 103, normal: math.V3(0, -1, 0), entryEdge: 202, exitEdge: 0},
		},
		farEdges: []fanEdge{
			{edge: 201, from: math.P3(0, 0, 0), to: math.P3(0, 7, 7), leftFace: 101, rightFace: 102},
			{edge: 202, from: math.P3(0, 0, 0), to: math.P3(0, -7, 7), leftFace: 102, rightFace: 103},
		},
	}
	sp, err := solveRunoutSpread(fan)
	if err != nil {
		t.Fatalf("solveRunoutSpread: %v", err)
	}
	if len(sp.splits) != 2 {
		t.Fatalf("splits = %d, want 2", len(sp.splits))
	}
	// The A-flank piece starts at ta; the B-flank piece ends at tb; consecutive pieces meet at splits.
	a := sp.pieces[101]
	b := sp.pieces[102]
	c := sp.pieces[103]
	if a.tIn.DistanceTo(fan.ta) > 1e-9 {
		t.Errorf("A-flank piece must start at ta, got %v", a.tIn)
	}
	if c.tOut.DistanceTo(fan.tb) > 1e-9 {
		t.Errorf("B-flank piece must end at tb, got %v", c.tOut)
	}
	if a.tOut.DistanceTo(sp.splits[201]) > 1e-9 {
		t.Errorf("face101.tOut must equal split 201")
	}
	// Full weld chain: a broken split-sharing (e.g. each face computing its own boundary point
	// instead of reading the shared sp.splits entry) would fail one of these even though the
	// first two assertions above pass.
	if b.tIn.DistanceTo(sp.splits[201]) > 1e-9 {
		t.Errorf("face102.tIn must equal split 201")
	}
	if b.tOut.DistanceTo(sp.splits[202]) > 1e-9 {
		t.Errorf("face102.tOut must equal split 202")
	}
	if c.tIn.DistanceTo(sp.splits[202]) > 1e-9 {
		t.Errorf("face103.tIn must equal split 202")
	}
}
