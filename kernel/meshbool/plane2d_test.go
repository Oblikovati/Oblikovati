// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math"
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

// TestInCircleSignMatchesPredicates cross-checks the rational in-circle test
// against predicates.InCircle on the same projected coordinates, across all three
// drop axes, including the near-cocircular regime.
func TestInCircleSignMatchesPredicates(t *testing.T) {
	r := rand.New(rand.NewSource(0x1c08))
	for i := 0; i < 20000; i++ {
		a, b, c := rc(r), rc(r), rc(r)
		var d [3]float64
		if i%2 == 0 {
			d = rc(r)
		} else {
			// place d on the circumcircle of a,b,c in the xy plane (near-cocircular)
			ox, oy, ok := circumcentreXY(a, b, c)
			if !ok {
				continue
			}
			ang := r.Float64() * 6.283185307179586
			rad := math.Hypot(a[0]-ox, a[1]-oy)
			d = [3]float64{ox + rad*math.Cos(ang), oy + rad*math.Sin(ang), 0}
		}
		for axis := 0; axis < 3; axis++ {
			au, av := projF(a, axis)
			bu, bv := projF(b, axis)
			cu, cv := projF(c, axis)
			du, dv := projF(d, axis)
			want := predicates.InCircle(au, av, bu, bv, cu, cv, du, dv)
			if got := inCircleSign(pt(a), pt(b), pt(c), pt(d), axis); got != want {
				t.Fatalf("case %d axis %d: inCircleSign=%d, predicates=%d", i, axis, got, want)
			}
		}
	}
}

func circumcentreXY(a, b, c [3]float64) (float64, float64, bool) {
	d1x, d1y := b[0]-a[0], b[1]-a[1]
	d2x, d2y := c[0]-a[0], c[1]-a[1]
	det := d1x*d2y - d1y*d2x
	if det == 0 {
		return 0, 0, false
	}
	r1 := (b[0]*b[0] + b[1]*b[1] - a[0]*a[0] - a[1]*a[1]) / 2
	r2 := (c[0]*c[0] + c[1]*c[1] - a[0]*a[0] - a[1]*a[1]) / 2
	return (r1*d2y - r2*d1y) / det, (d1x*r2 - d2x*r1) / det, true
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

// TestInCircleFilterMatchesExact is the safety net for the in-circle interval filter:
// over a large random sweep mixing exact and constructed vertices, across all three
// projection axes, inCircleSign (interval then exact) must equal the pure exact
// determinant. A wrong certification would show as a mismatch.
func TestInCircleFilterMatchesExact(t *testing.T) {
	rng := rand.New(rand.NewSource(0x1c1c))
	for i := 0; i < 20000; i++ {
		a := randPoint(rng, i%3 == 0)
		b := randPoint(rng, i%5 == 0)
		c := randPoint(rng, i%7 == 0)
		d := randPoint(rng, i%2 == 0)
		axis := i % 3
		if got, want := inCircleSign(a, b, c, d, axis), inCircleExact(a, b, c, d, axis); got != want {
			t.Fatalf("quad %d axis %d: inCircleSign=%d, exact=%d (interval mis-certified)", i, axis, got, want)
		}
	}
}

// TestInCircleNearCocircular stresses the filter at its hardest point: three points on
// the radius-5 circle (a,b,c) and a fourth on the circle at (3,4) lifted off it by a
// rational epsilon from 1 down to 1e-40 in both signs, plus 0 exactly. For every
// epsilon inCircleSign must equal the exact sign — the interval certifies the correct
// side or defers to exact; the cocircular case (epsilon 0) must yield 0.
func TestInCircleNearCocircular(t *testing.T) {
	a := Point{big.NewRat(5, 1), big.NewRat(0, 1), big.NewRat(0, 1)}
	b := Point{big.NewRat(0, 1), big.NewRat(5, 1), big.NewRat(0, 1)}
	c := Point{big.NewRat(-5, 1), big.NewRat(0, 1), big.NewRat(0, 1)}
	den := big.NewInt(1)
	ten := big.NewInt(10)
	for k := 0; k <= 40; k++ {
		for _, sign := range []int64{1, -1, 0} {
			dv := big.NewRat(4, 1)
			if sign != 0 {
				dv = new(big.Rat).Add(big.NewRat(4, 1), new(big.Rat).SetFrac(big.NewInt(sign), new(big.Int).Set(den)))
			}
			d := Point{big.NewRat(3, 1), dv, big.NewRat(0, 1)}
			if got, want := inCircleSign(a, b, c, d, 2), inCircleExact(a, b, c, d, 2); got != want {
				t.Fatalf("k=%d sign=%d: inCircleSign=%d, exact=%d", k, sign, got, want)
			}
		}
		den.Mul(den, ten)
	}
}
