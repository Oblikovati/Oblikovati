// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestCdtCoversLoopsCertifiesACleanTriangulation: a triangulation that tiles the domain exactly is
// certified, and one missing a triangle (a HOLE, the shape constraint-recovery failure leaves) or
// carrying an extra one (a LEAK past a missing wall) is not. Tolerance-free by construction: the
// shoelace area of a polygon is reproduced exactly by any triangulation that covers it.
func TestCdtCoversLoopsCertifiesACleanTriangulation(t *testing.T) {
	t.Parallel()
	// Unit square, split into two triangles.
	pts := [][2]float64{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	loops := [][]int{{0, 1, 2, 3}}
	full := [][3]int{{0, 1, 2}, {0, 2, 3}}
	if !cdtCoversLoops(pts, loops, full) {
		t.Error("a triangulation that tiles the square exactly must be certified")
	}
	if cdtCoversLoops(pts, loops, full[:1]) {
		t.Error("half the square left untriangulated is a HOLE and must not be certified")
	}
	if cdtCoversLoops(pts, loops, append(append([][3]int{}, full...), [3]int{0, 1, 2})) {
		t.Error("a doubled triangle over-covers and must not be certified")
	}
	if cdtCoversLoops(pts, loops, nil) {
		t.Error("an empty triangulation covers nothing")
	}
}

// TestCdtCoversLoopsSubtractsHoles pins that a hole loop is removed from the domain area, so a
// triangulation that correctly leaves the hole empty is certified while one that FILLS it (the
// classic leak past an unrecovered hole wall) is not.
func TestCdtCoversLoopsSubtractsHoles(t *testing.T) {
	t.Parallel()
	// A 4x4 square with a 2x2 hole, tiled by the 8 triangles of the surrounding ring.
	pts := [][2]float64{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {1, 1}, {3, 1}, {3, 3}, {1, 3}}
	loops := [][]int{{0, 1, 2, 3}, {4, 5, 6, 7}}
	ring := [][3]int{
		{0, 1, 5}, {0, 5, 4}, {1, 2, 6}, {1, 6, 5},
		{2, 3, 7}, {2, 7, 6}, {3, 0, 4}, {3, 4, 7},
	}
	if !cdtCoversLoops(pts, loops, ring) {
		t.Error("the ring around the hole tiles the domain exactly and must be certified")
	}
	if cdtCoversLoops(pts, loops, append(append([][3]int{}, ring...), [3]int{4, 5, 6}, [3]int{4, 6, 7})) {
		t.Error("filling the hole is a LEAK and must not be certified")
	}
}

// TestCoverageFloorScalesWithTheModel: the coverage bracket is model-relative (ADR-0042), so the SAME
// relative defect is caught on a µm-scale domain and on a 1000x one. An absolute epsilon floor would
// swamp the small case (#1610's regression) and this pins that it does not.
func TestCoverageFloorScalesWithTheModel(t *testing.T) {
	t.Parallel()
	for _, scale := range []float64{1e-3, 1, 1e3} {
		pts := [][2]float64{{0, 0}, {scale, 0}, {scale, scale}, {0, scale}}
		loops := [][]int{{0, 1, 2, 3}}
		full := [][3]int{{0, 1, 2}, {0, 2, 3}}
		if !cdtCoversLoops(pts, loops, full) {
			t.Errorf("scale %g: an exact tiling must be certified", scale)
		}
		if cdtCoversLoops(pts, loops, full[:1]) {
			t.Errorf("scale %g: a 50%% hole must be caught", scale)
		}
	}
}

// stepBandFace builds a cylindrical band with a STEP notch — complex/D8's fillet band in miniature.
// Its (u,v) trim is bounded by three straight axial rulings (each discretizing to just its two
// endpoints) and three arcs, so the trim's whole middle stretch carries NO boundary point in the
// angular direction: a boundary-only triangulation must span it with triangles that chord across the
// full arc. The step keeps it off isoRectangleGrid's structured-grid fast path.
//
// Returns the face and its EXACT area, R·(Δu·L − (Δu/2)·h) — closed form, no oracle needed.
func stepBandFace(t *testing.T, r, l, h float64) (*topo.Face, float64) {
	t.Helper()
	// Pin the angle-zero reference so the face's u agrees with the rim arcs' own angles.
	cyl, err := geom.NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r)
	if err != nil {
		t.Fatalf("NewCylinderWithRef: %v", err)
	}
	half := stdmath.Pi / 4
	corners := [][2]float64{{0, 0}, {half, 0}, {half, h}, {2 * half, h}, {2 * half, l}, {0, l}}
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("c", "body", 0)))
	lin := topo.NewLineage(topo.Tok("c", "x", 0))
	verts := make([]*topo.Vertex, len(corners))
	for i, c := range corners {
		verts[i] = bld.AddVertex(cyl.PointAt(c[0], c[1]), lin)
	}
	uses := make([]topo.Use, len(corners))
	for i, c := range corners {
		j := (i + 1) % len(corners)
		uses[i] = topo.Fwd(bld.AddEdge(bandEdgeCurve(t, cyl, c, corners[j]), verts[i], verts[j], lin))
	}
	bld.AddFace(cyl, lin, topo.OuterLoop(uses...))
	return bld.Build().Faces()[0], r * (2*half*l - half*h)
}

