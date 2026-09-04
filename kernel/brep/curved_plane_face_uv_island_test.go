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
	straight, islands, _, open := splitImprintByKind([]geom.Curve3{seg, circle, seg})
	if len(straight) != 2 || len(islands) != 1 || len(open) != 0 {
		t.Fatalf("split = %d straight, %d islands, %d open; want 2, 1 and 0", len(straight), len(islands), len(open))
	}
	if len(islands[0]) != 1 || islands[0][0].curve != geom.Curve3(circle) {
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
	segs := c.islandCurveSegs(oneArc(circle))
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
	if got := len(c.islandSegs([]imprintCycle{oneCycle(inner), oneCycle(outer)})); got != 2*imprintSampleCount {
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
	if islandContactOK(c, []imprintCycle{oneCycle(circle)}, []geom.Curve3{across}) {
		t.Error("a straight imprint crossing the island must decline")
	}
	if !islandContactOK(c, []imprintCycle{oneCycle(circle)}, []geom.Curve3{clear}) {
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

// oneArc walks a closed curve over its whole domain.
func oneArc(cv geom.Curve3) imprintArc {
	lo, hi := cv.Domain()
	return imprintArc{curve: cv, t0: lo, t1: hi}
}

// oneCycle is the island a single closed curve makes.
func oneCycle(cv geom.Curve3) imprintCycle { return imprintCycle{oneArc(cv)} }

// cyclesOf is one single-curve island per curve.
func cyclesOf(cvs ...geom.Curve3) []imprintCycle {
	out := make([]imprintCycle, 0, len(cvs))
	for _, cv := range cvs {
		out = append(out, oneCycle(cv))
	}
	return out
}

// TestChainImprintCyclesAssemblesABigon: a section need not arrive as one curve. The two spiric
// branches of a single oval meet end to end and close; neither closes on its own, so neither is an
// island, and neither crosses the receiving face's boundary, so neither is an open crossing. Assembled
// first — as OCCT's PerformLoops builds wires before PerformAreas classifies them — the pair is one
// island (ADR-0062).
func TestChainImprintCyclesAssemblesABigon(t *testing.T) {
	t.Parallel()
	tor, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	plane, err := geom.NewPlane(math.P3(0, 6, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	arcs, ok := geom.TorusPlaneSection(tor, plane)
	if !ok || len(arcs) != 2 {
		t.Fatalf("want one oval in two branches, got ok=%v n=%d", ok, len(arcs))
	}
	straight, islands, _, open := splitImprintByKind(arcs)
	if len(straight) != 0 || len(open) != 0 {
		t.Fatalf("split = %d straight, %d open; both branches belong to the island", len(straight), len(open))
	}
	if len(islands) != 1 || len(islands[0]) != 2 {
		t.Fatalf("split made %d islands; want one cycle of two arcs", len(islands))
	}
	// The cycle is continuous: each arc ends where the next begins, and the last returns to the first.
	cyc := islands[0]
	for i, arc := range cyc {
		next := cyc[(i+1)%len(cyc)]
		end := arc.curve.PointAt(arc.t1)
		if d := float64(end.DistanceTo(next.curve.PointAt(next.t0))); d > 1e-9 {
			t.Errorf("arc %d ends %g from where the next begins: the cycle is not continuous", i, d)
		}
	}
}

// TestChainImprintCyclesLeavesOpenArcsAlone: arcs that do not close stay open, so a genuine crossing is
// never mistaken for an island.
func TestChainImprintCyclesLeavesOpenArcsAlone(t *testing.T) {
	t.Parallel()
	a, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, 1)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	b, err := geom.NewArc3d(math.P3(9, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, 1)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	cycles, rest := chainImprintCycles([]geom.Curve3{a, b})
	if len(cycles) != 0 {
		t.Errorf("two arcs that never meet made %d cycles, want none", len(cycles))
	}
	if len(rest) != 2 {
		t.Errorf("%d arcs came back open, want both", len(rest))
	}
}

// TestIslandsWalkNestedOrApartJudgesByTheWalk: the walk-based verdict is the one a NON-conic island
// gets — a torus's spiric oval has no closed form to separate it by — so it is exercised directly.
// A pair that crosses has samples on both sides of the other's ring; a pair apart, or nested, does not.
func TestIslandsWalkNestedOrApartJudgesByTheWalk(t *testing.T) {
	t.Parallel()
	c := islandChart(t)
	circle := func(x, r float64) geom.Curve3 {
		cv, err := geom.NewCircle(math.P3(math.Scalar(x), 0, 3), math.V3(0, 0, 1), r)
		if err != nil {
			t.Fatalf("NewCircle: %v", err)
		}
		return cv
	}
	if !islandsWalkNestedOrApart(c, cyclesOf(circle(0, 1), circle(10, 1))) {
		t.Error("two islands wholly apart were declined")
	}
	if !islandsWalkNestedOrApart(c, cyclesOf(circle(0, 4), circle(0, 1))) {
		t.Error("two NESTED islands were declined")
	}
	if islandsWalkNestedOrApart(c, cyclesOf(circle(0, 2), circle(3, 2))) {
		t.Error("two CROSSING islands were admitted")
	}
}
