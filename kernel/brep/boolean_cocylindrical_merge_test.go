// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The cocylindrical wall merge's corpus (ADR-0061 stage 5). Two kept faces on ONE surface that share
// a boundary are one face; two that share none stay two.

// cylinderWalls counts a body's analytic cylinder faces.
func cylinderWalls(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}

// TestBoreContinuingABoreIsOneWall: a blind bore drilled from the top, then continued to the bottom by
// a second coaxial drill of the SAME radius. The two tools' walls lie on one surface and meet at the
// circle where the first ended, which bounds nothing — the bore is one hole.
func TestBoreContinuingABoreIsOneWall(t *testing.T) {
	t.Parallel()
	blk, err := SolidBlock(math.P3(-3, -3, 0), math.P3(3, 3, 6), "blk")
	if err != nil {
		t.Fatalf("block: %v", err)
	}
	upper, _ := SolidCylinder(math.P3(0, 0, 3), math.V3(0, 0, 1), 1, 4)
	lower, _ := SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 1, 4)
	step, err := Boolean(Difference, blk, upper)
	if err != nil {
		t.Fatalf("first bore: %v", err)
	}
	through, err := Boolean(Difference, step, lower)
	if err != nil {
		t.Fatalf("continuing bore: %v", err)
	}
	assertWatertight(t, through)
	if n := cylinderWalls(through); n != 1 {
		t.Errorf("the continued bore has %d cylinder walls, want 1", n)
	}
	assertMergeMovedNoCoordinate(t, through)
}

// assertMergeMovedNoCoordinate pins the merge as combinatorial: it drops a boundary and re-chains the
// rest, so no edge acquires an achieved tolerance. A merge that nudged a seam to make the two loops
// meet would show here as a nonzero achieved boundary tolerance (ADR-0042).
func assertMergeMovedNoCoordinate(t *testing.T, b *topo.Body) {
	t.Helper()
	if tol := b.AchievedBoundaryTolerance(); tol != 0 {
		t.Errorf("the merged body's achieved boundary tolerance is %g, want 0 — the merge moved geometry", tol)
	}
}

// TestTwoBoresWithNoSharedEdgeStayTwoFaces is the NEGATIVE row: two bores on ONE surface — same axis,
// same radius, so geom.SurfacesCoincide answers yes — with a wall of material between them. They share
// no edge, so nothing dissolves and the body keeps both walls. A merge that fired on the surface alone
// would weld two holes into one and delete the material between them.
func TestTwoBoresWithNoSharedEdgeStayTwoFaces(t *testing.T) {
	t.Parallel()
	blk, err := SolidBlock(math.P3(-3, -3, 0), math.P3(3, 3, 10), "blk")
	if err != nil {
		t.Fatalf("block: %v", err)
	}
	bottom, _ := SolidCylinder(math.P3(0, 0, -1), math.V3(0, 0, 1), 1, 4) // z ∈ [−1, 3]
	top, _ := SolidCylinder(math.P3(0, 0, 7), math.V3(0, 0, 1), 1, 4)     // z ∈ [7, 11]
	one, err := Boolean(Difference, blk, bottom)
	if err != nil {
		t.Fatalf("first bore: %v", err)
	}
	both, err := Boolean(Difference, one, top)
	if err != nil {
		t.Fatalf("second bore: %v", err)
	}
	assertWatertight(t, both)
	assertMergeMovedNoCoordinate(t, both)
	if n := cylinderWalls(both); n != 2 {
		t.Errorf("two bores separated by material have %d cylinder walls, want 2 — they share no edge", n)
	}
}

// TestMergedWallCarriesBothParentsKeys: every reference key that existed on either parent still
// resolves on the merged face (ADR-0043 — a pick must survive the operation that consumed it).
func TestMergedWallCarriesBothParentsKeys(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	b, _ := SolidCylinder(math.P3(0, 0, 4), math.V3(0, 0, 1), 2, 3)
	keyA, keyB := wallKeyOf(t, a), wallKeyOf(t, b)
	res, err := Boolean(Union, a, b)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	for _, k := range [][]byte{keyA, keyB} {
		f, ok := res.FindFaceByKey(k)
		if !ok {
			t.Fatalf("the merged wall does not resolve parent key %q", string(k))
		}
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			t.Errorf("parent key %q resolved to a %T, not the merged cylinder wall", string(k), f.Geometry())
		}
	}
}

// wallKeyOf is the reference key of a cylinder primitive's wall face.
func wallKeyOf(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f.ReferenceKey()
		}
	}
	t.Fatal("the body has no cylinder wall")
	return nil
}

// TestMergedWallCarriesTheChartItsLoopsDetermine: the merged face's parametric trim is the union of
// the two in the covering space, and it is RECORDED — a reader must not have to re-derive it, and a
// pair whose fused loops determine none is declined instead (chartedMerge).
func TestMergedWallCarriesTheChartItsLoopsDetermine(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	b, _ := SolidCylinder(math.P3(0, 0, 4), math.V3(0, 0, 1), 2, 3)
	res, err := Boolean(Union, a, b)
	if err != nil {
		t.Fatalf("union: %v", err)
	}
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); !ok {
			continue
		}
		assertChartSpansTheUnion(t, f)
		return
	}
	t.Fatal("the union kept no cylinder wall")
}

// assertChartSpansTheUnion checks the merged wall's chart covers BOTH parents' axial runs: v reaches
// from the lower band's base to the upper band's top, which is what "the charts were unioned" means.
func assertChartSpansTheUnion(t *testing.T, f *topo.Face) {
	t.Helper()
	chart := f.Chart()
	if len(chart) == 0 {
		t.Fatal("the merged wall carries no chart; its loops determine one and it must be recorded")
	}
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, c := range chart {
		for _, p := range c {
			lo, hi = stdmath.Min(lo, float64(p.Y)), stdmath.Max(hi, float64(p.Y))
		}
	}
	if stdmath.Abs(lo) > 1e-9 || stdmath.Abs(hi-7) > 1e-9 {
		t.Errorf("the merged chart's v span is [%v, %v], want [0, 7] — the union of the two bands", lo, hi)
	}
}

