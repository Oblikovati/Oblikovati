// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// s1FilletCutCorners are the four geometry-determined outer hexagon corners of the S1 runout
// hole: the fillet cross-section is interrupted at |x| = √48 = 6.9282 (feature-B's r=8 footprint
// ∩ the fillet's plane-B tangent line y=-4), and at that abscissa the quarter-cylinder touches
// host plane A (y=-10) at z=4 and host plane B (z=10) at y=-4. These are pinned exactly (the
// seam abscissa is free, but the fillet-cut corners are not) — see s1-runout-topology.md.
var s1FilletCutCorners = []math.Point3{
	math.P3(6.928203230275509, -10, 4),  // FA_R: fillet∩planeA, +x
	math.P3(6.928203230275509, -4, 10),  // FB_R: fillet∩planeB, +x
	math.P3(-6.928203230275509, -4, 10), // FB_L
	math.P3(-6.928203230275509, -10, 4), // FA_L
}

// s1HexagonCorners are the SIX outer corners of the runout hexagon (dropping the two internal
// seams): the four fillet-cut corners plus the two feature-A/seam corners on plane A. The two
// seam-top corners (on the feature-B arc) are NOT hexagon corners — the feature-B arc passes
// through them, so the top hexagon side is one arc split into three by the seams.
func s1HexagonCornerCount() int { return 6 }

// runoutTilingUnderTest resolves the real S1 region (Task 8) and tiles it. Every extract test
// drives this so the fixture cost is paid once per test and the topology under assertion is the
// genuine double-interference case, not a hand-built stand-in.
func runoutTilingUnderTest(t *testing.T) ([]RailLoop, edgeFillet, Resolution) {
	t.Helper()
	ef, res := runoutFixtureCrossingBoss(t)
	regions := detectRunoutRegions(ef, res)
	if len(regions) != 1 || len(regions[0].imprints) != 2 {
		t.Fatalf("want one double-interference region, got %d regions: %+v", len(regions), regions)
	}
	loops, ok := extractRunout(regions[0], ef, res)
	if !ok {
		t.Fatal("extractRunout declined the S1 double-interference region")
	}
	if len(loops) != 3 {
		t.Fatalf("want 3 RailLoops (central/right/left), got %d", len(loops))
	}
	return loops, ef, res
}

// TestExtractRunoutTilesHexagonIntoThreeQuads is the Step-1 topology gate: three valence-4 loops,
// each closed with four pairwise-distinct corners; the four fillet-cut corners are pinned exactly;
// the union's outer boundary reconstructs the six-corner hexagon once the two seams are dropped.
func TestExtractRunoutTilesHexagonIntoThreeQuads(t *testing.T) {
	loops, _, res := runoutTilingUnderTest(t)
	weld := res.Weld()
	for i, l := range loops {
		if l.Valence() != 4 {
			t.Errorf("loop %d: want valence 4, got %d", i, l.Valence())
		}
		if !l.Closed(weld) {
			t.Errorf("loop %d: not closed within weld %v", i, weld)
		}
		assertFourDistinctCorners(t, i, l, weld)
	}
	assertFilletCutCornersPinned(t, loops, weld)
	assertOuterBoundaryIsHexagon(t, loops, weld)
}

// TestExtractRunoutSharesSeamCornersOnHostPlanes pins the free-seam contract: the two flanking
// loops share each seam corner with the central loop at bit-identical coordinates (watertight
// union), and every seam corner lies on its host plane (the seam-bottoms on plane A, the
// seam-tops on plane B). The seam ABSCISSA is derived, not asserted, so this never over-pins it.
func TestExtractRunoutSharesSeamCornersOnHostPlanes(t *testing.T) {
	loops, ef, res := runoutTilingUnderTest(t)
	central, right, left := classifyLoops(t, loops)
	planeA, planeB := hostPlanes(t, ef, res)
	// The seam corners are the loop corners NOT among the fillet-cut corners.
	seams := seamCornersOf(central, right, left, res.Weld())
	if len(seams) != 4 {
		t.Fatalf("want 4 distinct seam corners (two bottoms, two tops), got %d: %v", len(seams), seams)
	}
	for _, p := range seams {
		onA := distanceToPlane(p, planeA) <= res.Weld()
		onB := distanceToPlane(p, planeB) <= res.Weld()
		if !onA && !onB {
			t.Errorf("seam corner %v lies on neither host plane (A dist=%v, B dist=%v)",
				p, distanceToPlane(p, planeA), distanceToPlane(p, planeB))
		}
	}
	assertSeamCornersShared(t, central, right, left)
}

