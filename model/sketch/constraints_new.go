// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
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

// residualAD: each grounded coordinate must hold its frozen value (v ordered X,Y per
// point, matching Variables()).
func (c *GroundConstraint) residualAD(v []ad.Number) []ad.Number {
	out := make([]ad.Number, 0, len(c.pts)*2)
	for i := range c.pts {
		out = append(out, v[2*i].AddConst(-c.x0[i]), v[2*i+1].AddConst(-c.y0[i]))
	}
	return out
}

// Residuals reports each point's drift from its frozen position.
func (c *GroundConstraint) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *GroundConstraint) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

// Variables returns the grounded points' coordinates.
func (c *GroundConstraint) Variables() []*math.Scalar {
	out := make([]*math.Scalar, 0, len(c.pts)*2)
	for _, p := range c.pts {
		out = append(out, &p.X, &p.Y)
	}
	return out
}

// OffsetConstraint holds line L2 parallel to L1 at a fixed signed perpendicular distance
// (the geometric relation behind an offset; pairs with a driven offset dimension). When live is
// non-nil the distance is read from it each solve — an offset segment DRIVEN by one shared offset
// dimension, so editing the dimension moves a whole uniform-offset loop (Dist keeps the last live
// value so a serialized constraint reloads frozen at that distance, live being a closure).
type OffsetConstraint struct {
	constraintBase
	L1, L2 *Line
	Dist   float64
	live   func() float64
}

// currentDist is the target signed distance: the live value when driven, else the fixed Dist.
func (c *OffsetConstraint) currentDist() float64 {
	if c.live != nil {
		return c.live()
	}
	return c.Dist
}

// AddOffset constrains L2 parallel to L1 at the signed perpendicular distance dist.
func (g *GeometricConstraints) AddOffset(l1, l2 *Line, dist float64) *OffsetConstraint {
	c := &OffsetConstraint{constraintBase: newConstraint(), L1: l1, L2: l2, Dist: dist}
	g.add(c)
	return c
}

// AddOffsetDriven constrains L2 parallel to L1 at the signed perpendicular distance live() reads each
// solve — the binding that keeps a uniform-offset loop's segments tied to one driving dimension. Dist
// snapshots the initial value so a serialized (closure-less) reload stays at that distance.
func (g *GeometricConstraints) AddOffsetDriven(l1, l2 *Line, live func() float64) *OffsetConstraint {
	c := &OffsetConstraint{constraintBase: newConstraint(), L1: l1, L2: l2, Dist: live(), live: live}
	g.add(c)
	return c
}

// residualAD: L2 stays parallel to L1 (cross of directions zero) and L2.A holds a fixed
// signed perpendicular distance from L1's line. The signed distance is d1×(L2.A−L1.A) /
// |d1|, the projection of L2.A−L1.A onto L1's left normal.
func (c *OffsetConstraint) residualAD(v []ad.Number) []ad.Number {
	a1, b1, a2, b2 := adTwoLines(v)
	d1, d2 := b1.Sub(a1), b2.Sub(a2)
	// Parallelism as the sine of the angle (scale-invariant, #1418); the perpendicular
	// distance of L2.A from L1 was already a true distance.
	parallel := adSineAngle(d1, d2)
	dist := adSignedPerpDistance(d1, a2.Sub(a1))
	return []ad.Number{parallel, dist.AddConst(-c.currentDist())}
}

// Residuals reports the parallel error and the perpendicular-distance error.
func (c *OffsetConstraint) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *OffsetConstraint) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

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

// residualAD: v = [Seed.X, Seed.Y, Member.X, Member.Y]. The member tracks the seed at a
// fixed offset; the offset (frozen or spacing-driven) is constant w.r.t. the solver
// variables, so it enters as an additive constant.
func (c *PatternConstraint) residualAD(v []ad.Number) []ad.Number {
	dx, dy := c.dx, c.dy
	if c.live != nil {
		dx, dy = c.live()
	}
	return []ad.Number{v[2].Sub(v[0]).AddConst(-dx), v[3].Sub(v[1]).AddConst(-dy)}
}

// Residuals reports the member's drift from seed + the (frozen or live) offset.
func (c *PatternConstraint) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *PatternConstraint) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

// Variables returns the seed and member coordinates.
func (c *PatternConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{&c.Seed.X, &c.Seed.Y, &c.Member.X, &c.Member.Y}
}
