// SPDX-License-Identifier: GPL-2.0-only

package validate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Guards for the EXACT self-intersection detector (M48/C3, Oblikovati/Oblikovati#3477). The old
// detector tessellated both faces, scanned triangle pairs and then forgave any crossing shallower
// than the two meshes' own faceting error — so the verdict moved with the caller's mesh.Quality. These
// tests pin the two halves of the replacement: the verdict no longer depends on mesh.Quality at all, and
// a face pair trimmed out of one surface sheet is still caught even though it has no intersection
// curve to find.

// twoBarelyOverlappingCylinders returns one body holding two parallel solid cylinders of radius r
// whose sides pass through each other by exactly `depth` — a genuine interpenetration of two CURVED
// faces whose depth is dialable.
func twoBarelyOverlappingCylinders(t *testing.T, r, depth float64) *topo.Body {
	t.Helper()
	a, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r, 2)
	if err != nil {
		t.Fatalf("SolidCylinder a: %v", err)
	}
	b, err := brep.SolidCylinder(math.P3(math.Scalar(2*r-depth), 0, 0), math.V3(0, 0, 1), r, 2)
	if err != nil {
		t.Fatalf("SolidCylinder b: %v", err)
	}
	return topo.MergeBodies(topo.NewLineage(topo.Tok("imp", "body", 0)), true, a, b)
}

// TestSelfIntersectionVerdictIsIndependentOfQuality is the #3477 regression, and the falsifiable one:
// two curved faces that pass through each other by 0.01 are reported at EVERY quality.
//
// Under the mesh detector the allowance was the sum of the two faces' chord tolerances — 2 × 0.05 at
// DefaultQuality and 2 × 1e-3 at PropertyQuality — so this same body was VALID for a display caller
// and INVALID for a property caller. Falsify by reinstating any faceting allowance and the
// DefaultQuality row goes red.
func TestSelfIntersectionVerdictIsIndependentOfQuality(t *testing.T) {
	t.Parallel()
	body := twoBarelyOverlappingCylinders(t, 1, 0.01)
	for _, q := range []struct {
		name string
		q    mesh.Quality
	}{{"default", mesh.DefaultQuality()}, {"property", mesh.PropertyQuality()}} {
		if hits := SelfIntersections(body, q.q); len(hits) == 0 {
			t.Errorf("%s quality (chord tol %g) missed a 0.01-deep crossing of two cylinder sides",
				q.name, q.q.Tol())
		}
	}
}

// stackedBlocks merges two solid blocks that TOUCH — one resting on the other — into a single body,
// sharing no topology. It is the kernel-scale form of what two turns of a coil do at pitch = profile
// depth: separate walls of one body meeting, not passing through each other.
func stackedBlocks(t *testing.T, lower, upper [2]math.Point3) *topo.Body {
	t.Helper()
	a, err := brep.SolidBlock(lower[0], lower[1], "lower")
	if err != nil {
		t.Fatalf("SolidBlock lower: %v", err)
	}
	b, err := brep.SolidBlock(upper[0], upper[1], "upper")
	if err != nil {
		t.Fatalf("SolidBlock upper: %v", err)
	}
	return topo.MergeBodies(topo.NewLineage(topo.Tok("stack", "body", 0)), true, a, b)
}

// TestCoincidentFacesAreReportedWhicheverWayTheyFace pins the ONE configuration this detector cannot
// resolve locally, so the limit is on the record rather than discovered again. Two coincident faces
// that FACE each other are, in any neighbourhood of the shared patch, indistinguishable from two solids
// resting on one another — material on both sides in both cases — so a wall folded through itself
// (#2086's sheet-metal lip, which must be caught) and a block stacked on a block (which is contact)
// read the same. The detector reports both, exactly as the mesh detector it replaced did; separating
// them needs a global containment query, not a face pair.
func TestCoincidentFacesAreReportedWhicheverWayTheyFace(t *testing.T) {
	t.Parallel()
	p := math.P3
	body := stackedBlocks(t, [2]math.Point3{p(0, 0, 0), p(2, 2, 1)}, [2]math.Point3{p(0, 0, 1), p(2, 2, 2)})
	if hits := SelfIntersections(body, mesh.DefaultQuality()); len(hits) == 0 {
		t.Error("two coincident overlapping faces must be reported, as the mesh detector reported them")
	}
}

