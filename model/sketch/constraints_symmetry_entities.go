// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// Entity symmetry (#1870): Inventor's AddSymmetry takes two SketchEntity operands, not only
// two points — two lines are made mirror images, two circular curves have their centres
// mirrored and radii equalised, about a mirror line. Both reuse the grounded point-symmetry
// equation (adPointSymmetry) rather than a fresh residual: a line's symmetry is its two
// endpoint pairs mirrored, a circle's is its centre mirrored plus equal radius.

// LineSymmetryConstraint makes two lines mirror images across a line: each of L1's endpoints
// is symmetric to the paired endpoint of L2. The endpoint pairing (A↔A/B↔B vs A↔B/B↔A) is
// fixed at creation from whichever the current geometry is closer to, so the solver moves the
// lines the least — the same "preserve the picked configuration" policy as CircularTangent.
type LineSymmetryConstraint struct {
	constraintBase
	L1, L2 *Line
	About  *Line
	cross  bool // true ⇒ L1.A mirrors L2.B and L1.B mirrors L2.A
}

// AddLineSymmetry constrains lines l1 and l2 to be symmetric about line about.
func (g *GeometricConstraints) AddLineSymmetry(l1, l2, about *Line) *LineSymmetryConstraint {
	c := &LineSymmetryConstraint{constraintBase: newConstraint(), L1: l1, L2: l2, About: about}
	c.cross = c.preferCross()
	g.add(c)
	return c
}

// preferCross reports whether the crossed endpoint pairing (A↔B, B↔A) is currently closer to
// symmetric than the straight pairing (A↔A, B↔B).
func (c *LineSymmetryConstraint) preferCross() bool {
	la, lb := c.About.A.Position(), c.About.B.Position()
	a1, b1 := c.L1.A.Position(), c.L1.B.Position()
	a2, b2 := c.L2.A.Position(), c.L2.B.Position()
	straight := pointSymmetryError(a1, a2, la, lb) + pointSymmetryError(b1, b2, la, lb)
	crossed := pointSymmetryError(a1, b2, la, lb) + pointSymmetryError(b1, a2, la, lb)
	return crossed < straight
}

// residualAD: v = [L1.A, L1.B, L2.A, L2.B, About.A, About.B] (X,Y each). Each L1 endpoint is
// symmetric to its paired L2 endpoint about the mirror line — four residuals, exactly the four
// DOF that mirroring removes from the second line.
func (c *LineSymmetryConstraint) residualAD(v []ad.Number) []ad.Number {
	a1, b1 := ad.V2(v[0], v[1]), ad.V2(v[2], v[3])
	a2, b2 := ad.V2(v[4], v[5]), ad.V2(v[6], v[7])
	la, lb := ad.V2(v[8], v[9]), ad.V2(v[10], v[11])
	pa, pb := a2, b2
	if c.cross {
		pa, pb = b2, a2
	}
	return append(adPointSymmetry(a1, pa, la, lb), adPointSymmetry(b1, pb, la, lb)...)
}
func (c *LineSymmetryConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *LineSymmetryConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}

func (c *LineSymmetryConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{
		&c.L1.A.X, &c.L1.A.Y, &c.L1.B.X, &c.L1.B.Y,
		&c.L2.A.X, &c.L2.A.Y, &c.L2.B.X, &c.L2.B.Y,
		&c.About.A.X, &c.About.A.Y, &c.About.B.X, &c.About.B.Y,
	}
}

// CircularSymmetryConstraint makes two circular curves (circles or arcs) symmetric across a
// line: their centres mirror and their radii are equal (#1870). A centre has no pairing
// ambiguity, so unlike LineSymmetry it needs no cross flag.
type CircularSymmetryConstraint struct {
	constraintBase
	C1, C2 CircularCurve
	About  *Line
}

// AddCircularSymmetry constrains circular curves c1 and c2 to be symmetric about line about.
func (g *GeometricConstraints) AddCircularSymmetry(c1, c2 CircularCurve, about *Line) *CircularSymmetryConstraint {
	c := &CircularSymmetryConstraint{constraintBase: newConstraint(), C1: c1, C2: c2, About: about}
	g.add(c)
	return c
}

// residualAD: v = [<C1 circularVars>, <C2 circularVars>, About.A, About.B]. The two centres are
// symmetric about the mirror line (two residuals) and the radii are equal (one) — three
// residuals, the DOF a mirror removes from the second circle (centre + radius).
func (c *CircularSymmetryConstraint) residualAD(v []ad.Number) []ad.Number {
	c1, r1, n1 := c.C1.circularFrameAD(v, 0)
	c2, r2, n2 := c.C2.circularFrameAD(v, n1)
	off := n1 + n2
	la, lb := ad.V2(v[off], v[off+1]), ad.V2(v[off+2], v[off+3])
	return append(adPointSymmetry(c1, c2, la, lb), r1.Sub(r2))
}
func (c *CircularSymmetryConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *CircularSymmetryConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}

func (c *CircularSymmetryConstraint) Variables() []*math.Scalar {
	vars := append(c.C1.circularVars(), c.C2.circularVars()...)
	return append(vars, &c.About.A.X, &c.About.A.Y, &c.About.B.X, &c.About.B.Y)
}

// pointSymmetryError is the float twin of adPointSymmetry: the summed magnitude of the
// midpoint-off-line and segment-not-perpendicular errors, used to pick a line-symmetry
// endpoint pairing from the current geometry (no gradient needed).
func pointSymmetryError(a, b, la, lb math.Point2) float64 {
	dx, dy := float64(lb.X-la.X), float64(lb.Y-la.Y)
	length := stdmath.Hypot(dx, dy)
	if length < float64(math.DefaultTolerance) {
		return stdmath.Hypot(float64(b.X-a.X), float64(b.Y-a.Y))
	}
	mx, my := float64(a.X+b.X)/2-float64(la.X), float64(a.Y+b.Y)/2-float64(la.Y)
	perp := (dx*my - dy*mx) / length
	proj := (float64(b.X-a.X)*dx + float64(b.Y-a.Y)*dy) / length
	return stdmath.Abs(perp) + stdmath.Abs(proj)
}