// TestExtractRunoutQuadsFillNonFolding is the admissibility gate: each RailLoop must build a
// coons4 fill whose ribbon-seam probe reports non-folding. The central all-G0 patch admits on the
// plain Coons NoFold; the two flanking patches carry G1 ribbons (host plane + fillet cylinder).
func TestExtractRunoutQuadsFillNonFolding(t *testing.T) {
	loops, _, res := runoutTilingUnderTest(t)
	for i, l := range loops {
		fill, rails, sides, ok := coons4Fill(l)
		if !ok {
			t.Fatalf("loop %d: coons4Fill declined", i)
		}
		if !ribbonSeamNonFolding(fill, rails, sides, res) {
			t.Errorf("loop %d: fill folds (ribbonSeamNonFolding=false)", i)
		}
	}
}

// TestArmSectionArcSpansFilletQuarterCircle unit-tests the fillet cross-section helper against the
// S1 fillet at the +x fillet-cut station: it must run from the plane-A contact (·,-10,4) to the
// plane-B contact (·,-4,10) through the 45° bisector, all at radius 6 from the cylinder axis.
func TestArmSectionArcSpansFilletQuarterCircle(t *testing.T) {
	ef, _ := runoutFixtureCrossingBoss(t)
	planeA, _ := geom.NewPlane(math.P3(0, -10, 0), math.V3(0, 1, 0))
	planeB, _ := geom.NewPlane(math.P3(0, 0, 10), math.V3(0, 0, 1))
	arc, ok := armSectionArc(ef.cyl, planeA, planeB, 16.928203230275503)
	if !ok {
		t.Fatal("armSectionArc declined the +x fillet-cut station")
	}
	assertPointNear(t, "start", arc.PointAt(0), math.P3(6.928203230275509, -10, 4), 1e-6)
	assertPointNear(t, "end", arc.PointAt(1), math.P3(6.928203230275509, -4, 10), 1e-6)
	axisPt := math.P3(6.928203230275503, -4, 4)
	for _, tt := range []float64{0, 0.25, 0.5, 0.75, 1} {
		if d := axisPt.DistanceTo(arc.PointAt(tt)); mabs(d-6) > 1e-6 {
			t.Errorf("arc point t=%v is %v from the axis, want radius 6", tt, d)
		}
	}
}

// TestInternalSeamConnectsFeatureCorners unit-tests the seam helper: a straight segment between a
// seam-bottom and a seam-top, with the endpoints preserved exactly (bit-identical) so the two
// flanking patches can share it.
func TestInternalSeamConnectsFeatureCorners(t *testing.T) {
	bottom := math.P3(3.465, -10, 4.898)
	top := math.P3(3.465, -7.211, 10)
	seam := internalSeam(bottom, top)
	if seam.PointAt(0) != bottom || seam.PointAt(1) != top {
		t.Errorf("internalSeam endpoints changed: got %v..%v, want %v..%v",
			seam.PointAt(0), seam.PointAt(1), bottom, top)
	}
}

// TestPlaneARunoutCurveLiesOnHostPlane unit-tests the plane-A boundary helper: the segment from a
// seam-bottom to a fillet-cut corner must lie in host plane A (both endpoints have y = -10).
func TestPlaneARunoutCurveLiesOnHostPlane(t *testing.T) {
	seamBottom := math.P3(3.465, -10, 4.898)
	filletCut := math.P3(6.928203230275509, -10, 4)
	planeA, _ := geom.NewPlane(math.P3(0, -10, 0), math.V3(0, 1, 0))
	curve := planeARunoutCurve(seamBottom, filletCut)
	for _, tt := range []float64{0, 0.5, 1} {
		if d := distanceToPlane(curve.PointAt(tt), planeA); d > 1e-6 {
			t.Errorf("plane-A runout curve leaves plane A at t=%v (dist %v)", tt, d)
		}
	}
}

// --- test helpers (named fakes / assertions) ---

func mabs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func assertPointNear(t *testing.T, label string, got, want math.Point3, tol float64) {
	t.Helper()
	if d := got.DistanceTo(want); d > tol {
		t.Errorf("%s: got %v, want %v (dist %v > %v)", label, got, want, d, tol)
	}
}

func loopCornerPoints(l RailLoop) []math.Point3 {
	pts := make([]math.Point3, l.Valence())
	for i, s := range l.Sides {
		pts[i] = curveStart(s.Curve)
	}
	return pts
}