// TestPerpendicularFacesTouchingAlongALineAreContact is the other half of the same regression, and the
// #2075 shape at body scale: a block set beside and above another touches it along ONE LINE, where each
// one's side face meets the other's horizontal face at a right angle. Their surfaces do intersect, and
// the intersection line runs inside both trims — which is exactly why an INCLUSIVE in-both-trims
// witness cannot decide this. Falsify by relaxing crossingWitness back to brep.PointInFaceTrim without
// strictlyInsideTrim: the contact line is inside both trims and every such touch reads as a crossing.
func TestPerpendicularFacesTouchingAlongALineAreContact(t *testing.T) {
	t.Parallel()
	p := math.P3
	body := stackedBlocks(t, [2]math.Point3{p(0, 0, 0), p(2, 2, 1)}, [2]math.Point3{p(2, 0, 1), p(4, 2, 2)})
	if hits := SelfIntersections(body, mesh.DefaultQuality()); len(hits) != 0 {
		t.Errorf("two blocks touching along one line report %d interpenetration(s): %+v", len(hits), hits)
	}
}

// TestOverlappingBlocksAreStillReported is the positive control for both guards above: move the upper
// block down so the two solids genuinely occupy the same space, and it must be reported.
func TestOverlappingBlocksAreStillReported(t *testing.T) {
	t.Parallel()
	p := math.P3
	body := stackedBlocks(t, [2]math.Point3{p(0, 0, 0), p(2, 2, 1)}, [2]math.Point3{p(0.5, 0.5, 0.5), p(2.5, 2.5, 2)})
	if hits := SelfIntersections(body, mesh.DefaultQuality()); len(hits) == 0 {
		t.Error("two blocks that genuinely overlap must be reported")
	}
}

// TestFaceScanCachesBoxesAndProbes is the guard on the fine-pitch coil hang. topo.Face.RangeBox walks
// the face's edges and evaluates their curves, so an O(F²) pair scan that asks for it inside the loop
// is O(F²·E) — measured, that took a coil's turn-clearance check (#2080, hundreds of swept faces) past
// twenty minutes. The scan must capture each box once, and compute each face's probes at most once.
func TestFaceScanCachesBoxesAndProbes(t *testing.T) {
	t.Parallel()
	body, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 3, 4), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	scan := newFaceScan(body.Faces())
	for i, f := range body.Faces() {
		if scan.boxes[i] != f.RangeBox() {
			t.Fatalf("face %d's captured box %v is not its own %v", i, scan.boxes[i], f.RangeBox())
		}
	}
	first := scan.trimProbes(0)
	if again := scan.trimProbes(0); len(first) == 0 || &again[0] != &first[0] {
		t.Error("a face's trim probes must be computed once and reused, not rebuilt per pair")
	}
}

// TestFaceScanBroadPhaseMatchesAnAllPairsWalk pins that the box tree is a PURE broad phase: it must
// yield exactly the pairs whose range boxes overlap — no more (which would cost exact geometry) and no
// fewer (which would hide an interpenetration). Falsify by querying a shrunken box: the counts diverge.
func TestFaceScanBroadPhaseMatchesAnAllPairsWalk(t *testing.T) {
	t.Parallel()
	merged := topo.MergeBodies(topo.NewLineage(topo.Tok("bp", "body", 0)), true,
		brepfixture.Tetra(1, math.V3(0, 0, 0)), brepfixture.Tetra(1, math.V3(0.2, 0.2, 0.2)), brepfixture.Tetra(1, math.V3(9, 0, 0)))
	scan := newFaceScan(merged.Faces())
	got, want := map[[2]int]bool{}, map[[2]int]bool{}
	for i := range scan.faces {
		scan.tree.Query(scan.boxes[i], func(j int) bool {
			if j > i {
				got[[2]int{i, j}] = true
			}
			return false
		})
		for j := i + 1; j < len(scan.boxes); j++ {
			if scan.boxes[i].Intersects(scan.boxes[j]) {
				want[[2]int{i, j}] = true
			}
		}
	}
	if len(want) == 0 {
		t.Fatal("the fixture produced no overlapping face pairs, so the comparison proves nothing")
	}
	for k := range want {
		if !got[k] {
			t.Errorf("the broad phase missed overlapping pair %v", k)
		}
	}
	if len(got) != len(want) {
		t.Errorf("the broad phase yielded %d pairs, an all-pairs walk yields %d", len(got), len(want))
	}
}

