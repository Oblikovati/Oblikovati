// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// filletTangencyConstraint is the internal, system-owned constraint that keeps a fillet arc
// tangent to one of the two edges it blends, at their shared point. It is created with the
// fillet (see [Sketch.AddFillet]), lives and dies with the arc, and is never user-visible or
// deletable.
//
// Tangency is expressed as the radius being perpendicular to the edge at the tangent point —
// (shared − center) · (far − shared) = 0 — NOT as the usual line-to-circle "perpendicular
// distance == radius" residual ([TangentConstraint]). That residual DEGENERATES for a fillet:
// the arc's endpoint is the tangent point and lies on the line, so ∂(perp-dist)/∂center and
// ∂(radius)/∂center are the same direction and the gradient vanishes at tangency, leaving the
// fillet arc under-ranked (never reaching DOF 0). The radius-perpendicular form stays full-rank
// there, so a filleted corner constrains cleanly (Oblikovati.AddIns.PartDesigner#69).
type filletTangencyConstraint struct {
	constraintBase
	center, shared, far *Point
}

// newFilletTangency builds the tangency constraint for a fillet arc against one edge: shared is
// the arc endpoint that lies on the edge (its tangent point), far the edge's other endpoint.
func newFilletTangency(center, shared, far *Point) *filletTangencyConstraint {
	return &filletTangencyConstraint{constraintBase: newConstraint(), center: center, shared: shared, far: far}
}

// residualAD: v = [center.X, center.Y, shared.X, shared.Y, far.X, far.Y]. The radius vector
// (shared − center) is perpendicular to the edge direction (far − shared) at tangency, so their
// dot product is zero.
func (c *filletTangencyConstraint) residualAD(v []ad.Number) []ad.Number {
	center := ad.V2(v[0], v[1])
	shared := ad.V2(v[2], v[3])
	far := ad.V2(v[4], v[5])
	radial := shared.Sub(center)
	edge := far.Sub(shared)
	return []ad.Number{radial.Dot(edge)}
}

func (c *filletTangencyConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *filletTangencyConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}

func (c *filletTangencyConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{
		&c.center.X, &c.center.Y,
		&c.shared.X, &c.shared.Y,
		&c.far.X, &c.far.Y,
	}
}

// Deletable implements [NonDeletable]: fillet tangency is structural to the blend and cannot be
// removed on its own — it is dropped when its arc is.
func (c *filletTangencyConstraint) Deletable() bool { return false }
