// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
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
	// smoothFrameAD is the dual twin of smoothFrame: it also returns the endpoint p as a
	// dual (for the G0 coincidence residual), reading the curve's smoothVars from v
	// starting at off (off advances by len(smoothVars) between the two curves).
	smoothFrameAD(v []ad.Number, off int, p *Point) (point, tangent, curvature ad.Vec2, ok bool)
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

// SmoothCurveEndpoints returns a smooth curve's two endpoints (nil for a degenerate
// spline). The join-selection helpers below live with the constraint so every creation
// surface (app tool, wire router) picks the same join (#1643).
func SmoothCurveEndpoints(c SmoothCurve) (*Point, *Point) {
	switch t := c.(type) {
	case *Line:
		return t.A, t.B
	case *Arc:
		return t.Start, t.End
	case *Spline:
		if len(t.Points) >= 2 {
			return t.Points[0], t.Points[len(t.Points)-1]
		}
	}
	return nil, nil
}

// NearestSmoothJoin returns the closest endpoint of c1 to an endpoint of c2 — the join
// the Smooth constraint should make continuous.
func NearestSmoothJoin(c1, c2 SmoothCurve) (*Point, *Point, bool) {
	a1, b1 := SmoothCurveEndpoints(c1)
	a2, b2 := SmoothCurveEndpoints(c2)
	if a1 == nil || a2 == nil {
		return nil, nil, false
	}
	best1, best2, bestD := a1, a2, stdmath.Inf(1)
	for _, p := range []*Point{a1, b1} {
		for _, q := range []*Point{a2, b2} {
			if d := p.Position().DistanceTo(q.Position()); d < bestD {
				best1, best2, bestD = p, q, d
			}
		}
	}
	return best1, best2, true
}

// HasSplineCurve reports whether any of the curves is a spline. Smooth (G2) needs a
// curve with adjustable end curvature, so — like the reference tool — creation surfaces
// require at least one spline.
func HasSplineCurve(curves []SmoothCurve) bool {
	for _, c := range curves {
		if _, ok := c.(*Spline); ok {
			return true
		}
	}
	return false
}

// residualAD mirrors the float residual over duals: G0 join coincidence (2), G1 tangent
// collinearity (1), G2 curvature-vector match (2). The frames carry their own exact
// derivatives, so the spline circumcircle curvature differentiates by construction.
func (c *SmoothConstraint) residualAD(v []ad.Number) []ad.Number {
	p1, t1, k1, ok1 := c.C1.smoothFrameAD(v, 0, c.P1)
	p2, t2, k2, ok2 := c.C2.smoothFrameAD(v, len(c.C1.smoothVars()), c.P2)
	if !ok1 || !ok2 {
		return adConstResiduals(5, 1) // p1/p2 not endpoints: cannot be satisfied
	}
	return []ad.Number{
		p1.X.Sub(p2.X), p1.Y.Sub(p2.Y), // G0: join coincident
		t1.Cross(t2),                   // G1: tangents collinear
		k1.X.Sub(k2.X), k1.Y.Sub(k2.Y), // G2: equal curvature vector
	}
}
func (c *SmoothConstraint) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *SmoothConstraint) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

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

// --- dual smooth frames (#1417) -----------------------------------------------------

// smoothFrameAD for a line: tangent from p toward the other endpoint, zero curvature.
func (l *Line) smoothFrameAD(v []ad.Number, off int, p *Point) (ad.Vec2, ad.Vec2, ad.Vec2, bool) {
	a, b := ad.V2(v[off], v[off+1]), ad.V2(v[off+2], v[off+3])
	switch p {
	case l.A:
		return a, adUnit2(b.Sub(a)), adZeroVec2(), true
	case l.B:
		return b, adUnit2(a.Sub(b)), adZeroVec2(), true
	}
	return ad.Vec2{}, ad.Vec2{}, ad.Vec2{}, false
}

