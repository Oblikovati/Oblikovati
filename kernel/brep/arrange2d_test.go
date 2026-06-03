// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// seg is a terse segment constructor for tests.
func seg(x0, y0, x1, y1 float64) [2]math.Point2 {
	return [2]math.Point2{math.P2(x0, y0), math.P2(x1, y1)}
}

// squareSegs returns the four edges of an axis-aligned square [x0,x1]×[y0,y1].
func squareSegs(x0, y0, x1, y1 float64) [][2]math.Point2 {
	return [][2]math.Point2{
		seg(x0, y0, x1, y0), seg(x1, y0, x1, y1), seg(x1, y1, x0, y1), seg(x0, y1, x0, y0),
	}
}

// faceAreas returns the absolute outer-loop areas of the faces, sorted ascending.
func faceAreas(faces []Face2D) []float64 {
	out := make([]float64, len(faces))
	for i, f := range faces {
		out[i] = stdmath.Abs(signedArea2D(f.Outer))
	}
	sort.Float64s(out)
	return out
}

func TestArrangeSingleSquare(t *testing.T) {
	faces := Arrange(squareSegs(0, 0, 2, 2))
	if len(faces) != 1 {
		t.Fatalf("square → %d faces, want 1", len(faces))
	}
	if a := stdmath.Abs(signedArea2D(faces[0].Outer)); stdmath.Abs(a-4) > 1e-9 {
		t.Errorf("square area = %g, want 4", a)
	}
	if len(faces[0].Holes) != 0 {
		t.Errorf("square has %d holes, want 0", len(faces[0].Holes))
	}
}

func TestArrangeChordSplitsInTwo(t *testing.T) {
	segs := append(squareSegs(0, 0, 2, 2), seg(1, 0, 1, 2)) // vertical mid chord
	faces := Arrange(segs)
	if len(faces) != 2 {
		t.Fatalf("square+chord → %d faces, want 2", len(faces))
	}
	for _, a := range faceAreas(faces) {
		if stdmath.Abs(a-2) > 1e-9 {
			t.Errorf("half area = %g, want 2", a)
		}
	}
}

func TestArrangePlusSplitsInFour(t *testing.T) {
	segs := squareSegs(0, 0, 2, 2)
	segs = append(segs, seg(1, 0, 1, 2), seg(0, 1, 2, 1)) // a full cross
	faces := Arrange(segs)
	if len(faces) != 4 {
		t.Fatalf("square+cross → %d faces, want 4", len(faces))
	}
	for _, a := range faceAreas(faces) {
		if stdmath.Abs(a-1) > 1e-9 {
			t.Errorf("quarter area = %g, want 1", a)
		}
	}
}

func TestArrangeNestedSquareIsHoleAndFace(t *testing.T) {
	// A big square with a disjoint small square inside → the annulus (big with a hole)
	// plus the inner square as its own face.
	segs := append(squareSegs(0, 0, 6, 6), squareSegs(2, 2, 4, 4)...)
	faces := Arrange(segs)
	if len(faces) != 2 {
		t.Fatalf("nested squares → %d faces, want 2", len(faces))
	}
	var annulus, inner *Face2D
	for i := range faces {
		if stdmath.Abs(signedArea2D(faces[i].Outer)) > 20 {
			annulus = &faces[i]
		} else {
			inner = &faces[i]
		}
	}
	if annulus == nil || inner == nil {
		t.Fatal("expected one big (annulus) and one small (inner) face")
	}
	if len(annulus.Holes) != 1 {
		t.Errorf("annulus has %d holes, want 1", len(annulus.Holes))
	}
	if a := stdmath.Abs(signedArea2D(inner.Outer)); stdmath.Abs(a-4) > 1e-9 {
		t.Errorf("inner face area = %g, want 4", a)
	}
}
