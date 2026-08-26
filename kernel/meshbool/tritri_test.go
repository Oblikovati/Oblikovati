// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"math/rand"
	"testing"
)

func tri(pts ...[3]float64) [3]Point {
	return [3]Point{pt(pts[0]), pt(pts[1]), pt(pts[2])}
}

func TestIntersectTrianglesKnownCrossing(t *testing.T) {
	// t1 in z=0, t2 in x=0; planes meet on the y-axis. t1 spans y∈[-2,2] there,
	// t2 spans y∈[-1,1]; the overlap is the segment (0,-1,0)-(0,1,0).
	t1 := tri([3]float64{-2, -2, 0}, [3]float64{2, -2, 0}, [3]float64{0, 2, 0})
	t2 := tri([3]float64{0, -1, -2}, [3]float64{0, -1, 2}, [3]float64{0, 1, 0})
	got := IntersectTriangles(t1, t2)
	if got.Kind != Crossing {
		t.Fatalf("kind: got %d, want Crossing", got.Kind)
	}
	want := []Point{pt([3]float64{0, -1, 0}), pt([3]float64{0, 1, 0})}
	if !sameEndpoints(got, want) {
		t.Fatalf("segment {%v,%v} != want (0,-1,0)-(0,1,0)", got.P.Round(), got.Q.Round())
	}
}

func TestIntersectTrianglesDisjoint(t *testing.T) {
	t1 := tri([3]float64{0, 0, 0}, [3]float64{2, 0, 0}, [3]float64{0, 2, 0})
	t2 := tri([3]float64{0, 0, 5}, [3]float64{1, 0, 5}, [3]float64{0, 1, 5})
	if got := IntersectTriangles(t1, t2); got.Kind != Disjoint {
		t.Fatalf("kind: got %d, want Disjoint", got.Kind)
	}
}

func TestIntersectTrianglesCoplanar(t *testing.T) {
	t1 := tri([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0})
	t2 := tri([3]float64{1, 1, 0}, [3]float64{3, 1, 0}, [3]float64{1, 3, 0})
	if got := IntersectTriangles(t1, t2); got.Kind != Coplanar {
		t.Fatalf("kind: got %d, want Coplanar", got.Kind)
	}
}

func TestIntersectTrianglesTouching(t *testing.T) {
	// t2's crossing span on the y-axis starts exactly where t1's ends (y=2).
	t1 := tri([3]float64{-2, -2, 0}, [3]float64{2, -2, 0}, [3]float64{0, 2, 0})
	t2 := tri([3]float64{0, 2, 0}, [3]float64{0, 4, -1}, [3]float64{0, 4, 1})
	got := IntersectTriangles(t1, t2)
	if got.Kind != Touching {
		t.Fatalf("kind: got %d, want Touching", got.Kind)
	}
	if !got.P.Equal(pt([3]float64{0, 2, 0})) || !got.P.Equal(got.Q) {
		t.Fatalf("touch point: got %v (Q=%v), want (0,2,0)", got.P.Round(), got.Q.Round())
	}
}

// TestIntersectTrianglesProperties drives random small-box triangle pairs and
// checks the exact geometric contract of every Crossing/Touching result: both
// endpoints lie on both triangles' planes AND inside both triangles, and a
// Crossing has P!=Q while a Touching has P==Q. Ground truth is exact rational
// containment, independent of the segment-construction path.
func TestIntersectTrianglesProperties(t *testing.T) {
	r := rand.New(rand.NewSource(0x1c03))
	crossings, touches := 0, 0
	for i := range 30000 {
		t1, t2 := smallTri(r), smallTri(r)
		if degenerate(t1) || degenerate(t2) {
			continue
		}
		got := IntersectTriangles(t1, t2)
		switch got.Kind {
		case Crossing:
			crossings++
			if got.P.Equal(got.Q) {
				t.Fatalf("case %d: Crossing with P==Q", i)
			}
			assertOnBothInBoth(t, i, t1, t2, got.P)
			assertOnBothInBoth(t, i, t1, t2, got.Q)
		case Touching:
			touches++
			if !got.P.Equal(got.Q) {
				t.Fatalf("case %d: Touching with P!=Q", i)
			}
			assertOnBothInBoth(t, i, t1, t2, got.P)
		}
	}
	if crossings == 0 || touches == 0 {
		t.Fatalf("insufficient regime coverage: crossings=%d touches=%d", crossings, touches)
	}
	t.Logf("crossings=%d touches=%d", crossings, touches)
}

