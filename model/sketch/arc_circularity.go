// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// arcCircularityConstraint is the internal, system-owned constraint that keeps an arc's End
// point on its circle: |Center−End| = |Center−Start| (the radius). An arc stores its End as
// an independent free point, so without this constraint driving the arc's radius or a
// tangent (which move Center+Start) would leave End behind — the solved "arc" would no
// longer pass through its own endpoint at constant radius, a silent geometry-correctness bug
// that propagates into the profiles extrude/revolve consume (Oblikovati/Oblikovati#1419).
//
// It is created with the arc (see [Arcs.Add]), lives and dies with it, and is never
// user-visible or deletable — circularity is a property of the arc, not a user relation.
type arcCircularityConstraint struct {
	constraintBase
	arc *Arc
}

// newArcCircularity builds the circularity constraint for an arc.
func newArcCircularity(a *Arc) *arcCircularityConstraint {
	return &arcCircularityConstraint{constraintBase: newConstraint(), arc: a}
}

// residualAD: v = [Center.X, Center.Y, Start.X, Start.Y, End.X, End.Y]. The end and start
// must be equidistant from the center — a distance residual (|End−Center| − |Start−Center|),
// already scale-invariant, so no extra normalisation is needed (#1418).
func (c *arcCircularityConstraint) residualAD(v []ad.Number) []ad.Number {
	center := ad.V2(v[0], v[1])
	start := ad.V2(v[2], v[3])
	end := ad.V2(v[4], v[5])
	return []ad.Number{end.Sub(center).Length().Sub(start.Sub(center).Length())}
}

func (c *arcCircularityConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *arcCircularityConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}

func (c *arcCircularityConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{
		&c.arc.Center.X, &c.arc.Center.Y,
		&c.arc.Start.X, &c.arc.Start.Y,
		&c.arc.End.X, &c.arc.End.Y,
	}
}

// Deletable implements [NonDeletable]: the circularity record is structural and cannot be
// removed on its own — it is dropped when its arc is.
func (c *arcCircularityConstraint) Deletable() bool { return false }
