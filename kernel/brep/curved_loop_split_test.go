// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

func TestSplitEdgeAtPointsCutsAClosedRimInEveryBranch(t *testing.T) {
	t.Parallel()
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	res := geom.ResolutionForSize(2)
	// A whole rim traversed from parameter 0.3 one full turn to 1.3: a vertex at parameter 0 sits at
	// 1.0 inside that span, and one at 0.6 sits inside as itself.
	e := loopEdge{curve: circle, t0: 0.3, t1: 1.3}
	pieces := splitEdgeAtPoints(e, []math.Point3{circle.PointAt(0), circle.PointAt(0.6)}, res)
	if len(pieces) != 3 || pieces[0].t1 != 0.6 || pieces[1].t1 != 1.0 || pieces[2].t1 != 1.3 {
		t.Fatalf("pieces = %+v, want cuts at 0.6 and 1.0", pieces)
	}
	// A point off the curve, or at the edge's own end, cuts nothing.
	if got := splitEdgeAtPoints(e, []math.Point3{math.P3(2, 0, 0), circle.PointAt(0.3)}, res); len(got) != 1 {
		t.Errorf("off-curve and end points produced %d pieces, want the edge itself", len(got))
	}
}

func TestSplitEdgeAtPointsKeepsTraversalOrderWhenReversed(t *testing.T) {
	t.Parallel()
	seg := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	e := loopEdge{curve: seg, t0: 1, t1: 0}
	pieces := splitEdgeAtPoints(e, []math.Point3{math.P3(0.25, 0, 0), math.P3(0.75, 0, 0)}, geom.ResolutionForSize(1))
	if len(pieces) != 3 || pieces[0].t0 != 1 || pieces[0].t1 != 0.75 || pieces[2].t1 != 0 {
		t.Fatalf("reversed pieces = %+v, want 1→0.75→0.25→0", pieces)
	}
}

// bandRegion is the developed trim of a cylinder band between two wrapping rims, each sampled one step
// short of its full turn, the way loopToUV samples a rim.
func bandRegion(n int) trimRegion {
	rim := func(v float64) []math.Point2 {
		ring := make([]math.Point2, 0, n)
		for k := 0; k < n; k++ {
			ring = append(ring, math.P2(2*stdmath.Pi*float64(k)/float64(n), v))
		}
		return ring
	}
	return trimRegion{rings: [][]math.Point2{rim(2), rim(0)}, uPeriodic: true}
}

func TestTrimRegionContainsReadsATwoRimBand(t *testing.T) {
	t.Parallel()
	r := bandRegion(32)
	for _, u := range []float64{0, 1, 3, 6.2, 6.28} { // 6.2 lies in the rims' unsampled last step
		if !r.contains(math.P2(u, 1)) {
			t.Errorf("(%g, 1) between the rims reads outside", u)
		}
		if r.contains(math.P2(u, 2.5)) || r.contains(math.P2(u, -0.5)) {
			t.Errorf("(%g, ±) beyond a rim reads inside", u)
		}
	}
}

func TestLoopRayCrossingsCountsAWrappingRingInItsLastStep(t *testing.T) {
	t.Parallel()
	ring := bandRegion(32).rings[0] // the v=2 rim, samples at u = 2πk/32, the last step unsampled
	for _, u := range []float64{6.2, 6.25, 0.05, -0.05, 12.5} {
		if got := loopRayCrossings(math.P2(u, 1), ring, true, false, true); got != 1 {
			t.Errorf("upward ray at u=%g crosses the rim %d times, want 1", u, got)
		}
	}
}
