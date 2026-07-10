// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// ArcMidpointConstraint forces a point to the arc-length midpoint of an arc — Inventor's
// AddMidPointToArc, the arc twin of the point-on-line midpoint (#1872). The midpoint is the
// point on the arc's circle that bisects its sweep; it stays associative as the arc's
// endpoints and sweep change.
type ArcMidpointConstraint struct {
	constraintBase
	P *Point
	A *Arc
}

// AddMidpointToArc constrains point p to the midpoint of arc a. The point is seeded to the
// current arc midpoint so the solver starts on the correct one of the circle's two
// perpendicular-bisector intersections (the swept-arc side, not the complementary arc) —
// the same "preserve the picked configuration" policy as the tangent solution selection
// (#1844).
func (g *GeometricConstraints) AddMidpointToArc(p *Point, a *Arc) *ArcMidpointConstraint {
	p.SetPosition(arcMidpoint(a))
	c := &ArcMidpointConstraint{constraintBase: newConstraint(), P: p, A: a}
	g.add(c)
	return c
}

// residualAD: v = [P.X, P.Y, Center.X, Center.Y, Start.X, Start.Y, End.X, End.Y]. The point
// lies on the arc's circle (distance from the centre equals the radius) AND on the line
// through the centre perpendicular to the chord (its component along the chord is zero) — the
// two conditions that pin it to an arc midpoint. Which of the two midpoints is fixed by the
// seed in AddMidpointToArc; the solver tracks that branch continuously.
func (c *ArcMidpointConstraint) residualAD(v []ad.Number) []ad.Number {
	p := ad.V2(v[0], v[1])
	center := ad.V2(v[2], v[3])
	start, end := ad.V2(v[4], v[5]), ad.V2(v[6], v[7])
	radius := start.Sub(center).Length()
	onCircle := p.Sub(center).Length().Sub(radius)
	onBisector := adProjectionAlong(p.Sub(center), end.Sub(start))
	return []ad.Number{onCircle, onBisector}
}
func (c *ArcMidpointConstraint) Residuals() []float64 {
	return adResiduals(c.Variables(), c.residualAD)
}
func (c *ArcMidpointConstraint) Partials() [][]float64 {
	return adPartials(c.Variables(), c.residualAD)
}

func (c *ArcMidpointConstraint) Variables() []*math.Scalar {
	return []*math.Scalar{
		&c.P.X, &c.P.Y,
		&c.A.Center.X, &c.A.Center.Y,
		&c.A.Start.X, &c.A.Start.Y,
		&c.A.End.X, &c.A.End.Y,
	}
}

// arcMidpoint returns the arc-length midpoint of arc a: the point on its circle at half the
// swept angle from the start, measured in the arc's own sweep direction (CCW or CW).
func arcMidpoint(a *Arc) math.Point2 {
	c, s, e := a.Center.Position(), a.Start.Position(), a.End.Position()
	radius := stdmath.Hypot(float64(s.X-c.X), float64(s.Y-c.Y))
	start := stdmath.Atan2(float64(s.Y-c.Y), float64(s.X-c.X))
	end := stdmath.Atan2(float64(e.Y-c.Y), float64(e.X-c.X))
	var mid float64
	if a.CounterClockwise {
		mid = start + wrapTwoPi(end-start)/2
	} else {
		mid = start - wrapTwoPi(start-end)/2
	}
	return math.P2(math.Scalar(float64(c.X)+radius*stdmath.Cos(mid)), math.Scalar(float64(c.Y)+radius*stdmath.Sin(mid)))
}

// wrapTwoPi maps an angle into [0, 2π), the sweep magnitude between two directions.
func wrapTwoPi(a float64) float64 {
	a = stdmath.Mod(a, 2*stdmath.Pi)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
}
