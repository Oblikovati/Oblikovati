// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati/math"
)

// The Smooth (G2) constraint makes two curves curvature-continuous where they join.
// Where Tangent is G1 (equal tangent direction), Smooth additionally matches the
// curvature *vector* at the shared endpoint, so the join has no kink in curvature —
// Inventor's "Smooth (G2)" constraint. Two analytic curves can only be G2 when they are
// cocircular, so Smooth is meaningful when at least one side is a spline (a curve whose
// end curvature the solver can adjust); the app tool reflects that. Continuity uses the
// orientation-free curvature vector (pointing to the centre of curvature, magnitude =
// curvature), which sidesteps the signed-curvature conventions that get fragile when two
// curves are traversed in opposite directions through the join.

// SmoothCurve is a curve that can report its tangent and curvature at one of its
// endpoints — a Line, Arc, or Spline. The interface is sealed (smoothFrame/smoothVars are
// unexported) so only those types satisfy it.
type SmoothCurve interface {
	Entity
	// smoothFrame returns the tangent direction and the curvature vector at endpoint p
	// (the curvature vector points toward the centre of curvature, |v| = curvature). ok
	// is false when p is not one of the curve's endpoints.
	smoothFrame(p *Point) (tangent, curvature math.Vector2, ok bool)
	// smoothVars returns the scalar DOFs of the curve's defining points.
	smoothVars() []*math.Scalar
}

// SmoothConstraint makes curves C1 and C2 curvature-continuous at endpoints P1 and P2,
// which it also drives coincident (G0). Residuals: position match (2, G0), tangent
// continuity (1, G1), curvature-vector match (2, G2).
type SmoothConstraint struct {
	constraintBase
	C1, C2 SmoothCurve
	P1, P2 *Point
}

// AddSmooth constrains c1 (at endpoint p1) and c2 (at endpoint p2) to a smooth (G2) join.
func (g *GeometricConstraints) AddSmooth(c1, c2 SmoothCurve, p1, p2 *Point) *SmoothConstraint {
	c := &SmoothConstraint{constraintBase: newConstraint(), C1: c1, C2: c2, P1: p1, P2: p2}
	g.add(c)
	return c
}

func (c *SmoothConstraint) Residuals() []float64 {
	t1, k1, ok1 := c.C1.smoothFrame(c.P1)
	t2, k2, ok2 := c.C2.smoothFrame(c.P2)
	if !ok1 || !ok2 {
		return []float64{1, 1, 1, 1, 1} // p1/p2 not endpoints: cannot be satisfied
	}
	return []float64{
		c.P1.X - c.P2.X, c.P1.Y - c.P2.Y, // G0: join coincident
		t1.Cross(t2),             // G1: tangents collinear
		k1.X - k2.X, k1.Y - k2.Y, // G2: equal curvature vector
	}
}

func (c *SmoothConstraint) Variables() []*math.Scalar {
	return append(c.C1.smoothVars(), c.C2.smoothVars()...)
}

// --- per-curve smooth frames --------------------------------------------------------

// smoothFrame for a line: the tangent points from p toward the other endpoint; a line is
// straight, so its curvature vector is zero.
func (l *Line) smoothFrame(p *Point) (math.Vector2, math.Vector2, bool) {
	switch p {
	case l.A:
		return unit2(l.A.Position().VectorTo(l.B.Position())), math.Vector2{}, true
	case l.B:
		return unit2(l.B.Position().VectorTo(l.A.Position())), math.Vector2{}, true
	default:
		return math.Vector2{}, math.Vector2{}, false
	}
}

func (l *Line) smoothVars() []*math.Scalar {
	return []*math.Scalar{&l.A.X, &l.A.Y, &l.B.X, &l.B.Y}
}

