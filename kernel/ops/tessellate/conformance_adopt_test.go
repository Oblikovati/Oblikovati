// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/math"
)

// squareLoop is the 10×10 planar face the adoption tests measure on: boundary length 40, area 100.
func squareLoop() []math.Point3 {
	return []math.Point3{math.P3(0, 0, 0), math.P3(10, 0, 0), math.P3(10, 10, 0), math.P3(0, 10, 0)}
}

// TestFaceBoundaryLengthWalksTheWholeRing pins that the ring's CLOSING segment is counted (a square
// of side 10 measures 40, not 30) — the slack is a perimeter, and dropping the closing segment would
// understate it on a triangle by a third.
func TestFaceBoundaryLengthWalksTheWholeRing(t *testing.T) {
	t.Parallel()
	f := planarFaceFromLoop(t, squareLoop())
	if got := faceBoundaryLength(f, DefaultQuality()); stdmath.Abs(got-40) > 1e-9 {
		t.Errorf("faceBoundaryLength = %.9g, want 40 (the closed 10×10 ring)", got)
	}
}

// TestConformAreaSlackScalesWithToleranceAndSize pins the slack's derivation: it is the chordal
// tolerance times the boundary length, so it is an AREA that tracks both the mesh's accuracy contract
// and the model's size. A fixed epsilon would be wrong at either end (ADR-0042).
func TestConformAreaSlackScalesWithToleranceAndSize(t *testing.T) {
	t.Parallel()
	f := planarFaceFromLoop(t, squareLoop())
	fine := conformAreaSlack(f, Quality{ChordTolerance: 1e-3, AngleTolerance: 1})
	coarse := conformAreaSlack(f, Quality{ChordTolerance: 1e-1, AngleTolerance: 1})
	if stdmath.Abs(fine-1e-3*40) > 1e-12 {
		t.Errorf("conformAreaSlack at tol 1e-3 = %.9g, want %.9g", fine, 1e-3*40)
	}
	if stdmath.Abs(coarse/fine-100) > 1e-9 {
		t.Errorf("slack ratio coarse/fine = %.9g, want 100 (linear in the chordal tolerance)", coarse/fine)
	}
}

// TestConformingMeshIsFaithfulRejectsAreaLoss is the white-box statement of the adoption criterion,
// one arm per branch. The rejected case is complex/D8's shape of defect in miniature: a fold-free
// re-mesh that tiles far less of the face than the mesh it would replace.
func TestConformingMeshIsFaithfulRejectsAreaLoss(t *testing.T) {
	t.Parallel()
	f := planarFaceFromLoop(t, squareLoop())
	q := Quality{ChordTolerance: 1e-3, AngleTolerance: 1} // slack = 1e-3 × 40 = 0.04
	full := quadMesh(10, 10)
	cases := []struct {
		name       string
		remesh     *Mesh
		wantAdopt  bool
		wantReason string
	}{
		{"same area", quadMesh(10, 10), true, "an area-neutral re-tiling is the repair's normal case"},
		{"gains area", quadMesh(10, 11), true, "more area is closer to the truth for inscribed meshes"},
		{"loses within slack", quadMesh(10, 10-0.003), true, "0.03 < the 0.04 boundary-re-discretization bound"},
		{"loses past slack", quadMesh(10, 10-0.01), false, "0.1 > 0.04: not a re-discretization, a hole"},
		{"loses 39%", quadMesh(10, 6.1), false, "complex/D8's corner cylinder, in miniature"},
	}
	for _, c := range cases {
		if got := conformingMeshIsFaithful(c.remesh, full, f, q); got != c.wantAdopt {
			t.Errorf("%s: adopt = %v, want %v (%s)", c.name, got, c.wantAdopt, c.wantReason)
		}
	}
}

// TestConformingMeshIsFaithfulStillRejectsAddedFolds keeps the ORIGINAL fold arm live: an area-GROWING
// re-mesh that grows by FOLDING must still be refused (I3's host cones grew 30659 → 59056 on 4 folds).
func TestConformingMeshIsFaithfulStillRejectsAddedFolds(t *testing.T) {
	t.Parallel()
	f := planarFaceFromLoop(t, squareLoop())
	q := Quality{ChordTolerance: 1e-3, AngleTolerance: 1}
	folded := foldedQuadMesh()
	if validate.FoldEdgeCount(folded) == 0 {
		t.Fatal("fixture does not fold — the fold arm would be untested")
	}
	if validate.MeshArea(folded) <= validate.MeshArea(quadMesh(10, 10)) {
		t.Fatalf("folded fixture area %.6g does not exceed the flat quad's %.6g — the area arm would mask the fold arm",
			validate.MeshArea(folded), validate.MeshArea(quadMesh(10, 10)))
	}
	if conformingMeshIsFaithful(folded, quadMesh(10, 10), f, q) {
		t.Error("adopted a re-mesh that ADDS folds — the fold arm regressed")
	}
}

// quadMesh is a flat w×h rectangle on z=0 as two triangles (area w·h).
func quadMesh(w, h float64) *Mesh {
	m := &Mesh{}
	n := math.V3(0, 0, 1)
	m.AddVertex(math.P3(0, 0, 0), n)
	m.AddVertex(math.P3(w, 0, 0), n)
	m.AddVertex(math.P3(w, h, 0), n)
	m.AddVertex(math.P3(0, h, 0), n)
	m.AddTriangle(0, 1, 2)
	m.AddTriangle(0, 2, 3)
	return m
}

// foldedQuadMesh is a flat 10×10 quad plus a large flap hinged on its 0–2 diagonal and creased back
// over it — one folding interior edge, and MORE area than the flat quad (the I3 shape of defect).
func foldedQuadMesh() *Mesh {
	m := &Mesh{}
	n := math.V3(0, 0, 1)
	for _, p := range []math.Point3{math.P3(0, 0, 0), math.P3(10, 0, 0), math.P3(10, 10, 0), math.P3(5, 30, 0.5)} {
		m.AddVertex(p, n)
	}
	m.AddTriangle(0, 1, 2) // geometric normal +z
	m.AddTriangle(1, 0, 3) // hinged on edge 0–1, creased back: normal ≈ −z, area 150 (total 200 > 100)
	return m
}
