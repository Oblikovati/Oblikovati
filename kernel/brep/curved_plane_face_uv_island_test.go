// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// islandChart frames the 16×16 plate face at z=3 for the exact-frame trimmer.
func islandChart(t *testing.T) *planeFaceUV {
	t.Helper()
	pl, err := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	ring := []math.Point3{math.P3(-8, -8, 3), math.P3(8, -8, 3), math.P3(8, 8, 3), math.P3(-8, 8, 3)}
	f := planarFaceFromRings(pl, [][]math.Point3{ring}, topo.Lineage{})
	c, ok := newPlaneFaceUV(f, geom.ResolutionForSize(16))
	if !ok {
		t.Fatal("newPlaneFaceUV declined a square plate face")
	}
	return c
}

// TestSplitImprintByKind separates the closed conics from the straight imprint segments.
func TestSplitImprintByKind(t *testing.T) {
	t.Parallel()
	circle, err := geom.NewCircle(math.P3(0, 0, 3), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatal(err)
	}
	seg := geom.NewLineSegment(math.P3(-8, 0, 3), math.P3(8, 0, 3))
	straight, islands := splitImprintByKind([]geom.Curve3{seg, circle, seg})
	if len(straight) != 2 || len(islands) != 1 {
		t.Fatalf("split = %d straight, %d islands; want 2 and 1", len(straight), len(islands))
	}
	if islands[0] != geom.Curve3(circle) {
		t.Error("the circle must land in the island list, not the straight one")
	}
}

// TestConicIslandSegsCoverTheWholeCircle: the sampled island is a closed chain of imprint-tagged segments
// that carries the analytic curve and spans its whole domain, so a kept run re-emits the exact circle.
func TestConicIslandSegsCoverTheWholeCircle(t *testing.T) {
	t.Parallel()
	c := islandChart(t)
	circle, err := geom.NewCircle(math.P3(0, 0, 3), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatal(err)
	}
	segs := c.conicIslandSegs(circle)
	if len(segs) != imprintSampleCount {
		t.Fatalf("island segments = %d, want %d", len(segs), imprintSampleCount)
	}
	if segs[0].kind != segImprint || segs[0].curve != geom.Curve3(circle) {
		t.Errorf("segment 0 = kind %v curve %T, want segImprint on the source circle", segs[0].kind, segs[0].curve)
	}
	if segs[0].tA != 0 || stdmath.Abs(segs[len(segs)-1].tB-1) > 1e-12 {
		t.Errorf("parameter span = [%g,%g], want the whole [0,1] domain", segs[0].tA, segs[len(segs)-1].tB)
	}
	if d := segs[0].a.DistanceTo(segs[len(segs)-1].b); float64(d) > 1e-12 {
		t.Errorf("the island chain is open by %g in (u,v); want a closed ring", d)
	}
	if got := sampledRingArea(segs); stdmath.Abs(got-stdmath.Pi*25) > 1e-2 {
		t.Errorf("island encloses %.4f in (u,v), want ≈ %.4f (π·5²)", got, stdmath.Pi*25)
	}
}

// TestIslandSegsConcatenatesEveryConic: two islands sample into two full chains.
func TestIslandSegsConcatenatesEveryConic(t *testing.T) {
	t.Parallel()
	c := islandChart(t)
	inner, err := geom.NewCircle(math.P3(0, 0, 3), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatal(err)
	}
	outer, err := geom.NewCircle(math.P3(0, 0, 3), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(c.islandSegs([]geom.Curve3{inner, outer})); got != 2*imprintSampleCount {
		t.Errorf("islandSegs = %d segments, want %d", got, 2*imprintSampleCount)
	}
	if got := len(c.islandSegs(nil)); got != 0 {
		t.Errorf("islandSegs(nil) = %d segments, want 0", got)
	}
}

// TestAssembleSegmentsKeepsFrameAndIsland: the square frame's four edges survive alongside the island, so
// the arrangement sees the plate minus the circle.
func TestAssembleSegmentsKeepsFrameAndIsland(t *testing.T) {
	t.Parallel()
	c := islandChart(t)
	circle, err := geom.NewCircle(math.P3(0, 0, 3), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatal(err)
	}
	segs := c.assembleSegments([]geom.Curve3{circle})
	frame, imprint := 0, 0
	for _, s := range segs {
		if s.kind == segPolygon {
			frame++
		} else {
			imprint++
		}
	}
	if frame != 4 || imprint != imprintSampleCount {
		t.Errorf("assembled %d frame + %d imprint segments, want 4 + %d", frame, imprint, imprintSampleCount)
	}
	if !planeFaceContactOK(c, []geom.Curve3{circle}) {
		t.Error("an island imprint must not trip the straight-imprint grazing gate")
	}
}

// sampledRingArea is the shoelace area of a sampled (u,v) ring, for the island's coverage assertion.
func sampledRingArea(segs []uvSeg) float64 {
	var twice float64
	for _, s := range segs {
		twice += float64(s.a.X*s.b.Y - s.b.X*s.a.Y)
	}
	return stdmath.Abs(twice) / 2
}

// TestIslandContactOKDeclinesACrossedIsland: an island a straight imprint cuts through would be resolved
// on the island's sampled chord, so the trim declines instead (#3460).
func TestIslandContactOKDeclinesACrossedIsland(t *testing.T) {
	t.Parallel()
	c := islandChart(t)
	circle, err := geom.NewCircle(math.P3(0, 0, 3), math.V3(0, 0, 1), 5)
	if err != nil {
		t.Fatal(err)
	}
	across := geom.NewLineSegment(math.P3(-8, 0, 3), math.P3(8, 0, 3))
	clear := geom.NewLineSegment(math.P3(-8, 7, 3), math.P3(8, 7, 3))
	if islandContactOK(c, []geom.Curve3{circle}, []geom.Curve3{across}) {
		t.Error("a straight imprint crossing the island must decline")
	}
	if !islandContactOK(c, []geom.Curve3{circle}, []geom.Curve3{clear}) {
		t.Error("a straight imprint clear of the island is fine")
	}
}

// TestConicPairSeparated: apart and nested pass; overlapping (two crossing circles) declines.
func TestConicPairSeparated(t *testing.T) {
	t.Parallel()
	at := func(x, r float64) planeConic {
		return planeConic{center: math.P2(math.Scalar(x), 0), maj: math.V2(1, 0), A: r, B: r}
	}
	for _, c := range []struct {
		name string
		a, b planeConic
		want bool
	}{
		{"apart", at(0, 2), at(10, 2), true},
		{"nested", at(0, 5), at(0, 2), true},
		{"crossing", at(0, 5), at(6, 3), false},
		{"touching", at(0, 5), at(5, 5), false},
	} {
		if got := conicPairSeparated(c.a, c.b); got != c.want {
			t.Errorf("%s: conicPairSeparated = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestConicsNestedOrApartScansEveryPair: one bad pair among many declines the whole set.
func TestConicsNestedOrApartScansEveryPair(t *testing.T) {
	t.Parallel()
	at := func(x, r float64) planeConic {
		return planeConic{center: math.P2(math.Scalar(x), 0), maj: math.V2(1, 0), A: r, B: r}
	}
	if !conicsNestedOrApart([]planeConic{at(0, 2), at(10, 2), at(20, 2)}) {
		t.Error("three mutually apart islands are fine")
	}
	if conicsNestedOrApart([]planeConic{at(0, 2), at(10, 2), at(11, 3)}) {
		t.Error("a crossing pair anywhere in the set must decline")
	}
}
