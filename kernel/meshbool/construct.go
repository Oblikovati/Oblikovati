// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// EdgePlaneCross returns the exact point where the segment e0->e1 crosses the
// plane through a,b,c. The result lies EXACTLY on both the segment and the plane,
// so a shared edge crossed by a plane yields one identical point regardless of
// which incident face asks — the conforming property the co-refinement needs.
//
// PRECONDITION: e0 and e1 lie strictly on opposite sides of the plane
// (Orient3D(a,b,c,e0) and Orient3D(a,b,c,e1) have opposite nonzero signs). The
// caller establishes this; a non-straddling edge has no single crossing point.
func EdgePlaneCross(e0, e1, a, b, c Point) Point {
	d0 := orient3DVal(a, b, c, e0)
	d1 := orient3DVal(a, b, c, e1)
	// The crossing parameter t solves d0 + t*(d1-d0) = 0, i.e. t = d0/(d0-d1);
	// the orientation determinant is affine along the edge, so this is exact.
	t := new(big.Rat).Quo(d0, new(big.Rat).Sub(d0, d1))
	return Point{
		X: lerp(e0.X, e1.X, t),
		Y: lerp(e0.Y, e1.Y, t),
		Z: lerp(e0.Z, e1.Z, t),
	}
}

// lerp returns the exact a + t*(b-a).
func lerp(a, b, t *big.Rat) *big.Rat {
	diff := new(big.Rat).Sub(b, a)
	return new(big.Rat).Add(a, new(big.Rat).Mul(t, diff))
}