// bandEdgeCurve is the exact curve between two (u,v) corners of a cylindrical band: a circular arc
// when they share v (an iso-v rim) and a straight axial ruling when they share u.
func bandEdgeCurve(t *testing.T, cyl geom.Cylinder, a, b [2]float64) geom.Curve3 {
	t.Helper()
	if a[0] == b[0] {
		return geom.NewLineSegment(cyl.PointAt(a[0], a[1]), cyl.PointAt(b[0], b[1]))
	}
	centre := math.P3(0, 0, math.Scalar(a[1]))
	arc, err := geom.NewArc3d(centre, math.V3(0, 0, 1), math.V3(1, 0, 0), cyl.Radius, a[0], b[0]-a[0])
	if err != nil {
		t.Fatalf("NewArc3d: %v", err)
	}
	return arc
}

// TestConformingRemeshFollowsTheCylinderInsteadOfChordingIt is the falsifiable guard for the
// conformance re-mesh's INTERIOR refinement (bestConformingPatch).
//
// A boundary-only triangulation puts a node only where the boundary has one. On a band bounded by
// STRAIGHT axial rulings — which discretize to two points each — that leaves the whole middle of the
// face with no node in the angular direction, so single triangles span the full arc and realise it as
// their CHORD: 2·sin(Δu/2)/Δu of the true area, i.e. −10% at 90°. complex/D8's r=30 fillet band lost
// exactly this way (21339.83 tiled against a closed form of 23340.06, −8.57%), and that one face was
// the whole of that case's shipped-vs-closed-form area gap.
//
// The re-mesh must therefore tile the band's CLOSED FORM to the chordal tolerance it was asked for.
// Falsify by restoring the boundary-only triangulation in bestConformingPatch: this goes RED at ~−10%.
func TestConformingRemeshFollowsTheCylinderInsteadOfChordingIt(t *testing.T) {
	t.Parallel()
	f, exact := stepBandFace(t, 30, 460, 5)
	q := PropertyQuality()
	m := conformingCylConeMesh(f, q)
	if m == nil {
		t.Fatal("conformingCylConeMesh declined a well-formed cylindrical band")
	}
	got := validate.MeshArea(m)
	if rel := (got - exact) / exact; rel < -1e-4 || rel > 1e-4 { // tol:numeric (relative area fraction)
		t.Errorf("conformance re-mesh tiles %.6g of the band, closed form %.6g (rel %+.4f%%) — the "+
			"triangulation is chording across the cylinder instead of following it", got, exact, rel*100)
	}
	if n := validate.FoldEdgeCount(m); n != 0 {
		t.Errorf("conformance re-mesh folds on %d edge(s); a developable metric (u,v) must not fold", n)
	}
}

// TestConformingRemeshKeepsEveryBoundarySegment: interior refinement must not cost the re-mesh the one
// property it exists for — every boundary segment stays a triangle edge, so the face still conforms to
// its neighbour. Guards against "fixed the area, re-opened the crack".
func TestConformingRemeshKeepsEveryBoundarySegment(t *testing.T) {
	t.Parallel()
	f, _ := stepBandFace(t, 30, 460, 5)
	q := PropertyQuality()
	m := conformingCylConeMesh(f, q)
	if m == nil {
		t.Fatal("conformingCylConeMesh declined a well-formed cylindrical band")
	}
	ring := faceOuterBoundary(f, q)
	for i := range ring {
		a, b := ring[i], ring[(i+1)%len(ring)]
		if !meshHasSegment(m, a, b) {
			t.Fatalf("boundary segment %d (%v→%v) is not an edge of the re-mesh — the face would crack", i, a, b)
		}
	}
}
