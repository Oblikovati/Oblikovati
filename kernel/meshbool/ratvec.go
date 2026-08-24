// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// Exact rational 3-vector helpers, used to build the plane-intersection line and
// parametrize points along it during triangle-triangle intersection.

// rcross returns the exact cross product a×b.
func rcross(a, b [3]*big.Rat) [3]*big.Rat {
	return [3]*big.Rat{
		crossDiff(a[1], b[2], a[2], b[1]),
		crossDiff(a[2], b[0], a[0], b[2]),
		crossDiff(a[0], b[1], a[1], b[0]),
	}
}

// rdot returns the exact dot product a·b.
func rdot(a, b [3]*big.Rat) *big.Rat {
	x := new(big.Rat).Mul(a[0], b[0])
	y := new(big.Rat).Mul(a[1], b[1])
	z := new(big.Rat).Mul(a[2], b[2])
	return x.Add(x, y).Add(x, z)
}

// triNormal returns the exact (unnormalized) normal of triangle tri.
func triNormal(tri [3]Point) [3]*big.Rat {
	return rcross(tri[1].sub(tri[0]), tri[2].sub(tri[0]))
}

// rcollinear reports whether x lies exactly on the line through a and b.
func rcollinear(a, b, x Point) bool {
	c := rcross(x.sub(a), b.sub(a))
	return c[0].Sign() == 0 && c[1].Sign() == 0 && c[2].Sign() == 0
}

// segParam returns (x-a)·(b-a), the exact ordering parameter of x along a→b; it
// runs from 0 at a to (b-a)·(b-a) at b, so a point is on segment [a,b] when it is
// collinear and its parameter lies in that closed range.
func segParam(a, b, x Point) *big.Rat {
	return rdot(x.sub(a), b.sub(a))
}