// TestAmbiguousPairingIsRefusedByName drives a REAL pair through the merge to a refusal, and asserts
// the refusal reaches the recorder.
//
// The pair is a cylinder's own wall against a face that carries ONE edge on that wall's seam. The wall
// walks its seam twice — up and down — so both traversals run with that single edge, and which of them
// it dissolves is not decidable. That is the ambiguous-pairing exit, and it must ship as a named
// diagnostic rather than a body quietly carrying two faces where one belongs.
func TestAmbiguousPairingIsRefusedByName(t *testing.T) {
	t.Parallel()
	wall := wallFaceOf(t, math.P3(0, 0, 0), 2, 4)
	rec := &diag.Recorder{}
	if _, ok := mergeOnSharedBoundary(wall, faceOnSeamOf(t, wall), rec); ok {
		t.Fatal("a pair whose shared boundary cannot be paired one-to-one was merged anyway")
	}
	assertMergeDeclineRecorded(t, rec, "an edge of one runs with two of the other")
}

// faceOnSeamOf builds a face on the SAME surface whose only boundary is the given wall's seam edge,
// walked the other way. It is the smallest pair that makes the pairing ambiguous.
func faceOnSeamOf(t *testing.T, wall curvedFace) curvedFace {
	t.Helper()
	seam := wall.loops[0].edges[0]
	return curvedFace{surface: wall.surface, reversed: wall.reversed,
		loops: []curvedLoop{{edges: []loopEdge{{curve: seam.curve, t0: seam.t1, t1: seam.t0}}}}}
}

// TestUndecidedChartIsRefusedByName: the fused loops of a pair on a DOUBLY periodic surface — a torus
// — do not determine which of the two regions the merged face is, and ADR-0063 refuses to guess. The
// merge must refuse with it and say so, not hand the face a's chart or none.
func TestUndecidedChartIsRefusedByName(t *testing.T) {
	t.Parallel()
	torus, err := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	equator := geom.Circle{Center: math.P3(0, 0, 0), Normal: math.V3(0, 0, 1).AsUnit(),
		RefDir: math.V3(1, 0, 0).AsUnit(), Radius: 6.5}
	face := curvedFace{surface: torus, loops: []curvedLoop{{edges: []loopEdge{{curve: equator, t0: 0, t1: 1}}}}}
	if _, why := chartedMerge(face, face, face.loops); why != declineUndecidedChart {
		t.Fatalf("a torus band whose fused loops bound two regions gave %q, want the undecided-chart decline", why)
	}
	rec := &diag.Recorder{}
	recordMergeDecline(rec, face, declineUndecidedChart)
	assertMergeDeclineRecorded(t, rec, "do not determine a trim")
}

// TestMixedComplementIsRefusedByName: the complement flag is the ONE datum a face's rings cannot
// carry (ADR-0063), so a pair that disagrees on it cannot be given either parent's answer. Inheriting
// a's silently would decide the merged face's outer loop by which operand happened to be first.
func TestMixedComplementIsRefusedByName(t *testing.T) {
	t.Parallel()
	a := wallFaceOf(t, math.P3(0, 0, 0), 2, 4)
	b := a
	b.outerless = true
	if _, why := chartedMerge(a, b, a.loops); why != declineMixedComplement {
		t.Fatalf("a complement merged with a patch gave %q, want the mixed-complement decline", why)
	}
	rec := &diag.Recorder{}
	recordMergeDecline(rec, a, declineMixedComplement)
	assertMergeDeclineRecorded(t, rec, "closed-surface complement")
}

// TestARefusedPairIsReportedOnceAcrossTheWholeScan: merging one pair restarts the scan, so a pair that
// refuses is revisited. It must still be reported ONCE — a defect repeated on every pass says nothing
// the first one did not, and buries the pairs that matter.
func TestARefusedPairIsReportedOnceAcrossTheWholeScan(t *testing.T) {
	t.Parallel()
	wall := wallFaceOf(t, math.P3(0, 0, 0), 2, 4)
	rec := &diag.Recorder{}
	// Two bands that DO merge, plus the seam face that cannot be paired: the merge of the first pair
	// restarts the scan over the refusing one.
	faces := mergeCoincidentFaces([]curvedFace{
		wall, wallFaceOf(t, math.P3(0, 0, 4), 2, 3), faceOnSeamOf(t, wall)}, rec)
	if len(faces) != 2 {
		t.Fatalf("the scan left %d faces, want 2 (the two bands merged, the seam face refused)", len(faces))
	}
	n := 0
	for _, d := range rec.Records() {
		if d.Code == CodeCocylindricalMergeUndecided {
			n++
		}
	}
	if n != 1 {
		t.Errorf("the refused pair was reported %d times, want exactly 1; got %v", n, rec.Records())
	}
}

// assertMergeDeclineRecorded checks the named decline reached the recorder as a Defect, carrying the
// reason that refused.
func assertMergeDeclineRecorded(t *testing.T, rec *diag.Recorder, reason string) {
	t.Helper()
	for _, d := range rec.Records() {
		if d.Code == CodeCocylindricalMergeUndecided && d.Severity == diag.Defect &&
			strings.Contains(d.Detail, reason) {
			return
		}
	}
	t.Errorf("no %q Defect naming %q on the recorder; got %v", CodeCocylindricalMergeUndecided, reason, rec.Records())
}
