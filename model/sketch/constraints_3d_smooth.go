// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Tangent (G1) and Smooth (G2) constraints for 3D curves — the 3D counterparts of
// constraints_smooth.go, closing the types-declared-but-unimplemented gap of issue
// #142 (M22-F05 / PBI-236). Both join two curves at endpoints: Tangent drives the
// join coincident (G0) with collinear tangents (G1); Smooth additionally matches the
// orientation-free curvature vector (G2). As in 2D, two analytic curves can only be
// G2 when cocircular, so Smooth is meaningful when at least one side is a spline —
// the app tool and the router enforce that.

// SmoothCurve3D is a 3D curve that can report its tangent and curvature at one of its
// endpoints — a Line3D, Arc3D, or Spline3D. The interface is sealed (the methods are
// unexported) so only those types satisfy it.
type SmoothCurve3D interface {
	Entity
	// smoothFrame3D returns the tangent direction and the curvature vector at endpoint
	// p (the curvature vector points toward the centre of curvature, |v| = curvature).
	// ok is false when p is not one of the curve's endpoints (or the curve is
	// degenerate there).
	smoothFrame3D(p *Point3D) (tangent, curvature math.Vector3, ok bool)
	// smoothVars3D returns the scalar DOFs of the curve's defining points.
	smoothVars3D() []*math.Scalar
	// smoothEnds3D returns the curve's free endpoints (empty for a closed curve).
	smoothEnds3D() []*Point3D
}

// Tangent3D joins curves C1 and C2 with collinear tangents (G1) at endpoints P1 and
// P2, which it also drives coincident (G0). Residuals: position match (3, G0),
// tangent cross product (3, G1 — rank 2; the solver's rank analysis absorbs the
// redundant component, as with Parallel3D).
type Tangent3D struct {
	constraintBase
	C1, C2 SmoothCurve3D
	P1, P2 *Point3D
}

// NewTangent3D constrains c1 (at endpoint p1) and c2 (at endpoint p2) to a tangent
// (G1) join.
func NewTangent3D(c1, c2 SmoothCurve3D, p1, p2 *Point3D) *Tangent3D {
	return &Tangent3D{constraintBase: newConstraint(), C1: c1, C2: c2, P1: p1, P2: p2}
}

func (c *Tangent3D) Residuals() []float64 {
	t1, _, ok1 := c.C1.smoothFrame3D(c.P1)
	t2, _, ok2 := c.C2.smoothFrame3D(c.P2)
	if !ok1 || !ok2 {
		return []float64{1, 1, 1, 1, 1, 1} // p1/p2 not endpoints: cannot be satisfied
	}
	cr := t1.Cross(t2)
	return []float64{
		float64(c.P1.X - c.P2.X), float64(c.P1.Y - c.P2.Y), float64(c.P1.Z - c.P2.Z), // G0
		float64(cr.X), float64(cr.Y), float64(cr.Z), // G1
	}
}

func (c *Tangent3D) Variables() []*math.Scalar {
	return append(c.C1.smoothVars3D(), c.C2.smoothVars3D()...)
}

// Smooth3D joins curves C1 and C2 curvature-continuously (G2) at endpoints P1 and P2:
// the Tangent3D residuals plus the curvature-vector match.
type Smooth3D struct {
	constraintBase
	C1, C2 SmoothCurve3D
	P1, P2 *Point3D
}

// NewSmooth3D constrains c1 (at endpoint p1) and c2 (at endpoint p2) to a smooth (G2)
// join.
func NewSmooth3D(c1, c2 SmoothCurve3D, p1, p2 *Point3D) *Smooth3D {
	return &Smooth3D{constraintBase: newConstraint(), C1: c1, C2: c2, P1: p1, P2: p2}
}

func (c *Smooth3D) Residuals() []float64 {
	t1, k1, ok1 := c.C1.smoothFrame3D(c.P1)
	t2, k2, ok2 := c.C2.smoothFrame3D(c.P2)
	if !ok1 || !ok2 {
		return []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
	}
	cr := t1.Cross(t2)
	return []float64{
		float64(c.P1.X - c.P2.X), float64(c.P1.Y - c.P2.Y), float64(c.P1.Z - c.P2.Z), // G0
		float64(cr.X), float64(cr.Y), float64(cr.Z), // G1
		float64(k1.X - k2.X), float64(k1.Y - k2.Y), float64(k1.Z - k2.Z), // G2
	}
}

func (c *Smooth3D) Variables() []*math.Scalar {
	return append(c.C1.smoothVars3D(), c.C2.smoothVars3D()...)
}

// NearestEndpointPair3D returns the endpoint of c1 and the endpoint of c2 that lie
// closest to each other — how the tangent/smooth tools and the wire API choose the
// join when the caller picks only the two curves (mirrors the 2D Smooth tool's
// nearest-endpoint behaviour). ok is false when either curve has no free endpoint
// (e.g. a closed spline).
func NearestEndpointPair3D(c1, c2 SmoothCurve3D) (p1, p2 *Point3D, ok bool) {
	ends1, ends2 := c1.smoothEnds3D(), c2.smoothEnds3D()
	best := stdmath.Inf(1)
	for _, p := range ends1 {
		for _, q := range ends2 {
			if d := float64(p.Position().DistanceTo(q.Position())); d < best {
				p1, p2, best = p, q, d
			}
		}
	}
	return p1, p2, p1 != nil && p2 != nil
}