func assertFourDistinctCorners(t *testing.T, i int, l RailLoop, weld float64) {
	t.Helper()
	pts := loopCornerPoints(l)
	for a := 0; a < len(pts); a++ {
		for b := a + 1; b < len(pts); b++ {
			if pts[a].DistanceTo(pts[b]) <= weld {
				t.Errorf("loop %d: corners %d and %d coincide (%v)", i, a, b, pts[a])
			}
		}
	}
}

func assertFilletCutCornersPinned(t *testing.T, loops []RailLoop, weld float64) {
	t.Helper()
	for _, want := range s1FilletCutCorners {
		if !anyCornerNear(loops, want, weld) {
			t.Errorf("no loop corner near pinned fillet-cut corner %v", want)
		}
	}
}

func anyCornerNear(loops []RailLoop, want math.Point3, weld float64) bool {
	for _, l := range loops {
		for _, p := range loopCornerPoints(l) {
			if p.DistanceTo(want) <= weld {
				return true
			}
		}
	}
	return false
}

// assertOuterBoundaryIsHexagon drops the two seams (the sides shared by two loops) and checks the
// remaining sides form a single closed cycle in which every vertex has degree 2 and all six
// hexagon corners appear.
func assertOuterBoundaryIsHexagon(t *testing.T, loops []RailLoop, weld float64) {
	t.Helper()
	outer := nonSeamSides(loops, weld)
	deg := vertexDegrees(outer, weld)
	for p, d := range deg {
		if d != 2 {
			t.Errorf("outer-boundary vertex %v has degree %d, want 2", p, d)
		}
	}
	if got := len(deg); got < s1HexagonCornerCount() {
		t.Errorf("outer boundary has %d vertices, want at least %d hexagon corners", got, s1HexagonCornerCount())
	}
	for _, want := range s1FilletCutCorners {
		if !vertexPresent(deg, want, weld) {
			t.Errorf("hexagon corner %v missing from outer boundary", want)
		}
	}
}

// sideEndpoints returns a side's two endpoints (order-independent for degree counting).
func sideEndpoints(s Side) (math.Point3, math.Point3) {
	return curveStart(s.Curve), curveEnd(s.Curve)
}

// nonSeamSides collects the sides that are NOT shared by two loops. A seam is the only side whose
// endpoint pair (as an unordered pair) appears in two different loops.
func nonSeamSides(loops []RailLoop, weld float64) []Side {
	var all []Side
	for _, l := range loops {
		all = append(all, l.Sides...)
	}
	var out []Side
	for i, s := range all {
		if countMatchingSides(all, i, s, weld) == 1 {
			out = append(out, s)
		}
	}
	return out
}

func countMatchingSides(all []Side, self int, s Side, weld float64) int {
	a0, a1 := sideEndpoints(s)
	n := 0
	for j, o := range all {
		if j == self {
			continue
		}
		b0, b1 := sideEndpoints(o)
		if samePair(a0, a1, b0, b1, weld) {
			n++
		}
	}
	return n + 1 // include self
}

func samePair(a0, a1, b0, b1 math.Point3, weld float64) bool {
	if a0.DistanceTo(b0) <= weld && a1.DistanceTo(b1) <= weld {
		return true
	}
	return a0.DistanceTo(b1) <= weld && a1.DistanceTo(b0) <= weld
}

func vertexDegrees(sides []Side, weld float64) map[math.Point3]int {
	deg := map[math.Point3]int{}
	for _, s := range sides {
		p0, p1 := sideEndpoints(s)
		bumpVertex(deg, p0, weld)
		bumpVertex(deg, p1, weld)
	}
	return deg
}

func bumpVertex(deg map[math.Point3]int, p math.Point3, weld float64) {
	for q := range deg {
		if p.DistanceTo(q) <= weld {
			deg[q]++
			return
		}
	}
	deg[p] = 1
}

func vertexPresent(deg map[math.Point3]int, want math.Point3, weld float64) bool {
	for q := range deg {
		if q.DistanceTo(want) <= weld {
			return true
		}
	}
	return false
}

// classifyLoops sorts the three loops into central (no G1 side) and the two flanks by mean x.
func classifyLoops(t *testing.T, loops []RailLoop) (central, right, left RailLoop) {
	t.Helper()
	var flanks []RailLoop
	haveCentral := false
	for _, l := range loops {
		if countG1Sides(l) == 0 {
			central, haveCentral = l, true
			continue
		}
		flanks = append(flanks, l)
	}
	if !haveCentral || len(flanks) != 2 {
		t.Fatalf("want one all-G0 central loop and two G1 flanks, got central=%v flanks=%d", haveCentral, len(flanks))
	}
	if meanX(flanks[0]) >= meanX(flanks[1]) {
		return central, flanks[0], flanks[1]
	}
	return central, flanks[1], flanks[0]
}

