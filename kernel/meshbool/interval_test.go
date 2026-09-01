// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"testing"
)

// TestOrient3DIntervalNearCoplanar is the safety net for the interval filter at its
// hardest point: a vertex a hair off a plane. A, B, C span the plane z=0 with
// non-dyadic (constructed) in-plane coordinates, so the quad takes the interval /
// exact path, not predicates.Orient3D. D is placed on the plane and lifted by a
// rational epsilon shrinking from 1 to 1e-40 and by 0 exactly. For every epsilon and
// sign, Orient3D must equal the exact sign: the interval either certifies the correct
// sign, or — when it straddles zero — defers to the exact predicate. A wrong
// certification (a false sign near the plane) would be caught here.
func TestOrient3DIntervalNearCoplanar(t *testing.T) {
	t.Parallel()
	a := Point{big.NewRat(1, 3), big.NewRat(0, 1), big.NewRat(0, 1)}
	b := Point{big.NewRat(4, 3), big.NewRat(0, 1), big.NewRat(0, 1)}
	c := Point{big.NewRat(1, 3), big.NewRat(1, 1), big.NewRat(0, 1)}
	den := big.NewInt(1)
	ten := big.NewInt(10)
	for k := 0; k <= 40; k++ {
		for _, sign := range []int64{1, -1, 0} {
			eps := new(big.Rat)
			if sign != 0 {
				eps.SetFrac(big.NewInt(sign), new(big.Int).Set(den))
			}
			d := Point{big.NewRat(2, 3), big.NewRat(1, 4), eps}
			if got, want := Orient3D(a, b, c, d), orient3DVal(a, b, c, d).Sign(); got != want {
				t.Fatalf("k=%d sign=%d: Orient3D=%d, exact=%d (interval mis-certified)", k, sign, got, want)
			}
		}
		den.Mul(den, ten)
	}
}

// TestOrient3DIntervalCertifies checks that a clearly non-degenerate constructed quad
// resolves through the interval filter (its result matches exact) and that an exactly
// coplanar constructed quad returns 0 — the interval defers and the exact path yields
// the true 0, never a filtered near-zero guess.
func TestOrient3DIntervalCertifies(t *testing.T) {
	t.Parallel()
	a := Point{big.NewRat(1, 3), big.NewRat(0, 1), big.NewRat(0, 1)}
	b := Point{big.NewRat(7, 3), big.NewRat(0, 1), big.NewRat(0, 1)}
	c := Point{big.NewRat(1, 3), big.NewRat(2, 1), big.NewRat(0, 1)}
	if s, ok := orient3DInterval(a, b, c, Point{big.NewRat(1, 3), big.NewRat(1, 3), big.NewRat(1, 1)}); !ok || s == 0 {
		t.Fatalf("off-plane quad: interval returned (%d, %v), want a certified nonzero", s, ok)
	}
	if _, ok := orient3DInterval(a, b, c, Point{big.NewRat(1, 3), big.NewRat(1, 3), big.NewRat(0, 1)}); ok {
		t.Fatal("exactly coplanar quad was certified; the interval must straddle zero")
	}
	if got := Orient3D(a, b, c, Point{big.NewRat(1, 3), big.NewRat(1, 3), big.NewRat(0, 1)}); got != 0 {
		t.Fatalf("exactly coplanar quad: Orient3D=%d, want 0", got)
	}
}

// TestISquareContains checks the three iSquare regimes — a positive interval, a
// negative interval, and one straddling zero (whose lower bound must be 0, not a
// spurious negative) — each enclosing the true square range.
func TestISquareContains(t *testing.T) {
	t.Parallel()
	if s := iSquare(interval{2, 3}); s.lo > 4 || s.hi < 9 {
		t.Fatalf("positive interval: [%v,%v] does not enclose [4,9]", s.lo, s.hi)
	}
	if s := iSquare(interval{-3, -2}); s.lo > 4 || s.hi < 9 {
		t.Fatalf("negative interval: [%v,%v] does not enclose [4,9]", s.lo, s.hi)
	}
	if s := iSquare(interval{-3, 2}); s.lo != 0 || s.hi < 9 {
		t.Fatalf("straddling interval: [%v,%v], want [0, >=9]", s.lo, s.hi)
	}
}

// TestCoordInterval checks the coordinate bracket: an exact binary64 collapses to a
// point, and a non-dyadic rational is strictly enclosed.
func TestCoordInterval(t *testing.T) {
	t.Parallel()
	if iv := coordInterval(big.NewRat(3, 4)); iv.lo != 0.75 || iv.hi != 0.75 {
		t.Fatalf("dyadic 3/4: interval [%v,%v], want the point 0.75", iv.lo, iv.hi)
	}
	third := big.NewRat(1, 3)
	iv := coordInterval(third)
	lo := new(big.Rat).SetFloat64(iv.lo)
	hi := new(big.Rat).SetFloat64(iv.hi)
	if lo.Cmp(third) > 0 || hi.Cmp(third) < 0 || iv.lo >= iv.hi {
		t.Fatalf("1/3 not strictly enclosed by [%v,%v]", iv.lo, iv.hi)
	}
}

// TestIntervalOpsContain checks that the interval operations enclose the true result
// for representative operands, including sign-mixed multiplication (all four corner
// products relevant).
func TestIntervalOpsContain(t *testing.T) {
	t.Parallel()
	encloses := func(iv interval, exact *big.Rat) bool {
		lo := new(big.Rat).SetFloat64(iv.lo)
		hi := new(big.Rat).SetFloat64(iv.hi)
		return lo.Cmp(exact) <= 0 && hi.Cmp(exact) >= 0
	}
	x := coordInterval(big.NewRat(1, 3))                // ~0.3333
	y := coordInterval(big.NewRat(-5, 7))               // ~-0.7142
	if !encloses(iSub(x, y), big.NewRat(1*7+5*3, 21)) { // 1/3 - (-5/7) = 22/21
		t.Fatal("iSub does not enclose 22/21")
	}
	if !encloses(iAdd(x, y), big.NewRat(7-15, 21)) { // 1/3 + (-5/7) = -8/21
		t.Fatal("iAdd does not enclose -8/21")
	}
	if !encloses(iMul(x, y), big.NewRat(-5, 21)) { // (1/3)(-5/7) = -5/21
		t.Fatal("iMul does not enclose -5/21")
	}
}
