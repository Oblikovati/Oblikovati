// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// centroid returns the exact centroid of triangle t — a point strictly interior to
// it, used as a face's point-in-solid classification sample (see raycast.go).
func centroid(t [3]Point) Point {
	return Point{
		X: avg3(t[0].X, t[1].X, t[2].X),
		Y: avg3(t[0].Y, t[1].Y, t[2].Y),
		Z: avg3(t[0].Z, t[1].Z, t[2].Z),
	}
}

func avg3(a, b, c *big.Rat) *big.Rat {
	s := new(big.Rat).Add(a, b)
	s.Add(s, c)
	return s.Quo(s, big.NewRat(3, 1))
}
