// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"math/rand"
	"testing"
)

// TestOrient3DFastPathMatchesExact is the safety net for the float fast path: over a
// large random sweep mixing exact-binary64 vertices (which take the delegated
// filtered-exact predicate) and constructed non-dyadic vertices (which take the
// pure-rational path), Orient3D must always equal the sign of the pure exact
// determinant. Both paths are exact, so any disagreement would be a bug in the
// fast-path gate or a sign-convention seam between the two predicates.
func TestOrient3DFastPathMatchesExact(t *testing.T) {
	rng := rand.New(rand.NewSource(20260824))
	for i := 0; i < 20000; i++ {
		a := randPoint(rng, i%3 == 0)
		b := randPoint(rng, i%5 == 0)
		c := randPoint(rng, i%7 == 0)
		d := randPoint(rng, i%2 == 0)
		if got, want := Orient3D(a, b, c, d), orient3DVal(a, b, c, d).Sign(); got != want {
			t.Fatalf("quad %d: fast Orient3D=%d, exact sign=%d", i, got, want)
		}
	}
}

// TestOrient3DExactCoplanar checks that exact-binary64 coplanar and off-plane cases
// resolve to 0 / nonzero through the fast path — the predicate's exact fallback must
// return 0 for the truly coplanar quad, not a filtered near-zero guess.
func TestOrient3DExactCoplanar(t *testing.T) {
	a := FromCoords(0, 0, 0)
	b := FromCoords(2, 0, 0)
	c := FromCoords(0, 2, 0)
	if got := Orient3D(a, b, c, FromCoords(1, 1, 0)); got != 0 {
		t.Fatalf("coplanar quad: Orient3D=%d, want 0", got)
	}
	if got := Orient3D(a, b, c, FromCoords(1, 1, 1)); got == 0 {
		t.Fatal("off-plane quad classified coplanar")
	}
}

// TestFloat64Exact checks the fast-path gate: a dyadic coordinate is exact, a
// constructed 1/3 coordinate is not (so it must fall to the rational path).
func TestFloat64Exact(t *testing.T) {
	if _, ok := FromCoords(1.5, -2.25, 3).float64Exact(); !ok {
		t.Fatal("dyadic coordinates should be float-exact")
	}
	third := Point{new(big.Rat).SetFrac64(1, 3), big.NewRat(0, 1), big.NewRat(0, 1)}
	if _, ok := third.float64Exact(); ok {
		t.Fatal("1/3 must not be float-exact")
	}
	// Power-of-two denominator (passes the cheap reject) but a numerator too wide for
	// the 53-bit mantissa, so the round-trip check must still fail it.
	wide := new(big.Rat).SetFrac(new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 60), big.NewInt(1)), big.NewInt(2))
	if _, ok := (Point{wide, big.NewRat(0, 1), big.NewRat(0, 1)}).float64Exact(); ok {
		t.Fatal("(2^60+1)/2 is not exactly representable and must be rejected")
	}
}

// TestOrient3DWidePowerOfTwoDenom drives a quad through floatQuad's second pass: a
// vertex whose denominator is a power of two (passing the cheap screen) but whose
// numerator is too wide for the mantissa, so the round-trip disqualifies it and the
// quad resolves through the interval/exact path. The result must match exact.
func TestOrient3DWidePowerOfTwoDenom(t *testing.T) {
	wide := new(big.Rat).SetFrac(new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 60), big.NewInt(1)), big.NewInt(2))
	a, b, c := FromCoords(0, 0, 0), FromCoords(1, 0, 0), FromCoords(0, 1, 0)
	d := Point{wide, big.NewRat(1, 1), big.NewRat(1, 1)}
	if got, want := Orient3D(a, b, c, d), orient3DVal(a, b, c, d).Sign(); got != want {
		t.Fatalf("wide-denom quad: Orient3D=%d, exact=%d", got, want)
	}
}

func randCoord(rng *rand.Rand) float64 { return (rng.Float64()*2 - 1) * 10 }

// randPoint returns an exact-binary64 point, or one perturbed by 1/3 on X so it is
// no longer dyadic and forces the pure-rational Orient3D path.
func randPoint(rng *rand.Rand, constructed bool) Point {
	p := FromCoords(randCoord(rng), randCoord(rng), randCoord(rng))
	if constructed {
		p.X = new(big.Rat).Add(p.X, big.NewRat(1, 3))
	}
	return p
}
