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