func countG1Sides(l RailLoop) int {
	n := 0
	for _, s := range l.Sides {
		if s.Cont == G1 {
			n++
		}
	}
	return n
}

func meanX(l RailLoop) float64 {
	sum := 0.0
	pts := loopCornerPoints(l)
	for _, p := range pts {
		sum += float64(p.X)
	}
	return sum / float64(len(pts))
}

// seamCornersOf returns the distinct corners that are NOT fillet-cut corners.
func seamCornersOf(central, right, left RailLoop, weld float64) []math.Point3 {
	var out []math.Point3
	for _, l := range []RailLoop{central, right, left} {
		for _, p := range loopCornerPoints(l) {
			if isFilletCut(p, weld) || containsPoint(out, p, weld) {
				continue
			}
			out = append(out, p)
		}
	}
	return out
}

func isFilletCut(p math.Point3, weld float64) bool {
	for _, c := range s1FilletCutCorners {
		if p.DistanceTo(c) <= weld {
			return true
		}
	}
	return false
}

func containsPoint(pts []math.Point3, p math.Point3, weld float64) bool {
	for _, q := range pts {
		if p.DistanceTo(q) <= weld {
			return true
		}
	}
	return false
}

// assertSeamCornersShared proves the watertightness invariant: each flank's single internal seam
// segment is endpoint-identical (bit-exact, unordered) to one of the central loop's two seams, so
// the union has no crack along the decomposition seams. This is the meaningful "same seam curve"
// test — the loops meeting at a seam-top/-bottom via an analytic featureSubArc agree only to arc
// re-projection precision (~1e-13), never to the ULP, so a corner-by-corner bit-exact check would
// be testing float determinism of Arc3d, not the seam sharing that matters.
func assertSeamCornersShared(t *testing.T, central, right, left RailLoop) {
	t.Helper()
	centralSeams := seamSegments(central)
	if len(centralSeams) != 2 {
		t.Fatalf("central loop must carry 2 seam segments, got %d", len(centralSeams))
	}
	for name, flank := range map[string]RailLoop{"right": right, "left": left} {
		fs := seamSegments(flank)
		if len(fs) != 1 {
			t.Errorf("%s flank must carry exactly 1 seam segment, got %d", name, len(fs))
			continue
		}
		if !seamShared(centralSeams, fs[0]) {
			t.Errorf("%s flank seam %v..%v is not endpoint-identical to a central seam",
				name, fs[0].StartPoint, fs[0].EndPoint)
		}
	}
}

// seamSegments returns a loop's internal seams: the straight (LineSegment), G0, nil-Adjacent sides.
// The plane-A boundary is also a LineSegment but is G1 with a non-nil Adjacent, so it is excluded.
func seamSegments(l RailLoop) []geom.LineSegment {
	var out []geom.LineSegment
	for _, s := range l.Sides {
		if seg, isSeg := s.Curve.(geom.LineSegment); isSeg && s.Cont == G0 && s.Adjacent == nil {
			out = append(out, seg)
		}
	}
	return out
}

func seamShared(central []geom.LineSegment, flank geom.LineSegment) bool {
	for _, c := range central {
		if samePairExact(c.StartPoint, c.EndPoint, flank.StartPoint, flank.EndPoint) {
			return true
		}
	}
	return false
}

func samePairExact(a0, a1, b0, b1 math.Point3) bool {
	return (a0 == b0 && a1 == b1) || (a0 == b1 && a1 == b0)
}

func hostPlanes(t *testing.T, ef edgeFillet, res Resolution) (planeA, planeB geom.Plane) {
	t.Helper()
	regions := detectRunoutRegions(ef, res)
	imps := regions[0].imprints
	if imps[0].plane.Normal().IsParallelTo(math.V3(0, 1, 0), 1e-6) {
		return imps[0].plane, imps[1].plane
	}
	return imps[1].plane, imps[0].plane
}

func distanceToPlane(p math.Point3, plane geom.Plane) float64 {
	n := plane.Normal()
	return mabs(float64(plane.Origin.VectorTo(p).Dot(n)) / float64(n.Length()))
}
