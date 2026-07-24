// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// BenchmarkLocateEdgeBSplineStrip measures the per-pick cost of edge picking a high-aspect B-spline
// panel — the #2010 hot path. LocateUsingPoint(kind=KindEdge) runs closerEdge → discretizeEdge →
// densifyStarvedRail → starvedEdgeTarget for every edge every call, and UNLIKE curved-face picking
// (RayCastFaces, memoized wholesale by pickTess) it has NO whole-face memo, so before this fix each
// pick re-ran metricScale (~25 DerivativesAt per incident B-spline face) on both starved rails every
// frame. With faceMetricScale the metric is computed once per face lifetime and every subsequent pick
// reuses it. This benchmark lives in package ops (not the ops_test pick_bench_test.go beside
// BenchmarkRayCastFacesSphere) because it reuses the unexported #2009 cylindricalStripBody fixture.
func BenchmarkLocateEdgeBSplineStrip(b *testing.B) {
	s := cylindricalStripSurfaceBunched(b, bunchedR, bunchedSweep, bunchedH, bunchedK)
	body, _ := cylindricalStripBody(b, s, bunchedR, bunchedSweep, bunchedH)
	// A point on the v=0 straight rail's midpoint (r·cosα, −r·sinα, h/2) so the edge pick resolves,
	// exercising closerEdge/starvedEdgeTarget on both rails identically to the interactive hover pick.
	alpha := bunchedSweep / 2
	p := math.P3(bunchedR*stdmath.Cos(alpha), -bunchedR*stdmath.Sin(alpha), bunchedH/2)
	q := DefaultQuality()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := LocateUsingPoint(body, topo.KindEdge, p, 1.0, q); !ok {
			b.Fatal("edge pick missed the rail it is aimed at")
		}
	}
}
