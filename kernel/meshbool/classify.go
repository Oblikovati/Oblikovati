// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math"
	"math/big"
)

// Point-in-solid classification by the generalized winding number (Jacobson,
// Kavan & Sorkine, "Robust Inside-Outside Segmentation using Generalized Winding
// Numbers", SIGGRAPH 2013): the sum of signed solid angles the mesh subtends at a
// point, divided by 4π — ~1 inside a closed outward-oriented mesh, ~0 outside.
//
// This stage is floating-point (the solid angle is transcendental). That is not
// the compromise #2084 was about: in #2084 the per-operand winding classification
// was already correct with clear margins — the tear was missing CONFORMANCE, which
// the exact co-refinement now guarantees. Classification only has to pick the side
// of a face that is, by construction, entirely on one side of the other solid, and
// the sample is the face centroid (strictly interior), so the winding number is
// unambiguous. (Faces coplanar with the other surface are the exception and need
// coplanar-keep handling, a later layer, not a winding test.)

// insideMesh reports whether p lies inside the closed, outward-oriented triangle
// mesh, by rounding its generalized winding number away from zero.
func insideMesh(p Point, mesh [][3]Point) bool {
	return math.Abs(windingNumber(p, mesh)) >= 0.5
}

// windingNumber returns the generalized winding number of p with respect to mesh.
func windingNumber(p Point, mesh [][3]Point) float64 {
	pf := p.floats()
	total := 0.0
	for _, t := range mesh {
		total += solidAngle(pf, t)
	}
	return total / (4 * math.Pi)
}

// solidAngle returns the signed solid angle triangle t subtends at p, via the
// Van Oosterom–Strackee formula (numerically stable through atan2).
func solidAngle(p [3]float64, t [3]Point) float64 {
	a := fsub(t[0].floats(), p)
	b := fsub(t[1].floats(), p)
	c := fsub(t[2].floats(), p)
	la, lb, lc := fnorm(a), fnorm(b), fnorm(c)
	num := fdot(a, fcross(b, c))
	den := la*lb*lc + fdot(a, b)*lc + fdot(b, c)*la + fdot(c, a)*lb
	return 2 * math.Atan2(num, den)
}

// centroid returns the exact centroid of triangle t — a point strictly interior to
// it, used as the face's classification sample.
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

// floats returns p's coordinates as float64 for the winding-number computation.
func (p Point) floats() [3]float64 {
	x, _ := p.X.Float64()
	y, _ := p.Y.Float64()
	z, _ := p.Z.Float64()
	return [3]float64{x, y, z}
}

func fsub(a, b [3]float64) [3]float64 { return [3]float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func fdot(a, b [3]float64) float64    { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func fnorm(a [3]float64) float64      { return math.Sqrt(fdot(a, a)) }

func fcross(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
