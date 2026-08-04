// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	gm "oblikovati.org/math"
)

// unitSquareMesh is a 1×1 square in the z=0 plane split into two triangles. consistentWinding
// winds both triangles the same way (a clean patch); when false the second triangle is wound
// backwards, so its geometric normal opposes the first across the shared diagonal — a fold.
func unitSquareMesh(consistentWinding bool) *Mesh {
	m := &Mesh{
		Positions: []gm.Point3{gm.P3(0, 0, 0), gm.P3(1, 0, 0), gm.P3(1, 1, 0), gm.P3(0, 1, 0)},
		Normals:   []gm.Vector3{gm.V3(0, 0, 1), gm.V3(0, 0, 1), gm.V3(0, 0, 1), gm.V3(0, 0, 1)},
	}
	if consistentWinding {
		m.Indices = []int{0, 1, 2, 0, 2, 3}
	} else {
		m.Indices = []int{0, 1, 2, 0, 3, 2} // second triangle reversed → normal flips
	}
	return m
}

// TestFoldEdgeCountCleanMesh: a consistently wound patch has no folds.
func TestFoldEdgeCountCleanMesh(t *testing.T) {
	if got := FoldEdgeCount(unitSquareMesh(true)); got != 0 {
		t.Fatalf("clean mesh FoldEdgeCount = %d, want 0", got)
	}
	if got := FoldEdges(unitSquareMesh(true)); len(got) != 0 {
		t.Fatalf("clean mesh FoldEdges = %v, want none", got)
	}
}

// TestFoldEdgeCountDetectsFold: the reversed triangle folds across the shared diagonal (vertices
// 0 and 2), and the detector reports exactly that one interior edge.
func TestFoldEdgeCountDetectsFold(t *testing.T) {
	m := unitSquareMesh(false)
	if got := FoldEdgeCount(m); got != 1 {
		t.Fatalf("folded mesh FoldEdgeCount = %d, want 1", got)
	}
	edges := FoldEdges(m)
	if len(edges) != 1 || edges[0] != [2]int{0, 2} {
		t.Fatalf("folded mesh FoldEdges = %v, want [[0 2]] (the shared diagonal)", edges)
	}
}

// TestMeshAreaSumsTriangles: MeshArea is the true surface area, independent of winding (it sums
// |triangle| areas), so both the clean and folded unit squares measure 1.0.
func TestMeshAreaSumsTriangles(t *testing.T) {
	for _, consistent := range []bool{true, false} {
		if got := MeshArea(unitSquareMesh(consistent)); math.Abs(got-1.0) > 1e-12 {
			t.Errorf("MeshArea(consistent=%v) = %.12f, want 1.0", consistent, got)
		}
	}
}

// TestFoldEdgeCountOnCleanCurvedSolid: a clean analytic cylinder solid tessellates fold-free, the
// production prerequisite the detector guards (it pairs with the watertight free-edge guard).
func TestFoldEdgeCountOnCleanCurvedSolid(t *testing.T) {
	cyl, err := brep.SolidCylinder(gm.P3(0, 0, 0), gm.V3(0, 0, 1), 5, 10)
	if err != nil {
		t.Fatalf("build cylinder: %v", err)
	}
	for _, gq := range gateQualities() {
		mesh, _ := TessellateBody(cyl, gq.q)
		if got := FoldEdgeCount(mesh); got != 0 {
			t.Errorf("%s quality: clean cylinder tessellated with %d fold edges; want 0", gq.name, got)
		}
	}
}

// TestFoldDetectorIgnoresNullSliver pins BOTH directions of the degenerate-normal guard added for the
// macOS/arm64 Z1 failure (PR #2013): a null sliver beside a real triangle is not a fold, while two
// well-formed opposed triangles still are. Reproduces the measured arm64 geometry — a 1.24e-17-area
// triangle adjacent to a 0.0937-area one on a face of 234 — so the guard cannot be widened into a
// blanket "ignore small triangles" without this failing.
func TestFoldDetectorIgnoresNullSliver(t *testing.T) {
	sliver := &Mesh{
		Positions: []gm.Point3{
			gm.P3(9.807852804032304, 1.9509032201612806, 20),
			gm.P3(9.903926402016152, 0.9754516100806394, 20),
			gm.P3(9.5, 1.5, 20),
			// Collinear with the shared edge to the last bit: area ~1e-17, normal direction is noise.
			gm.P3(9.855889603024228, 1.46317741512096, 20),
		},
		Normals: []gm.Vector3{gm.V3(0, 0, 1), gm.V3(0, 0, 1), gm.V3(0, 0, 1), gm.V3(0, 0, 1)},
		Indices: []int{0, 1, 2, 0, 3, 1},
	}
	if n := FoldEdgeCount(sliver); n != 0 {
		t.Fatalf("null sliver counted as %d fold edge(s); a triangle of ~1e-17 area has no orientation "+
			"to oppose with — its normal is decided by the last ulp", n)
	}

	// Two well-formed triangles folded about the shared edge: the guard must NOT hide this.
	folded := &Mesh{
		Positions: []gm.Point3{
			gm.P3(0, 0, 0), gm.P3(1, 0, 0), gm.P3(0.5, 1, 0), gm.P3(0.5, 1, 0.05),
		},
		Normals: []gm.Vector3{gm.V3(0, 0, 1), gm.V3(0, 0, 1), gm.V3(0, 0, 1), gm.V3(0, 0, 1)},
		Indices: []int{0, 1, 2, 1, 0, 3},
	}
	if FoldEdgeCount(folded) == 0 {
		t.Fatal("a genuine fold between two well-formed triangles was not detected — the degenerate-normal " +
			"guard must narrow what counts as EVIDENCE, never narrow the gate itself")
	}
}
