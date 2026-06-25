// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// The 3D-sketch bend (issue #143, M22-F05/PBI-237): Inventor's
// SketchArcs3D.AddAsBend — a tangent arc fillet inserted at the corner of two
// connected 3D lines. The corner is cut back to the tangent points and a circular
// arc joins them; a Bend3D constraint keeps the arc tangent to both trimmed lines
// (and at its creation radius) as the sketch solves. The bend's arc IS a regular
// Arc3D — as in Inventor, where bends live in the arcs collection.

// AddBend3D inserts a tangent-arc bend of the given radius at the shared corner of
// l1 and l2, returning the new arc. The lines are trimmed in place to the tangent
// points (the arc shares their endpoint objects, so chains stay connected) and the
// maintaining Bend3D constraint is added to the sketch.
//
//	arc, err := sk.AddBend3D(rail1, rail2, 0.5) // 5 mm corner bend
func (s *Sketch3D) AddBend3D(l1, l2 *Line3D, radius float64) (*Arc3D, error) {
	g, err := bendGeometry(l1, l2, radius)
	if err != nil {
		return nil, err
	}
	e1, e2 := splitBendCorner(s, l1, l2, g)
	e1.SetPosition(g.t1)
	e2.SetPosition(g.t2)
	arc := s.addArc3DPts(s.newPoint3D(g.center), e1, e2, true)
	s.geomCons.add(newBend3DBound(arc, l1, l2, e1, e2, radius))
	return arc, nil
}

// bendCorner holds the solved fillet frame: the trim (tangent) positions and the
// arc center.
type bendCorner struct {
	t1, t2, center math.Point3
}

// bendGeometry resolves the shared corner of l1/l2 and computes the tangent-arc
// frame for the given radius, rejecting degenerate corners with the offending
// values (code style: errors carry value + expected shape).
func bendGeometry(l1, l2 *Line3D, radius float64) (bendCorner, error) {
	if radius <= 0 {
		return bendCorner{}, fmt.Errorf("bend: radius %g must be > 0", radius)
	}
	e1, e2, ok := sharedBendCorner(l1, l2)
	if !ok {
		return bendCorner{}, fmt.Errorf("bend: lines %d and %d do not share an endpoint", l1.EntityID(), l2.EntityID())
	}
	c := e1.Position()
	d1, len1 := unitAndLength(c, otherBendEnd(l1, e1).Position())
	d2, len2 := unitAndLength(c, otherBendEnd(l2, e2).Position())
	sinT := float64(d1.Cross(d2).Length())
	cosT := float64(d1.Dot(d2))
	if sinT < float64(math.DefaultTolerance) {
		return bendCorner{}, fmt.Errorf("bend: lines %d and %d are collinear (no corner to fill)", l1.EntityID(), l2.EntityID())
	}
	trim := radius * (1 + cosT) / sinT // r / tan(θ/2)
	if trim >= len1 || trim >= len2 {
		return bendCorner{}, fmt.Errorf("bend: radius %g needs %g trimmed from each line, but lines are %g and %g long", radius, trim, len1, len2)
	}
	return bendCorner{
		t1:     c.TranslateBy(d1.Scale(math.Scalar(trim))),
		t2:     c.TranslateBy(d2.Scale(math.Scalar(trim))),
		center: c.TranslateBy(unit3(d1.Add(d2)).Scale(math.Scalar(radius / stdmath.Sqrt((1-cosT)/2)))), // r / sin(θ/2) along the bisector
	}, nil
}

// sharedBendCorner returns the endpoint of l1 and the endpoint of l2 that form the
// corner — the same point object or two coincident points (chained lines re-create
// endpoint objects, so identity cannot be assumed).
func sharedBendCorner(l1, l2 *Line3D) (*Point3D, *Point3D, bool) {
	for _, p := range []*Point3D{l1.A, l1.B} {
		for _, q := range []*Point3D{l2.A, l2.B} {
			if p == q || coincident3D(p, q) {
				return p, q, true
			}
		}
	}
	return nil, nil, false
}

// splitBendCorner returns the two corner endpoints to trim, splitting a single
// shared point object into two (one per line) so each line can be cut back to its
// own tangent point.
func splitBendCorner(s *Sketch3D, l1, l2 *Line3D, g bendCorner) (*Point3D, *Point3D) {
	e1, e2, _ := sharedBendCorner(l1, l2)
	if e1 != e2 {
		return e1, e2
	}
	split := s.newPoint3D(g.t2)
	if l2.A == e2 {
		l2.A = split
	} else {
		l2.B = split
	}
	return e1, split
}

// otherBendEnd returns the line endpoint that is not p.
func otherBendEnd(l *Line3D, p *Point3D) *Point3D {
	if l.A == p {
		return l.B
	}
	return l.A
}

// unitAndLength returns the unit direction from a to b and the distance.
func unitAndLength(a, b math.Point3) (math.Vector3, float64) {
	v := a.VectorTo(b)
	return unit3(v), float64(v.Length())
}

