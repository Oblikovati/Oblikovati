// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "testing"

func TestMergeFacesBox(t *testing.T) {
	// A box's soup is 12 triangles (2 per planar face); merging must recover the 6
	// square faces, each a 4-vertex outer loop with no holes.
	faces := MergeFaces(boxMesh([3]float64{0, 0, 0}, [3]float64{2, 2, 2}))
	if len(faces) != 6 {
		t.Fatalf("box merged to %d faces, want 6", len(faces))
	}
	for i, f := range faces {
		if len(f.Outer) != 4 {
			t.Fatalf("face %d has %d outer vertices, want 4", i, len(f.Outer))
		}
		if len(f.Holes) != 0 {
			t.Fatalf("face %d has %d holes, want 0", i, len(f.Holes))
		}
	}
}

func TestMergeFacesBooleanResultAllPlanar(t *testing.T) {
	// Every face of a box-union result merges to a planar quad (the offset boxes
	// produce only rectangular sub-faces): 0 holes and >=4 vertices each.
	a := boxMesh([3]float64{0, 0, 0}, [3]float64{2, 2, 2})
	b := boxMesh([3]float64{1, 1, 1}, [3]float64{3, 3, 3})
	faces := MergeFaces(Boolean(a, b, Union))
	if len(faces) == 0 {
		t.Fatal("no merged faces")
	}
	for i, f := range faces {
		if len(f.Outer) < 3 {
			t.Fatalf("face %d outer loop has %d vertices", i, len(f.Outer))
		}
	}
}

func TestMergeFacesWithHole(t *testing.T) {
	// An annular region in z=0 (outer square with a square hole) must merge to one
	// face with one hole loop.
	faces := MergeFaces(annulusSoup())
	if len(faces) != 1 {
		t.Fatalf("annulus merged to %d faces, want 1", len(faces))
	}
	f := faces[0]
	if len(f.Outer) != 4 {
		t.Fatalf("outer loop has %d vertices, want 4", len(f.Outer))
	}
	if len(f.Holes) != 1 || len(f.Holes[0]) != 4 {
		t.Fatalf("holes = %d loops (first with %d verts), want one 4-vertex hole", len(f.Holes), holeLen(f))
	}
}

func holeLen(f Face) int {
	if len(f.Holes) == 0 {
		return 0
	}
	return len(f.Holes[0])
}

// annulusSoup triangulates the region between an outer 6x6 square and an inner 2x2
// square hole (both in z=0) into 8 CCW triangles.
func annulusSoup() [][3]Point {
	a := [4][3]float64{{0, 0, 0}, {6, 0, 0}, {6, 6, 0}, {0, 6, 0}}
	b := [4][3]float64{{2, 2, 0}, {4, 2, 0}, {4, 4, 0}, {2, 4, 0}}
	var soup [][3]Point
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		soup = append(soup,
			[3]Point{pt(a[i]), pt(a[j]), pt(b[j])},
			[3]Point{pt(a[i]), pt(b[j]), pt(b[i])},
		)
	}
	return soup
}
