// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"math/rand"
	"testing"

	"oblikovati.org/kernel/predicates"
)

// TestOrient2MatchesPredicates cross-checks the projected rational orient2 against
// predicates.Orient2D on the same projected coordinates, across all three drop
// axes, in both the general and near-collinear regimes.
func TestOrient2MatchesPredicates(t *testing.T) {
	r := rand.New(rand.NewSource(0x1c05))
	for i := 0; i < 20000; i++ {
		a, b := rc(r), rc(r)
		var c [3]float64
		if i%2 == 0 {
			s := r.Float64() // on line a-b → collinear/near-collinear regime
			for k := 0; k < 3; k++ {
				c[k] = a[k] + s*(b[k]-a[k])
			}
		} else {
			c = rc(r)
		}
		for axis := 0; axis < 3; axis++ {
			au, av := projF(a, axis)
			bu, bv := projF(b, axis)
			cu, cv := projF(c, axis)
			want := predicates.Orient2D(au, av, bu, bv, cu, cv)
			if got := orient2(pt(a), pt(b), pt(c), axis); got != want {
				t.Fatalf("case %d axis %d: orient2=%d, predicates=%d", i, axis, got, want)
			}
		}
	}
}

func TestOrient2KnownCCW(t *testing.T) {
	a := pt([3]float64{0, 0, 0})
	b := pt([3]float64{1, 0, 0})
	c := pt([3]float64{0, 1, 0})
	if got := orient2(a, b, c, 2); got != 1 { // CCW in the xy projection
		t.Fatalf("CCW: got %d, want +1", got)
	}
	if got := orient2(a, c, b, 2); got != -1 {
		t.Fatalf("CW: got %d, want -1", got)
	}
}

// TestSegSegCrossExact generates crossing coplanar segment pairs (in the xy plane
// and in a tilted integer plane) and verifies the constructed crossing lies
// exactly on both segments and, for the tilted plane, exactly on the plane; the
// two argument orders yield the identical point.
func TestSegSegCrossExact(t *testing.T) {
	r := rand.New(rand.NewSource(0x1c06))
	crossed := 0
	for i := 0; i < 40000; i++ {
		var a, b, c, d Point
		tilted := i%2 == 1
		if tilted {
			a, b, c, d = tiltedPt(r), tiltedPt(r), tiltedPt(r), tiltedPt(r)
		} else {
			a, b, c, d = flatPt(r), flatPt(r), flatPt(r), flatPt(r)
		}
		const axis = 2 // both families project non-degenerately by dropping z
		if !segmentsProperlyCross(a, b, c, d, axis) {
			continue
		}
		x := SegSegCross(a, b, c, d, axis)
		if !collinear(a, b, x) || !between(a, b, x) {
			t.Fatalf("case %d: crossing not on segment ab", i)
		}
		if !collinear(c, d, x) || !between(c, d, x) {
			t.Fatalf("case %d: crossing not on segment cd", i)
		}
		if tilted && sumXYZ(x).Sign() != 0 {
			t.Fatalf("case %d: crossing left the plane x+y+z=0", i)
		}
		if x2 := SegSegCross(c, d, a, b, axis); !x.Equal(x2) {
			t.Fatalf("case %d: SegSegCross not order-independent", i)
		}
		crossed++
	}
	if crossed == 0 {
		t.Fatal("no properly crossing pair was generated; test exercised nothing")
	}
	t.Logf("exact segment crossings verified: %d", crossed)
}

// --- helpers ---

func projF(p [3]float64, axis int) (u, v float64) {
	switch axis {
	case 0:
		return p[1], p[2]
	case 1:
		return p[0], p[2]
	default:
		return p[0], p[1]
	}
}

// flatPt is a point in the z=0 plane with small integer coordinates.
func flatPt(r *rand.Rand) Point {
	return pt([3]float64{float64(r.Intn(21) - 10), float64(r.Intn(21) - 10), 0})
}

// tiltedPt is a point on the plane x+y+z=0 with small integer coordinates.
func tiltedPt(r *rand.Rand) Point {
	x, y := float64(r.Intn(21)-10), float64(r.Intn(21)-10)
	return pt([3]float64{x, y, -(x + y)})
}

func sumXYZ(p Point) *big.Rat {
	s := new(big.Rat).Add(p.X, p.Y)
	return s.Add(s, p.Z)
}
