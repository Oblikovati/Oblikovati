// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"testing"
)

func seg(a, b [3]float64) [2]Point { return [2]Point{pt(a), pt(b)} }

func TestRefineFaceSingleChord(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{10, 0, 0}, [3]float64{0, 10, 0})
	segments := [][2]Point{seg([3]float64{5, 0, 0}, [3]float64{0, 5, 0})} // boundary-to-boundary chord
	tris := RefineFace(face, segments)
	assertFaceConformed(t, face, tris, segments)
}

func TestRefineFaceTwoCrossing(t *testing.T) {
	face := tri([3]float64{0, 0, 0}, [3]float64{10, 0, 0}, [3]float64{0, 10, 0})
	segments := [][2]Point{
		seg([3]float64{5, 0, 0}, [3]float64{0, 5, 0}), // x+y=5
		seg([3]float64{0, 0, 0}, [3]float64{5, 5, 0}), // y=x, from the corner
	}
	tris := RefineFace(face, segments)
	assertFaceConformed(t, face, tris, segments)

	cross := pt([3]float64{2.5, 2.5, 0}) // the exact crossing, must be a vertex
	if !meshHasVertex(tris, cross) {
		t.Fatal("the segment crossing point (2.5,2.5) is not a triangulation vertex")
	}
}

func TestRefineFaceTiltedPlane(t *testing.T) {
	// Same topology on a non-axis-aligned face, exercising planeAxis in the driver.
	face := tri([3]float64{0, 0, 0}, [3]float64{6, 0, 3}, [3]float64{0, 6, 3})
	// Two points on distinct edges, giving an interior chord in the face plane.
	segments := [][2]Point{seg([3]float64{3, 0, 1.5}, [3]float64{0, 3, 1.5})}
	tris := RefineFace(face, segments)
	assertFaceConformed(t, face, tris, segments)
}

func TestRefineFaceSegmentAlongEdge(t *testing.T) {
	// A constraint lying along the face's own bottom edge: the face corners are
	// collinear with it but beyond its endpoints, exercising the range guard in
	// verticesOnSegment.
	face := tri([3]float64{0, 0, 0}, [3]float64{10, 0, 0}, [3]float64{0, 10, 0})
	segments := [][2]Point{seg([3]float64{2, 0, 0}, [3]float64{8, 0, 0})}
	tris := RefineFace(face, segments)
	assertFaceConformed(t, face, tris, segments)
	if !meshHasVertex(tris, pt([3]float64{2, 0, 0})) || !meshHasVertex(tris, pt([3]float64{8, 0, 0})) {
		t.Fatal("edge-collinear constraint endpoints were not inserted as vertices")
	}
}

func TestRefineFacePreservesOrientation(t *testing.T) {
	// A face wound clockwise in its projection: every refined sub-triangle must keep
	// that same (input) orientation, not the internal CCW-normalized one.
	face := tri([3]float64{0, 0, 0}, [3]float64{0, 4, 0}, [3]float64{4, 0, 0})
	inSign := orient2(face[0], face[1], face[2], 2)
	if inSign >= 0 {
		t.Fatal("test face is not clockwise in the xy projection")
	}
	tris := RefineFace(face, [][2]Point{seg([3]float64{2, 0, 0}, [3]float64{0, 2, 0})})
	for _, tt := range tris {
		if orient2(tt[0], tt[1], tt[2], 2) != inSign {
			t.Fatalf("refined triangle orientation %d != input %d", orient2(tt[0], tt[1], tt[2], 2), inSign)
		}
	}
}

// assertFaceConformed checks the refined triangulation tiles the face exactly (area
// conserved) and no triangle edge properly crosses any constraint — i.e. every
// constraint is covered by triangulation edges.
func assertFaceConformed(t *testing.T, face [3]Point, tris [][3]Point, segments [][2]Point) {
	t.Helper()
	axis := planeAxis(face)
	want := new(big.Rat).Abs(orient2Val(face[0], face[1], face[2], axis))
	sum := new(big.Rat)
	for _, tt := range tris {
		d := orient2Val(tt[0], tt[1], tt[2], axis)
		if d.Sign() <= 0 {
			t.Fatalf("degenerate or clockwise triangle in refined face")
		}
		sum.Add(sum, d)
		for e := 0; e < 3; e++ {
			p, q := tt[e], tt[(e+1)%3]
			for _, s := range segments {
				if segmentsProperlyCross(p, q, s[0], s[1], axis) {
					t.Fatalf("triangle edge (%v,%v) crosses a constraint segment", p.Round(), q.Round())
				}
			}
		}
	}
	if sum.Cmp(want) != 0 {
		t.Fatalf("area not conserved: got %s, want %s", sum.RatString(), want.RatString())
	}
}

func meshHasVertex(tris [][3]Point, v Point) bool {
	for _, tt := range tris {
		for _, p := range tt {
			if p.Equal(v) {
				return true
			}
		}
	}
	return false
}