// --- per-curve smooth frames ----------------------------------------------------------

// smoothFrame3D for a line: the tangent points from p toward the other endpoint; a
// line is straight, so its curvature vector is zero.
func (l *Line3D) smoothFrame3D(p *Point3D) (math.Vector3, math.Vector3, bool) {
	switch p {
	case l.A:
		return unit3(l.A.Position().VectorTo(l.B.Position())), math.Vector3{}, true
	case l.B:
		return unit3(l.B.Position().VectorTo(l.A.Position())), math.Vector3{}, true
	default:
		return math.Vector3{}, math.Vector3{}, false
	}
}

func (l *Line3D) smoothVars3D() []*math.Scalar { return line3DVars(l) }
func (l *Line3D) smoothEnds3D() []*Point3D     { return []*Point3D{l.A, l.B} }

// smoothFrame3D for an arc: the tangent is perpendicular to the radius at p within
// the arc's plane (normal from center/start/end); the curvature vector points from p
// to the centre with magnitude 1/r. The tangent's sign along the arc is irrelevant —
// both G1 residual forms (cross product) are sign-free.
func (a *Arc3D) smoothFrame3D(p *Point3D) (math.Vector3, math.Vector3, bool) {
	if p != a.Start && p != a.End {
		return math.Vector3{}, math.Vector3{}, false
	}
	center := a.Center.Position()
	normal := center.VectorTo(a.Start.Position()).Cross(center.VectorTo(a.End.Position()))
	radial := center.VectorTo(p.Position())
	r := radial.Length()
	if r < math.DefaultTolerance || normal.Length() < math.DefaultTolerance {
		return math.Vector3{}, math.Vector3{}, false // degenerate arc (collinear or zero radius)
	}
	tangent := unit3(normal.Cross(radial))
	curvature := p.Position().VectorTo(center).Scale(1 / (r * r)) // toward centre, |.|=1/r
	return tangent, curvature, true
}

func (a *Arc3D) smoothVars3D() []*math.Scalar {
	return []*math.Scalar{
		&a.Center.X, &a.Center.Y, &a.Center.Z,
		&a.Start.X, &a.Start.Y, &a.Start.Z,
		&a.End.X, &a.End.Y, &a.End.Z,
	}
}

func (a *Arc3D) smoothEnds3D() []*Point3D { return []*Point3D{a.Start, a.End} }

// smoothFrame3D for a spline: the tangent points from the endpoint toward its
// adjacent defining point; the curvature vector is that of the circle through the
// three terminal points (zero when collinear) — the same representative-polyline
// convention as the 2D spline and the Catmull–Rom sampler.
func (s *Spline3D) smoothFrame3D(p *Point3D) (math.Vector3, math.Vector3, bool) {
	n := len(s.Points)
	if n < 2 || s.Closed {
		return math.Vector3{}, math.Vector3{}, false
	}
	var adjacent, second *Point3D
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
		return math.Vector3{}, math.Vector3{}, false
	}
	tangent := unit3(p.Position().VectorTo(adjacent.Position()))
	if second == nil {
		return tangent, math.Vector3{}, true
	}
	return tangent, curvatureVector3(second.Position(), adjacent.Position(), p.Position()), true
}

func (s *Spline3D) smoothVars3D() []*math.Scalar {
	vars := make([]*math.Scalar, 0, len(s.Points)*3)
	for _, p := range s.Points {
		vars = append(vars, &p.X, &p.Y, &p.Z)
	}
	return vars
}

func (s *Spline3D) smoothEnds3D() []*Point3D {
	if len(s.Points) < 2 || s.Closed {
		return nil
	}
	return []*Point3D{s.Points[0], s.Points[len(s.Points)-1]}
}

// --- geometry helpers -------------------------------------------------------------------

// unit3 returns v scaled to unit length, or the zero vector when v is (near) zero.
func unit3(v math.Vector3) math.Vector3 {
	l := v.Length()
	if l < math.DefaultTolerance {
		return math.Vector3{}
	}
	return v.Scale(1 / l)
}

// curvatureVector3 returns the curvature vector at p of the circle through a, b, p —
// it points from p toward the circle's centre with magnitude 1/R, and is zero when
// the three points are collinear (infinite radius).
func curvatureVector3(a, b, p math.Point3) math.Vector3 {
	center, ok := circumcenter3(a, b, p)
	if !ok {
		return math.Vector3{}
	}
	toCenter := p.VectorTo(center)
	r := toCenter.Length()
	if r < math.DefaultTolerance {
		return math.Vector3{}
	}
	return toCenter.Scale(1 / (r * r))
}

// circumcenter3 returns the centre of the circle through a, b, c in 3D, and false
// when they are (near) collinear. Standard construction: with u = b−a, v = c−a and
// n = u×v, the centre is a + (|u|²(v×n) + |v|²(n×u)) / (2|n|²).
func circumcenter3(a, b, c math.Point3) (math.Point3, bool) {
	u := a.VectorTo(b)
	v := a.VectorTo(c)
	n := u.Cross(v)
	n2 := n.LengthSquared()
	if n2 < math.DefaultTolerance*math.DefaultTolerance {
		return math.Point3{}, false
	}
	offset := v.Cross(n).Scale(u.LengthSquared()).Add(n.Cross(u).Scale(v.LengthSquared())).Scale(1 / (2 * n2))
	return a.TranslateBy(offset), true
}