// Bend3D keeps a bend's arc joined to (G0) and tangent to (G1) its two trimmed
// lines, at the bend radius captured when it was created. The joined endpoints P1/P2
// are usually the very point objects the arc shares with the lines (no G0 rows are
// emitted then — see appendBendJoin3D), but the constraint also accepts split
// points — the wire addConstraint path — which it pulls together on solve.
type Bend3D struct {
	constraintBase
	Arc    *Arc3D
	L1, L2 *Line3D
	P1, P2 *Point3D
	Radius float64
}

// NewBend3D constrains arc as the bend between l1 and l2, joining at the line
// endpoints nearest the arc's start/end. The bend radius is captured from the arc's
// current geometry.
func NewBend3D(arc *Arc3D, l1, l2 *Line3D) (*Bend3D, error) {
	if arc == nil || l1 == nil || l2 == nil {
		return nil, fmt.Errorf("bend: needs an arc and two lines, got %v/%v/%v", arc, l1, l2)
	}
	p1 := nearerEnd(l1, arc.Start.Position())
	p2 := nearerEnd(l2, arc.End.Position())
	return newBend3DBound(arc, l1, l2, p1, p2, float64(arc.Radius())), nil
}

// newBend3DBound builds the constraint over explicit join endpoints — the AddBend3D
// and deserialization constructor (the saved binding must be re-created, not
// re-derived).
func newBend3DBound(arc *Arc3D, l1, l2 *Line3D, p1, p2 *Point3D, radius float64) *Bend3D {
	return &Bend3D{constraintBase: newConstraint(), Arc: arc, L1: l1, L2: l2, P1: p1, P2: p2, Radius: radius}
}

// nearerEnd returns the line endpoint closer to the given position.
func nearerEnd(l *Line3D, to math.Point3) *Point3D {
	if l.A.Position().DistanceSquaredTo(to) <= l.B.Position().DistanceSquaredTo(to) {
		return l.A
	}
	return l.B
}

// residualAD mirrors the float residual over duals. v = [L1.A.xyz, L1.B.xyz, L2.A.xyz,
// L2.B.xyz, Arc.Center.xyz, Arc.Start.xyz, Arc.End.xyz]. Rows: optional G0 joins (each
// skipped when the line endpoint IS the arc endpoint), tangency at each join, the
// end-on-same-circle relation, and the bend radius.
func (c *Bend3D) residualAD(v []ad.Number) []ad.Number {
	center, start, end := adV3(v, 12), adV3(v, 15), adV3(v, 18)
	rs, re := center.Sub(start), center.Sub(end) // start → centre, end → centre
	d1, d2 := adLine3DDir(v, 0), adLine3DDir(v, 6)
	out := appendBendJoin3DAD(nil, c.P1, c.Arc.Start, adBendEndpoint(v, c.L1, c.P1, 0), start)
	out = appendBendJoin3DAD(out, c.P2, c.Arc.End, adBendEndpoint(v, c.L2, c.P2, 6), end)
	return append(out,
		d1.Dot(rs), d2.Dot(re), // tangency at each join
		re.Dot(re).Sub(rs.Dot(rs)),      // end on the same circle as start
		rs.Length().AddConst(-c.Radius), // hold the bend radius
	)
}
func (c *Bend3D) Residuals() []float64  { return adResiduals(c.Variables(), c.residualAD) }
func (c *Bend3D) Partials() [][]float64 { return adPartials(c.Variables(), c.residualAD) }

// adBendEndpoint returns the dual for a line's join endpoint p — its A end (v[off]) or B
// end (v[off+3]).
func adBendEndpoint(v []ad.Number, l *Line3D, p *Point3D, off int) ad.Vec3 {
	if p == l.B {
		return adV3(v, off+3)
	}
	return adV3(v, off)
}

// appendBendJoin3DAD appends the G0 rows pulling a split join endpoint onto its arc
// endpoint, skipping them when both are the same point object (the AddBend3D sharing):
// identically-zero rows carry no Jacobian rank, so they only inflated the redundancy
// count and flagged a fresh bend over-constrained (#145 audit finding on the #143 bend).
func appendBendJoin3DAD(out []ad.Number, p, arcEnd *Point3D, pd, arcd ad.Vec3) []ad.Number {
	if p == arcEnd {
		return out
	}
	d := pd.Sub(arcd)
	return append(out, d.X, d.Y, d.Z)
}

func (c *Bend3D) Variables() []*math.Scalar {
	vars := append(line3DVars(c.L1), line3DVars(c.L2)...)
	return append(vars,
		&c.Arc.Center.X, &c.Arc.Center.Y, &c.Arc.Center.Z,
		&c.Arc.Start.X, &c.Arc.Start.Y, &c.Arc.Start.Z,
		&c.Arc.End.X, &c.Arc.End.Y, &c.Arc.End.Z,
	)
}
