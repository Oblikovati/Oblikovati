// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
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

// TestTrimRegionContainsReadsATwoRimBand moved to trim_region_periodic_test.go, where the derivation
// that closes a two-rim band into one contour is tested with the rest of the chart's boundary cases
// (ADR-0063).

func TestLoopRayCrossingsCountsAWrappingRingInItsLastStep(t *testing.T) {
	t.Parallel()
	ring := wrappingRim(2, true, true) // the v=2 rim, samples at u = 2πk/32, the last step unsampled
	for _, u := range []float64{6.2, 6.25, 0.05, -0.05, 12.5} {
		if got := loopRayCrossings(math.P2(u, 1), ring, true, false, true); got != 1 {
			t.Errorf("upward ray at u=%g crosses the rim %d times, want 1", u, got)
		}
	}
}
