// SPDX-License-Identifier: GPL-2.0-only

package predicates

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

func TestInCircleBasic(t *testing.T) {
	// Unit-ish circle through (0,0),(1,0),(1,1) (CCW). Centre (0.5,0.5) is inside;
	// (2,2) is outside; (0,1) is on it (cocircular).
	if got := InCircle(0, 0, 1, 0, 1, 1, 0.5, 0.5); got != 1 {
		t.Fatalf("centre inside: got %d, want +1", got)
	}
	if got := InCircle(0, 0, 1, 0, 1, 1, 2, 2); got != -1 {
		t.Fatalf("far point: got %d, want -1", got)
	}
	if got := InCircle(0, 0, 1, 0, 1, 1, 0, 1); got != 0 {
		t.Fatalf("cocircular: got %d, want 0", got)
	}
}

// TestInCircleVsOracleUnderStress places d on the circumcircle of a,b,c (the
// cocircular knife-edge) and checks InCircle matches an independent exact oracle
// (rational circumcentre + squared-distance compare, orientation-corrected), with
// the naive float determinant proven to disagree on a non-trivial share.
func TestInCircleVsOracleUnderStress(t *testing.T) {
	r := rand.New(rand.NewSource(0x1c1c))
	teeth := 0
	const n = 20000
	for i := range n {
		a := [2]float64{r.Float64() * 100, r.Float64() * 100}
		b := [2]float64{r.Float64() * 100, r.Float64() * 100}
		c := [2]float64{r.Float64() * 100, r.Float64() * 100}
		o, ok := circumcentre(a, b, c)
		if !ok {
			continue // collinear
		}
		ang := r.Float64() * 2 * math.Pi
		rad := math.Hypot(a[0]-o[0], a[1]-o[1])
		d := [2]float64{o[0] + rad*math.Cos(ang), o[1] + rad*math.Sin(ang)} // ~on the circle
		want := oracleInCircle(a, b, c, d)
		if got := InCircle(a[0], a[1], b[0], b[1], c[0], c[1], d[0], d[1]); got != want {
			t.Fatalf("case %d: InCircle=%d, oracle=%d", i, got, want)
		}
		if naiveInCircle(a, b, c, d) != want {
			teeth++
		}
	}
	if teeth == 0 {
		t.Fatalf("stress never reached a case naive float got wrong (%d cases)", n)
	}
	t.Logf("naive float in-circle disagreed with the exact sign on %d/%d near-cocircular cases", teeth, n)
}

// --- oracle and naive reference ---

// oracleInCircle returns the exact InCircle sign via the circumcircle: d is inside
// iff its squared distance to the exact circumcentre is less than the radius
// squared, corrected by the orientation of a,b,c (Shewchuk's sign convention).
func oracleInCircle(a, b, c, d [2]float64) int {
	o, ok := ratCircumcentre(a, b, c)
	if !ok {
		return 0
	}
	r2 := ratDist2(o, a)
	dd := ratDist2(o, d)
	insideSign := r2.Cmp(dd) // +1 if d inside (radius > distance)
	return exactOrient2D(a[0], a[1], b[0], b[1], c[0], c[1]) * insideSign
}

// ratCircumcentre solves the two perpendicular-bisector equations exactly; nil if
// a,b,c are collinear.
func ratCircumcentre(a, b, c [2]float64) ([2]*big.Rat, bool) {
	d1x, d1y := ratDiff(b[0], a[0]), ratDiff(b[1], a[1])
	d2x, d2y := ratDiff(c[0], a[0]), ratDiff(c[1], a[1])
	det := crossDiff(d1x, d2y, d1y, d2x)
	if det.Sign() == 0 {
		return [2]*big.Rat{}, false
	}
	r1 := halfDiffSq(b, a)
	r2 := halfDiffSq(c, a)
	ox := new(big.Rat).Quo(crossDiff(r1, d2y, r2, d1y), det)
	oy := new(big.Rat).Quo(crossDiff(d1x, r2, d2x, r1), det)
	return [2]*big.Rat{ox, oy}, true
}

// halfDiffSq returns (|p|^2 - |q|^2)/2 exactly, the right-hand side of the
// perpendicular-bisector equation.
func halfDiffSq(p, q [2]float64) *big.Rat {
	s := new(big.Rat).Sub(ratSumSquares(ratOf(p[0]), ratOf(p[1])), ratSumSquares(ratOf(q[0]), ratOf(q[1])))
	return s.Quo(s, big.NewRat(2, 1))
}

func ratDist2(o [2]*big.Rat, p [2]float64) *big.Rat {
	dx := new(big.Rat).Sub(o[0], ratOf(p[0]))
	dy := new(big.Rat).Sub(o[1], ratOf(p[1]))
	return ratSumSquares(dx, dy)
}

func circumcentre(a, b, c [2]float64) ([2]float64, bool) {
	d1x, d1y := b[0]-a[0], b[1]-a[1]
	d2x, d2y := c[0]-a[0], c[1]-a[1]
	det := d1x*d2y - d1y*d2x
	if det == 0 {
		return [2]float64{}, false
	}
	r1 := (b[0]*b[0] + b[1]*b[1] - a[0]*a[0] - a[1]*a[1]) / 2
	r2 := (c[0]*c[0] + c[1]*c[1] - a[0]*a[0] - a[1]*a[1]) / 2
	return [2]float64{(r1*d2y - r2*d1y) / det, (d1x*r2 - d2x*r1) / det}, true
}

func naiveInCircle(a, b, c, d [2]float64) int {
	adx, ady := a[0]-d[0], a[1]-d[1]
	bdx, bdy := b[0]-d[0], b[1]-d[1]
	cdx, cdy := c[0]-d[0], c[1]-d[1]
	alift := adx*adx + ady*ady
	blift := bdx*bdx + bdy*bdy
	clift := cdx*cdx + cdy*cdy
	det := alift*(bdx*cdy-cdx*bdy) + blift*(cdx*ady-adx*cdy) + clift*(adx*bdy-bdx*ady)
	return signOf(det)
}