// smoothFrameAD for an arc: tangent ⟂ the radius at p, curvature vector p→centre with
// magnitude 1/r (computed as (centre−p)/r²).
func (a *Arc) smoothFrameAD(v []ad.Number, off int, p *Point) (ad.Vec2, ad.Vec2, ad.Vec2, bool) {
	center := ad.V2(v[off], v[off+1])
	var pt ad.Vec2
	switch p {
	case a.Start:
		pt = ad.V2(v[off+2], v[off+3])
	case a.End:
		pt = ad.V2(v[off+4], v[off+5])
	default:
		return ad.Vec2{}, ad.Vec2{}, ad.Vec2{}, false
	}
	radial := pt.Sub(center) // centre → p
	rsq := radial.Dot(radial)
	if rsq.Val() < math.DefaultTolerance*math.DefaultTolerance {
		return ad.Vec2{}, ad.Vec2{}, ad.Vec2{}, false
	}
	tangent := adUnit2(ad.V2(radial.Y.Neg(), radial.X))
	curvature := center.Sub(pt).MulN(ad.Const(1).Div(rsq))
	return pt, tangent, curvature, true
}

// smoothFrameAD for a spline: tangent from the endpoint toward its adjacent control
// point; curvature the circle through the three terminal control points.
func (s *Spline) smoothFrameAD(v []ad.Number, off int, p *Point) (ad.Vec2, ad.Vec2, ad.Vec2, bool) {
	pi, ai, si, ok := splineTerminalIndices(s, p)
	if !ok {
		return ad.Vec2{}, ad.Vec2{}, ad.Vec2{}, false
	}
	pt, adj := adSplinePoint(v, off, pi), adSplinePoint(v, off, ai)
	tangent := adUnit2(adj.Sub(pt))
	if si < 0 {
		return pt, tangent, adZeroVec2(), true
	}
	return pt, tangent, adCurvatureVector(adSplinePoint(v, off, si), adj, pt), true
}

// splineTerminalIndices returns the indices (in Points order) of the endpoint p, its
// adjacent point, and the second-adjacent point (si<0 when the spline has fewer than
// three points, i.e. no curvature), or ok=false when p is not a spline endpoint.
func splineTerminalIndices(s *Spline, p *Point) (pi, ai, si int, ok bool) {
	n := len(s.Points)
	if n < 2 {
		return 0, 0, -1, false
	}
	switch p {
	case s.Points[0]:
		si = -1
		if n >= 3 {
			si = 2
		}
		return 0, 1, si, true
	case s.Points[n-1]:
		si = -1
		if n >= 3 {
			si = n - 3
		}
		return n - 1, n - 2, si, true
	}
	return 0, 0, -1, false
}

// adSplinePoint returns control point k of a spline as a dual, from its seeded segment.
func adSplinePoint(v []ad.Number, off, k int) ad.Vec2 {
	return ad.V2(v[off+2*k], v[off+2*k+1])
}

// adCurvatureVector is the dual twin of curvatureVector: the curvature vector at p of the
// circle through a, b, p (zero when collinear or degenerate).
func adCurvatureVector(a, b, p ad.Vec2) ad.Vec2 {
	center, ok := adCircumcenter(a, b, p)
	if !ok {
		return adZeroVec2()
	}
	toCenter := center.Sub(p)
	rsq := toCenter.Dot(toCenter)
	if rsq.Val() < math.DefaultTolerance*math.DefaultTolerance {
		return adZeroVec2()
	}
	return toCenter.MulN(ad.Const(1).Div(rsq))
}

// adCircumcenter is the dual twin of circumcenter: the centre of the circle through a, b,
// c, and false when they are collinear (the determinant vanishes).
func adCircumcenter(a, b, c ad.Vec2) (ad.Vec2, bool) {
	d := a.X.Mul(b.Y.Sub(c.Y)).Add(b.X.Mul(c.Y.Sub(a.Y))).Add(c.X.Mul(a.Y.Sub(b.Y))).Scale(2)
	if stdmath.Abs(d.Val()) < math.DefaultTolerance {
		return ad.Vec2{}, false
	}
	a2, b2, c2 := a.Dot(a), b.Dot(b), c.Dot(c)
	ux := a2.Mul(b.Y.Sub(c.Y)).Add(b2.Mul(c.Y.Sub(a.Y))).Add(c2.Mul(a.Y.Sub(b.Y))).Div(d)
	uy := a2.Mul(c.X.Sub(b.X)).Add(b2.Mul(a.X.Sub(c.X))).Add(c2.Mul(b.X.Sub(a.X))).Div(d)
	return ad.V2(ux, uy), true
}
