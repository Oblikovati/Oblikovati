// SPDX-License-Identifier: GPL-2.0-only

package predicates

import (
	"math"
	"math/big"
)

// InCircle is the exact-sign in-circle test the Delaunay flip needs: is point d
// inside the circle through a, b, c? It returns +1 if d is strictly inside the
// circumcircle when a,b,c are in counterclockwise order (−1 if a,b,c are clockwise),
// −1 for d strictly outside, and 0 when the four points are cocircular. The sign is
// exact, so an "is this edge Delaunay" decision is one global truth, never a
// tolerance flip that could loop the flip algorithm.
//
// Example:
//
//	predicates.InCircle(0,0, 1,0, 1,1, 0.5,0.5) // +1 (centre is inside)
func InCircle(ax, ay, bx, by, cx, cy, dx, dy float64) int {
	if det, certified := filterInCircle(ax, ay, bx, by, cx, cy, dx, dy); certified {
		return signOf(det)
	}
	return exactInCircle(ax, ay, bx, by, cx, cy, dx, dy)
}

// iccFilterA bounds the forward error of the floating-point in-circle estimate
// relative to its permanent; Shewchuk's a-priori bound for binary64 (the in-circle
// determinant is degree four, hence a larger constant than orient's).
const iccFilterA = (10.0 + 96.0*epsilon) * epsilon

// filterInCircle returns the estimate and whether its sign is certified.
func filterInCircle(ax, ay, bx, by, cx, cy, dx, dy float64) (det float64, certified bool) {
	adx, ady := ax-dx, ay-dy
	bdx, bdy := bx-dx, by-dy
	cdx, cdy := cx-dx, cy-dy

	bdxcdy, cdxbdy := rounded(bdx*cdy), rounded(cdx*bdy)
	cdxady, adxcdy := rounded(cdx*ady), rounded(adx*cdy)
	adxbdy, bdxady := rounded(adx*bdy), rounded(bdx*ady)
	alift := rounded(adx*adx) + rounded(ady*ady)
	blift := rounded(bdx*bdx) + rounded(bdy*bdy)
	clift := rounded(cdx*cdx) + rounded(cdy*cdy)

	det = rounded(alift*(bdxcdy-cdxbdy)) + rounded(blift*(cdxady-adxcdy)) + rounded(clift*(adxbdy-bdxady))
	permanent := (math.Abs(bdxcdy)+math.Abs(cdxbdy))*alift +
		(math.Abs(cdxady)+math.Abs(adxcdy))*blift +
		(math.Abs(adxbdy)+math.Abs(bdxady))*clift
	return det, math.Abs(det) >= iccFilterA*permanent
}

// exactInCircle recomputes the in-circle determinant over exact rationals.
func exactInCircle(ax, ay, bx, by, cx, cy, dx, dy float64) int {
	adx, ady := ratDiff(ax, dx), ratDiff(ay, dy)
	bdx, bdy := ratDiff(bx, dx), ratDiff(by, dy)
	cdx, cdy := ratDiff(cx, dx), ratDiff(cy, dy)

	alift := ratSumSquares(adx, ady)
	blift := ratSumSquares(bdx, bdy)
	clift := ratSumSquares(cdx, cdy)

	t1 := new(big.Rat).Mul(alift, crossDiff(bdx, cdy, cdx, bdy))
	t2 := new(big.Rat).Mul(blift, crossDiff(cdx, ady, adx, cdy))
	t3 := new(big.Rat).Mul(clift, crossDiff(adx, bdy, bdx, ady))
	return t1.Add(t1, t2).Add(t1, t3).Sign()
}

// ratSumSquares returns x*x + y*y exactly.
func ratSumSquares(x, y *big.Rat) *big.Rat {
	xx := new(big.Rat).Mul(x, x)
	yy := new(big.Rat).Mul(y, y)
	return xx.Add(xx, yy)
}
