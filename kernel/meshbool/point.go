// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"fmt"
	"math/big"

	"oblikovati.org/math"
)

// Point is an exact 3D position with rational coordinates. See doc.go: original
// vertices convert exactly and constructed intersection vertices are built by
// exact rational arithmetic, so every arrangement decision on Points is exact.
type Point struct {
	X, Y, Z *big.Rat
}

// FromCoords builds an exact Point from finite float64 coordinates (no rounding —
// a binary64 is a dyadic rational). It panics on a non-finite coordinate, which is
// a caller bug: geometry positions are finite.
func FromCoords(x, y, z float64) Point {
	return Point{ratOf(x), ratOf(y), ratOf(z)}
}

// FromPoint3 builds an exact Point from a kernel [math.Point3].
func FromPoint3(p math.Point3) Point {
	return FromCoords(p.X, p.Y, p.Z)
}

// Round returns the nearest float64 position, the single place precision is lost.
// It is used only when the finished manifold is written back to the kernel.
func (p Point) Round() math.Point3 {
	x, _ := p.X.Float64()
	y, _ := p.Y.Float64()
	z, _ := p.Z.Float64()
	return math.P3(x, y, z)
}

// Equal reports exact coordinate equality. Because constructed points are exact,
// two vertices coincide iff Equal is true — there is no tolerance and no ambiguity.
func (p Point) Equal(q Point) bool {
	return p.X.Cmp(q.X) == 0 && p.Y.Cmp(q.Y) == 0 && p.Z.Cmp(q.Z) == 0
}

// float64Exact returns p's coordinates as float64 and reports whether every
// coordinate is exactly a binary64 (the float equals the rational, so an exact
// predicate on the floats gives the same sign as one on the rationals). Original
// tessellation vertices are exact; constructed intersection vertices — built by
// rational arithmetic with denominators that are not powers of two — generally are
// not. This is the gate for the float fast path in [Orient3D].
func (p Point) float64Exact() ([3]float64, bool) {
	var f [3]float64
	for i, c := range [3]*big.Rat{p.X, p.Y, p.Z} {
		v, _ := c.Float64()
		r := new(big.Rat).SetFloat64(v) // nil on a non-finite round-trip
		if r == nil || r.Cmp(c) != 0 {
			return f, false
		}
		f[i] = v
	}
	return f, true
}

// sub returns the exact vector p-q as three rationals.
func (p Point) sub(q Point) [3]*big.Rat {
	return [3]*big.Rat{
		new(big.Rat).Sub(p.X, q.X),
		new(big.Rat).Sub(p.Y, q.Y),
		new(big.Rat).Sub(p.Z, q.Z),
	}
}

// ratOf converts a finite binary64 to its exact rational value.
func ratOf(x float64) *big.Rat {
	r := new(big.Rat).SetFloat64(x)
	if r == nil {
		panic(fmt.Sprintf("meshbool: non-finite coordinate %v; positions must be finite", x))
	}
	return r
}