// smoothFrame for an arc: the tangent is perpendicular to the radius at p; the curvature
// vector points from p to the centre with magnitude 1/r.
func (a *Arc) smoothFrame(p *Point) (math.Vector2, math.Vector2, bool) {
	if p != a.Start && p != a.End {
		return math.Vector2{}, math.Vector2{}, false
	}
	center := a.Center.Position()
	radial := center.VectorTo(p.Position()) // centre → p, length r
	r := radial.Length()
	if r < math.DefaultTolerance {
		return math.Vector2{}, math.Vector2{}, false
	}
	tangent := math.V2(-radial.Y, radial.X)                       // ⟂ radius
	curvature := p.Position().VectorTo(center).Scale(1 / (r * r)) // toward centre, |.|=1/r
	return unit2(tangent), curvature, true
}

func (a *Arc) smoothVars() []*math.Scalar {
	return []*math.Scalar{&a.Center.X, &a.Center.Y, &a.Start.X, &a.Start.Y, &a.End.X, &a.End.Y}
}

// smoothFrame for a spline: the tangent points from the endpoint toward its adjacent
// control point; the curvature vector is that of the circle through the three terminal
// control points (zero when collinear). Splines with fewer than three points are
// treated as straight.
func (s *Spline) smoothFrame(p *Point) (math.Vector2, math.Vector2, bool) {
	n := len(s.Points)
	if n < 2 {
		return math.Vector2{}, math.Vector2{}, false
	}
	var adjacent, second *Point
	switch p {
	case s.Points[0]:
		adjacent = s.Points[1]
		if n >= 3 {
			second = s.Points[2]
		}
	case s.Points[n-1]:
		adjacent = s.Points[n-2]
		if n >= 3 {
			second = s.Points[n-3]
		}
	default:
		return math.Vector2{}, math.Vector2{}, false
	}
	tangent := unit2(p.Position().VectorTo(adjacent.Position()))
	if second == nil {
		return tangent, math.Vector2{}, true
	}
	return tangent, curvatureVector(second.Position(), adjacent.Position(), p.Position()), true
}

func (s *Spline) smoothVars() []*math.Scalar {
	vars := make([]*math.Scalar, 0, len(s.Points)*2)
	for _, p := range s.Points {
		vars = append(vars, &p.X, &p.Y)
	}
	return vars
}

// --- geometry helpers ---------------------------------------------------------------

// unit2 returns v scaled to unit length, or the zero vector when v is (near) zero.
func unit2(v math.Vector2) math.Vector2 {
	l := v.Length()
	if l < math.DefaultTolerance {
		return math.Vector2{}
	}
	return v.Scale(1 / l)
}

// curvatureVector returns the curvature vector at p of the circle through a, b, p — it
// points from p toward the circle's centre with magnitude 1/R, and is zero when the
// three points are collinear (infinite radius).
func curvatureVector(a, b, p math.Point2) math.Vector2 {
	center, ok := circumcenter(a, b, p)
	if !ok {
		return math.Vector2{}
	}
	toCenter := p.VectorTo(center)
	r := toCenter.Length()
	if r < math.DefaultTolerance {
		return math.Vector2{}
	}
	return toCenter.Scale(1 / (r * r))
}

// circumcenter returns the centre of the circle through a, b, c, and false when they are
// collinear (the determinant vanishes).
func circumcenter(a, b, c math.Point2) (math.Point2, bool) {
	d := 2 * (a.X*(b.Y-c.Y) + b.X*(c.Y-a.Y) + c.X*(a.Y-b.Y))
	if stdmath.Abs(d) < math.DefaultTolerance {
		return math.Point2{}, false
	}
	a2, b2, c2 := originSq(a), originSq(b), originSq(c)
	ux := (a2*(b.Y-c.Y) + b2*(c.Y-a.Y) + c2*(a.Y-b.Y)) / d
	uy := (a2*(c.X-b.X) + b2*(a.X-c.X) + c2*(b.X-a.X)) / d
	return math.P2(ux, uy), true
}

// originSq is the squared distance of p from the origin (a circumcenter sub-term).
func originSq(p math.Point2) float64 { return p.X*p.X + p.Y*p.Y }