// TestUnresolvedSurfacePairDeclinesRatherThanReports pins the DIRECTION of the detector's one named
// decline, which is the opposite of boundary_cross.go's for the same geom.SurfaceIntersect verdict: a
// validity query that manufactured a defect on every pair the intersector cannot resolve would condemn
// healthy bodies and fail sound features through ValidateBodyEntities.
func TestUnresolvedSurfacePairDeclinesRatherThanReports(t *testing.T) {
	t.Parallel()
	if _, hit := declineUnresolvedSurfacePair(); hit {
		t.Error("an unresolvable surface pair must answer NO crossing, not a defect")
	}
}

// TestSelfIntersectionsPassATangentBlend is the #2077 population the faceting allowance existed to
// excuse: a fillet band is TANGENT to both the wall it blends and the face it runs onto, so two
// meshes of it stray to opposite sides of the true surfaces and appear to cross. On the exact B-rep
// the tangency is a shared edge and there is nothing to forgive — at either quality.
func TestSelfIntersectionsPassATangentBlend(t *testing.T) {
	t.Parallel()
	body, err := brep.SolidCylinderFilletedTop(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 10, 1)
	if err != nil {
		t.Fatalf("SolidCylinderFilletedTop: %v", err)
	}
	for _, q := range []mesh.Quality{mesh.DefaultQuality(), mesh.PropertyQuality()} {
		if hits := SelfIntersections(body, q); len(hits) != 0 {
			t.Errorf("a tangent fillet band reports %d self-intersection(s) at chord tol %g: %+v",
				len(hits), q.Tol(), hits)
		}
	}
}

// TestCoplanarNeighbourFacesAreClean is the #2074 body-level regression, carried over from the
// deleted triangle coplanar branch: two coplanar quads meeting along a shared edge — the shape a
// sheet-metal wall's end cap makes with the sheet's side — must report no self-intersection.
func TestCoplanarNeighbourFacesAreClean(t *testing.T) {
	t.Parallel()
	p := math.P3
	left := brepfixture.QuadBody("left", p(0, 0, 0), p(0, 2, 0), p(0, 2, 2), p(0, 0, 2))
	right := brepfixture.QuadBody("right", p(0, 2, 0), p(0, 4, 0), p(0, 4, 2), p(0, 2, 2))
	merged := topo.MergeBodies(topo.NewLineage(topo.Tok("imp", "body", 0)), false, left, right)
	if hits := SelfIntersections(merged, mesh.DefaultQuality()); len(hits) != 0 {
		t.Errorf("coplanar faces meeting along an edge report %d self-intersection(s): %+v", len(hits), hits)
	}
}

// TestDuplicateCoincidentFacesAreReported covers the same-sheet arm's rule 2: two faces with the SAME
// trim have no probe strictly inside the other, so only the mutual-coverage rule can see them — and
// they are the doubled-wall defect a bad import leaves behind.
func TestDuplicateCoincidentFacesAreReported(t *testing.T) {
	t.Parallel()
	p := math.P3
	one := brepfixture.QuadBody("w1", p(0, 0, 0), p(2, 0, 0), p(2, 0, 2), p(0, 0, 2))
	two := brepfixture.QuadBody("w2", p(0, 0, 0), p(2, 0, 0), p(2, 0, 2), p(0, 0, 2))
	merged := topo.MergeBodies(topo.NewLineage(topo.Tok("imp", "body", 0)), false, one, two)
	if hits := SelfIntersections(merged, mesh.DefaultQuality()); len(hits) != 1 {
		t.Errorf("a doubled coincident wall reports %d self-intersection(s), want 1: %+v", len(hits), hits)
	}
}