// TestIntersectTrianglesSymmetric checks IntersectTriangles(t1,t2) and (t2,t1)
// agree in kind and endpoint set — the intersection is a property of the pair.
func TestIntersectTrianglesSymmetric(t *testing.T) {
	r := rand.New(rand.NewSource(0x1c04))
	for i := range 30000 {
		t1, t2 := smallTri(r), smallTri(r)
		if degenerate(t1) || degenerate(t2) {
			continue
		}
		a := IntersectTriangles(t1, t2)
		b := IntersectTriangles(t2, t1)
		if a.Kind != b.Kind {
			t.Fatalf("case %d: kind %d vs %d", i, a.Kind, b.Kind)
		}
		if a.Kind == Crossing && !sameEndpoints(a, []Point{b.P, b.Q}) {
			t.Fatalf("case %d: asymmetric segment", i)
		}
	}
}

// --- helpers ---

func assertOnBothInBoth(t *testing.T, i int, t1, t2 [3]Point, p Point) {
	t.Helper()
	if !onPlane(t1, p) || !onPlane(t2, p) {
		t.Fatalf("case %d: endpoint %v not on both planes", i, p.Round())
	}
	if !inTriRat(t1, p) || !inTriRat(t2, p) {
		t.Fatalf("case %d: endpoint %v not inside both triangles", i, p.Round())
	}
}

func sameEndpoints(res TriTriResult, want []Point) bool {
	return (res.P.Equal(want[0]) && res.Q.Equal(want[1])) ||
		(res.P.Equal(want[1]) && res.Q.Equal(want[0]))
}

func smallTri(r *rand.Rand) [3]Point {
	f := func() [3]float64 {
		return [3]float64{float64(r.Intn(7) - 3), float64(r.Intn(7) - 3), float64(r.Intn(7) - 3)}
	}
	return tri(f(), f(), f())
}

func degenerate(t [3]Point) bool {
	n := triNormal(t)
	return n[0].Sign() == 0 && n[1].Sign() == 0 && n[2].Sign() == 0
}

// inTriRat reports whether p (assumed on plane(tri)) is inside or on triangle tri,
// via exact rational edge orientations projected onto the dominant axis.
func inTriRat(tri [3]Point, p Point) bool {
	axis := domAxisR(tri)
	o1 := rorient2(tri[0], tri[1], p, axis)
	o2 := rorient2(tri[1], tri[2], p, axis)
	o3 := rorient2(tri[2], tri[0], p, axis)
	neg := o1 < 0 || o2 < 0 || o3 < 0
	pos := o1 > 0 || o2 > 0 || o3 > 0
	if neg && pos {
		return false
	}
	return true
}

func TestAppendUniqueDedup(t *testing.T) {
	p := pt([3]float64{1, 2, 3})
	pts := appendUnique(nil, p)
	pts = appendUnique(pts, pt([3]float64{1, 2, 3})) // equal position → not appended
	pts = appendUnique(pts, pt([3]float64{1, 2, 4}))
	if len(pts) != 2 {
		t.Fatalf("appendUnique kept %d points, want 2 (duplicate must be dropped)", len(pts))
	}
}

func rorient2(a, b, p Point, axis int) int {
	au, av := dropR(a, axis)
	bu, bv := dropR(b, axis)
	pu, pv := dropR(p, axis)
	left := new(big.Rat).Mul(new(big.Rat).Sub(au, pu), new(big.Rat).Sub(bv, pv))
	right := new(big.Rat).Mul(new(big.Rat).Sub(av, pv), new(big.Rat).Sub(bu, pu))
	return left.Sub(left, right).Sign()
}

func dropR(p Point, axis int) (u, v *big.Rat) {
	switch axis {
	case 0:
		return p.Y, p.Z
	case 1:
		return p.X, p.Z
	default:
		return p.X, p.Y
	}
}

func domAxisR(t [3]Point) int {
	n := triNormal(t)
	ax := new(big.Rat).Abs(n[0])
	ay := new(big.Rat).Abs(n[1])
	az := new(big.Rat).Abs(n[2])
	if ax.Cmp(ay) >= 0 && ax.Cmp(az) >= 0 {
		return 0
	}
	if ay.Cmp(az) >= 0 {
		return 1
	}
	return 2
}
