// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"
)

// syntheticSpineClusterPair builds two single-imprint spineClusters with the given [lo,hi]
// spine intervals — a named fixture for exercising mergeSpineClusters' merge predicate in
// isolation, without paying for a real edgeFillet/imprint solve (runoutImprint/imprintCut are
// irrelevant to the merge decision itself, only lo/hi are).
func syntheticSpineClusterPair(loA, hiA, loB, hiB float64) []spineCluster {
	return []spineCluster{
		{lo: loA, hi: hiA},
		{lo: loB, hi: hiB},
	}
}

// TestMergeSpineClusters_SeparateRunoutsStaySeparate is the tolerance-slack regression: two
// independently-solved conics can report spine intervals that disagree by ULP-level float noise
// even when they truly touch, so the merge predicate must accept a model-relative weld slack
// (res.Weld()) rather than an exact `lo <= hiEdge` compare. Clusters separated by MORE than weld
// must stay two regions; clusters within weld (but not exactly touching — the case a bare
// compare gets wrong) must merge into one.
func TestMergeSpineClusters_SeparateRunoutsStaySeparate(t *testing.T) {
	weld := ResolutionForSize(50).Weld()

	beyond := syntheticSpineClusterPair(0, 10, 10+1.5*weld, 20)
	if got := mergeSpineClusters(beyond, weld); len(got) != 2 {
		t.Fatalf("want 2 regions for intervals separated by >weld (%v), got %d: %+v", weld, len(got), got)
	}

	within := syntheticSpineClusterPair(0, 10, 10+0.5*weld, 20)
	if got := mergeSpineClusters(within, weld); len(got) != 1 {
		t.Fatalf("want 1 region for intervals within weld (%v) of each other, got %d: %+v", weld, len(got), got)
	}
}

// TestDetectRunoutRegions_S1BossesClusterIntoOneRegion is the crux case this task exists for:
// S1's two bosses (one per host plane) both interfere in the SAME fillet span, so
// detectRunoutRegions must cluster their two imprints into a single coupled region — the
// double-interference hexagon this milestone targets (advisor pitfall 5) — rather than reporting
// two independent, non-overlapping runouts.
func TestDetectRunoutRegions_S1BossesClusterIntoOneRegion(t *testing.T) {
	ef, res := runoutFixtureCrossingBoss(t)
	regions := detectRunoutRegions(ef, res)
	if len(regions) != 1 {
		t.Fatalf("want 1 coupled region (S1's two bosses share a fillet span), got %d: %+v",
			len(regions), regions)
	}
	region := regions[0]
	if len(region.imprints) != 2 {
		t.Fatalf("want the region to bundle both bosses' imprints, got %d: %+v",
			len(region.imprints), region.imprints)
	}
	if len(region.cuts) != 2 {
		t.Fatalf("want the region to bundle both bosses' cuts, got %d: %+v", len(region.cuts), region.cuts)
	}
	if region.imprints[0].hostIsA == region.imprints[1].hostIsA {
		t.Fatalf("want the two imprints on different hosts (S1's two independent bosses), both had hostIsA=%v",
			region.imprints[0].hostIsA)
	}

	// The fillet-cut span is exact geometry (S1's r8/r6 bosses against the shared edge), not an
	// approximation, so the tolerance stays at model-relative Weld() scale rather than the brief's
	// rounded "~6.93" (=4*sqrt(3)) display value.
	wantSpan := 8 * stdmath.Sqrt(3) // both bosses' cuts together span 2*(4*sqrt(3)) along the spine
	span := region.hiEdge - region.loEdge
	if tol := 4 * res.Weld(); stdmath.Abs(span-wantSpan) > tol {
		t.Fatalf("want merged spine span ~%v (8*sqrt(3)), got %v [%v,%v] (tol %v)",
			wantSpan, span, region.loEdge, region.hiEdge, tol)
	}

	// Both extremes must actually come from cut data (not e.g. only one imprint's cut dominating
	// both ends because the other imprint was silently dropped): recompute the raw spine interval
	// of every bundled cut independently and check the region's [loEdge,hiEdge] matches their
	// combined min/max.
	wantLo, wantHi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, cut := range region.cuts {
		lo, hi := spineInterval(cut, ef.cyl)
		wantLo, wantHi = stdmath.Min(wantLo, lo), stdmath.Max(wantHi, hi)
	}
	if tol := 4 * res.Weld(); stdmath.Abs(region.loEdge-wantLo) > tol || stdmath.Abs(region.hiEdge-wantHi) > tol {
		t.Fatalf("want region interval [%v,%v] (both cuts' extremes), got [%v,%v] (tol %v)",
			wantLo, wantHi, region.loEdge, region.hiEdge, tol)
	}
}