// TestFacesShareOneSheetSeparatesCoplanarFromCrossing pins the arm SELECTOR: two quads on one plane
// are a same-sheet pair, and a quad meeting another at right angles is not.
func TestFacesShareOneSheetSeparatesCoplanarFromCrossing(t *testing.T) {
	t.Parallel()
	p := math.P3
	flat := brepfixture.QuadBody("a", p(0, 0, 0), p(2, 0, 0), p(2, 0, 2), p(0, 0, 2))
	same := brepfixture.QuadBody("b", p(1, 0, 1), p(3, 0, 1), p(3, 0, 3), p(1, 0, 3))
	cross := brepfixture.QuadBody("c", p(1, -1, 1), p(1, 1, 1), p(1, 1, 3), p(1, -1, 3))
	res := geom.ResolutionForSize(4)
	if !facesShareOneSheet(flat.Faces()[0], same.Faces()[0], res) {
		t.Error("two quads on the same plane must read as one surface sheet")
	}
	if facesShareOneSheet(flat.Faces()[0], cross.Faces()[0], res) {
		t.Error("two perpendicular quads must not read as one surface sheet")
	}
}

// TestFaceTrimProbesQuarterEveryEdge pins the probe set the trim-overlap and straddle rules are taken
// at: a quad yields its four corners and three interior points per edge, all on the face's own
// boundary. Two per edge samples a closed rim at only two angles — see edgeProbesPerEdge.
func TestFaceTrimProbesQuarterEveryEdge(t *testing.T) {
	t.Parallel()
	p := math.P3
	f := brepfixture.QuadBody("q", p(0, 0, 0), p(2, 0, 0), p(2, 0, 2), p(0, 0, 2)).Faces()[0]
	probes := faceTrimProbes(f)
	if len(probes) != 4*edgeProbesPerEdge {
		t.Fatalf("a four-edge face must yield %d probes, got %d", 4*edgeProbesPerEdge, len(probes))
	}
	for _, q := range probes {
		if d := distanceToFaceBoundary(f, q); d > 1e-12 { // tol:numeric — a probe is ON the boundary by construction
			t.Errorf("probe %v sits %g off the face's own boundary", q, d)
		}
	}
}

// TestEdgeCurvePointFollowsTheCurveNotTheChord: on a quarter arc the parameter midpoint is the
// point at 45°, which is 1 − cos(π/4) away from the chord midpoint the fallback would return.
func TestEdgeCurvePointFollowsTheCurveNotTheChord(t *testing.T) {
	t.Parallel()
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 1, 0, stdmath.Pi/2)
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("arc", "body", 0)))
	lin := topo.NewLineage(topo.Tok("arc", "x", 0))
	v0 := bld.AddVertex(arc.PointAt(0), lin)
	v1 := bld.AddVertex(arc.PointAt(1), lin)
	got := edgeCurvePointAt(bld.AddEdge(arc, v0, v1, lin), 0.5)
	want := arc.PointAt(0.5)
	if float64(got.DistanceTo(want)) > 1e-12 { // tol:numeric — the same evaluation twice
		t.Errorf("edge midpoint %v, want the curve's own mid-parameter point %v", got, want)
	}
}

// TestMidCrossingSampleReturnsTheMiddleOfTheAcceptedRun pins WHY the witness is the middle sample: a
// first-hit witness lands one sampling step past the rim of the crossing, where a caller cannot tell
// it from legitimate vertex contact (#1321). Accepting x ≥ 4 on a line through [0,8] must therefore
// answer near 6, not near 4.
func TestMidCrossingSampleReturnsTheMiddleOfTheAcceptedRun(t *testing.T) {
	t.Parallel()
	seg := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(8, 0, 0))
	box := math.NewBox(math.P3(0, -1, -1), math.P3(8, 1, 1))
	got, ok := midCrossingSample(seg, box, func(p math.Point3) bool { return float64(p.X) >= 4 })
	if !ok {
		t.Fatal("a run of accepted samples must produce a witness")
	}
	if x := float64(got.X); x < 5.5 || x > 6.5 {
		t.Errorf("witness x = %g, want the middle of the accepted run [4,8]", x)
	}
	if _, ok := midCrossingSample(seg, box, func(math.Point3) bool { return false }); ok {
		t.Error("no accepted sample must produce no witness")
	}
}
