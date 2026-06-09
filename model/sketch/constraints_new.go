// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// GroundConstraint freezes every point of an entity at its current position — the
// user-visible "fully fix this geometry" (Inventor's Ground), distinct from the internal
// single-point Fix.
type GroundConstraint struct {
	constraintBase
	pts    []*Point
	x0, y0 []float64
}

// AddGround grounds all of an entity's points at their current positions.
func (g *GeometricConstraints) AddGround(e Entity) *GroundConstraint {
	return g.AddGroundPoints(entityPoints(e)...)
}

// AddGroundPoints grounds the given points (used by restore and by AddGround).
func (g *GeometricConstraints) AddGroundPoints(pts ...*Point) *GroundConstraint {
	c := &GroundConstraint{constraintBase: newConstraint(), pts: pts}
	for _, p := range pts {
		c.x0 = append(c.x0, float64(p.X))
		c.y0 = append(c.y0, float64(p.Y))
	}
	g.add(c)
	return c
}

// Points returns the grounded points (for serialization).
func (c *GroundConstraint) Points() []*Point { return c.pts }

// Residuals reports each point's drift from its frozen position.
func (c *GroundConstraint) Residuals() []float64 {
	out := make([]float64, 0, len(c.pts)*2)
	for i, p := range c.pts {
		out = append(out, float64(p.X)-c.x0[i], float64(p.Y)-c.y0[i])
	}
	return out
}

// Variables returns the grounded points' coordinates.
func (c *GroundConstraint) Variables() []*math.Scalar {
	out := make([]*math.Scalar, 0, len(c.pts)*2)
	for _, p := range c.pts {
		out = append(out, &p.X, &p.Y)
	}
	return out
}

// OffsetConstraint holds line L2 parallel to L1 at a fixed signed perpendicular distance
// (the geometric relation behind an offset; pairs with a driven offset dimension).
type OffsetConstraint struct {
	constraintBase
	L1, L2 *Line
	Dist   float64
}

// AddOffset constrains L2 parallel to L1 at the signed perpendicular distance dist.
func (g *GeometricConstraints) AddOffset(l1, l2 *Line, dist float64) *OffsetConstraint {
	c := &OffsetConstraint{constraintBase: newConstraint(), L1: l1, L2: l2, Dist: dist}
	g.add(c)
	return c
}

// Residuals reports the parallel error and the perpendicular-distance error.
func (c *OffsetConstraint) Residuals() []float64 {
	d1x, d1y := lineDir(c.L1)
	d2x, d2y := lineDir(c.L2)
	parallel := float64(d1x*d2y - d1y*d2x)
	length := stdmath.Hypot(float64(d1x), float64(d1y))
	if length == 0 {
		return []float64{parallel, 0}
	}
	nx, ny := -float64(d1y)/length, float64(d1x)/length
	wx := float64(c.L2.A.X - c.L1.A.X)
	wy := float64(c.L2.A.Y - c.L1.A.Y)
	return []float64{parallel, wx*nx + wy*ny - c.Dist}
}

// Variables returns the two lines' endpoint coordinates.
func (c *OffsetConstraint) Variables() []*math.Scalar { return lineVars(c.L1, c.L2) }

// PatternConstraint binds a pattern member point to a seed point by a fixed offset vector,
// so moving the seed carries the member (the rigid link behind a sketch pattern member).
type PatternConstraint struct {
	constraintBase
	Seed, Member *Point
	dx, dy       float64
	live         func() (float64, float64) // parametric offset (spacing-driven); nil ⇒ frozen dx,dy
}

// AddPatternLink links member to seed at their current (frozen) offset.
func (g *GeometricConstraints) AddPatternLink(seed, member *Point) *PatternConstraint {
	c := &PatternConstraint{
		constraintBase: newConstraint(), Seed: seed, Member: member,
		dx: float64(member.X - seed.X), dy: float64(member.Y - seed.Y),
	}
	g.add(c)
	return c
}

// AddPatternLinkLive links member to seed at a live offset, so a parameter driving the
// offset (a pattern's spacing) repositions the member on each solve.
func (g *GeometricConstraints) AddPatternLinkLive(seed, member *Point, offset func() (float64, float64)) *PatternConstraint {
	c := &PatternConstraint{constraintBase: newConstraint(), Seed: seed, Member: member, live: offset}
	g.add(c)
	return c
}

// Residuals reports the member's drift from seed + the (frozen or live) offset.
func (c *PatternConstraint) Residuals() []float64 {
	dx, dy := c.dx, c.dy
	if c.live != nil {
		dx, dy = c.live()
	}
	return []float64{
		float64(c.Member.X-c.Seed.X) - dx,
		float64(c.Member.Y-c.Seed.Y) - dy,
	}
}

// Variables returns the seed and member coordinates.
func (c *PatternConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{&c.Seed.X, &c.Seed.Y, &c.Member.X, &c.Member.Y}
}
