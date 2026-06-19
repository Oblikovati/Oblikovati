// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// Constraint and dimension application. The interactive tools (sketch_constraint_tools.go)
// collect the picked entities and call these functions, which resolve them to the typed
// inputs the model/sketch API needs, add the constraint, re-solve the sketch, and clear
// the selection. A mismatched pick set returns a descriptive error.

// errNeed builds the "wrong selection" error for a constraint.
func errNeed(constraint, want string) error {
	return fmt.Errorf("%s needs %s", constraint, want)
}

func filterPoints(ents []sketch.Entity) []*sketch.Point {
	var out []*sketch.Point
	for _, e := range ents {
		if p, ok := e.(*sketch.Point); ok {
			out = append(out, p)
		}
	}
	return out
}

func filterLines(ents []sketch.Entity) []*sketch.Line {
	var out []*sketch.Line
	for _, e := range ents {
		if l, ok := e.(*sketch.Line); ok {
			out = append(out, l)
		}
	}
	return out
}

func filterCircles(ents []sketch.Entity) []*sketch.Circle {
	var out []*sketch.Circle
	for _, e := range ents {
		if c, ok := e.(*sketch.Circle); ok {
			out = append(out, c)
		}
	}
	return out
}

// circularCurvesFrom returns the circular curves (circles and arcs) among ents, in pick
// order, as the polymorphic [sketch.CircularCurve] the circular constraints accept — so
// concentric/tangent/equal/coincident treat an arc exactly like a circle.
func circularCurvesFrom(ents []sketch.Entity) []sketch.CircularCurve {
	var out []sketch.CircularCurve
	for _, e := range ents {
		switch c := e.(type) {
		case *sketch.Circle:
			out = append(out, c)
		case *sketch.Arc:
			out = append(out, c)
		}
	}
	return out
}

// pointPairFrom returns two driving points: two picked points, or one picked line's
// endpoints.
func pointPairFrom(ents []sketch.Entity) (*sketch.Point, *sketch.Point, bool) {
	if pts := filterPoints(ents); len(pts) >= 2 {
		return pts[0], pts[1], true
	}
	if lns := filterLines(ents); len(lns) >= 1 {
		return lns[0].A, lns[0].B, true
	}
	return nil, nil, false
}

// afterConstraint re-solves the sketch and clears the selection (the geometry changed);
// the common tail of every constraint/dimension application.
func (s *Session) afterConstraint() error {
	if s.activeSketch == nil {
		return errors.New("no active sketch")
	}
	s.activeSketch.Solve()
	s.Select(nil)
	return nil
}

// geom returns the active sketch's geometric-constraint collection.
func (s *Session) geom() *sketch.GeometricConstraints {
	return s.activeSketch.GeometricConstraints()
}

// applyCoincident makes two points coincident, or a point lie on a line/circle (the
// point-to-curve coincidence) — Inventor's polymorphic Coincident.
func applyCoincident(s *Session, ents []sketch.Entity) error {
	pts := filterPoints(ents)
	if len(pts) >= 2 {
		s.geom().AddCoincident(pts[0], pts[1])
		return s.afterConstraint()
	}
	if len(pts) >= 1 {
		if l := filterLines(ents); len(l) >= 1 {
			s.geom().AddPointOnLine(pts[0], l[0])
			return s.afterConstraint()
		}
		if c := circularCurvesFrom(ents); len(c) >= 1 {
			s.geom().AddPointOnCircle(pts[0], c[0])
			return s.afterConstraint()
		}
	}
	return errNeed("coincident", "two points, or a point and a line/circle/arc")
}

func applyHorizontal(s *Session, ents []sketch.Entity) error {
	a, b, ok := pointPairFrom(ents)
	if !ok {
		return errNeed("horizontal", "a line or two points")
	}
	s.geom().AddHorizontal(a, b)
	return s.afterConstraint()
}

func applyVertical(s *Session, ents []sketch.Entity) error {
	a, b, ok := pointPairFrom(ents)
	if !ok {
		return errNeed("vertical", "a line or two points")
	}
	s.geom().AddVertical(a, b)
	return s.afterConstraint()
}

func applyParallel(s *Session, ents []sketch.Entity) error {
	if l := filterLines(ents); len(l) >= 2 {
		s.geom().AddParallel(l[0], l[1])
		return s.afterConstraint()
	}
	return errNeed("parallel", needTwoLines)
}

func applyPerpendicular(s *Session, ents []sketch.Entity) error {
	if l := filterLines(ents); len(l) >= 2 {
		s.geom().AddPerpendicular(l[0], l[1])
		return s.afterConstraint()
	}
	return errNeed("perpendicular", needTwoLines)
}

func applyCollinear(s *Session, ents []sketch.Entity) error {
	if l := filterLines(ents); len(l) >= 2 {
		s.geom().AddCollinear(l[0], l[1])
		return s.afterConstraint()
	}
	return errNeed("collinear", needTwoLines)
}

func applyEqual(s *Session, ents []sketch.Entity) error {
	if l := filterLines(ents); len(l) >= 2 {
		s.geom().AddEqualLength(l[0], l[1])
		return s.afterConstraint()
	}
	if c := circularCurvesFrom(ents); len(c) >= 2 {
		s.geom().AddEqualRadius(c[0], c[1])
		return s.afterConstraint()
	}
	return errNeed("equal", "two lines, or two circles/arcs")
}

func applyConcentric(s *Session, ents []sketch.Entity) error {
	if c := circularCurvesFrom(ents); len(c) >= 2 {
		s.geom().AddConcentric(c[0], c[1])
		return s.afterConstraint()
	}
	return errNeed("concentric", "two circles or arcs")
}

// applyTangent makes a line tangent to a circle/arc, or two circles/arcs tangent to each
// other — Inventor's polymorphic Tangent.
func applyTangent(s *Session, ents []sketch.Entity) error {
	curves := circularCurvesFrom(ents)
	if l := filterLines(ents); len(l) >= 1 && len(curves) >= 1 {
		s.geom().AddTangent(l[0], curves[0])
		return s.afterConstraint()
	}
	if len(curves) >= 2 {
		s.geom().AddCircularTangent(curves[0], curves[1])
		return s.afterConstraint()
	}
	return errNeed("tangent", "a line and a circle/arc, or two circles/arcs")
}

// smoothCurvesFrom returns the smooth-capable curves (lines, arcs, splines) among ents.
func smoothCurvesFrom(ents []sketch.Entity) []sketch.SmoothCurve {
	var out []sketch.SmoothCurve
	for _, e := range ents {
		if c, ok := e.(sketch.SmoothCurve); ok {
			out = append(out, c)
		}
	}
	return out
}

// hasSpline reports whether any of the curves is a spline. Smooth (G2) needs a curve
// with adjustable curvature, so — like Inventor — the tool requires at least one spline.
func hasSpline(curves []sketch.SmoothCurve) bool {
	for _, c := range curves {
		if _, ok := c.(*sketch.Spline); ok {
			return true
		}
	}
	return false
}

// curveEndpoints returns a smooth curve's two endpoints (nil for a degenerate spline).
func curveEndpoints(c sketch.SmoothCurve) (*sketch.Point, *sketch.Point) {
	switch t := c.(type) {
	case *sketch.Line:
		return t.A, t.B
	case *sketch.Arc:
		return t.Start, t.End
	case *sketch.Spline:
		if len(t.Points) >= 2 {
			return t.Points[0], t.Points[len(t.Points)-1]
		}
	}
	return nil, nil
}

// nearestEndpointPair returns the closest endpoint of c1 to an endpoint of c2 — the join
// the Smooth constraint should make continuous.
func nearestEndpointPair(c1, c2 sketch.SmoothCurve) (*sketch.Point, *sketch.Point, bool) {
	a1, b1 := curveEndpoints(c1)
	a2, b2 := curveEndpoints(c2)
	if a1 == nil || a2 == nil {
		return nil, nil, false
	}
	best1, best2, bestD := a1, a2, stdmath.Inf(1)
	for _, p := range []*sketch.Point{a1, b1} {
		for _, q := range []*sketch.Point{a2, b2} {
			if d := p.Position().DistanceTo(q.Position()); d < bestD {
				best1, best2, bestD = p, q, d
			}
		}
	}
	return best1, best2, true
}

// applySmooth joins two curves with a smooth (G2, curvature-continuous) constraint at
// their nearest endpoints — Inventor's Smooth, which needs at least one spline.
func applySmooth(s *Session, ents []sketch.Entity) error {
	curves := smoothCurvesFrom(ents)
	if len(curves) >= 2 && hasSpline(curves) {
		if p1, p2, ok := nearestEndpointPair(curves[0], curves[1]); ok {
			s.geom().AddSmooth(curves[0], curves[1], p1, p2)
			return s.afterConstraint()
		}
	}
	return errNeed("smooth", "a spline and an adjacent curve (line/arc/spline)")
}

// applySymmetry mirrors two points across a line: pick the two points and the symmetry
// axis (any order — the line is the axis, the points are mirrored about it).
func applySymmetry(s *Session, ents []sketch.Entity) error {
	pts, lines := filterPoints(ents), filterLines(ents)
	if len(pts) >= 2 && len(lines) >= 1 {
		s.geom().AddSymmetry(pts[0], pts[1], lines[0])
		return s.afterConstraint()
	}
	return errNeed("symmetry", "two points and a line (the mirror axis)")
}

func applyFix(s *Session, ents []sketch.Entity) error {
	pts := filterPoints(ents)
	for _, l := range filterLines(ents) {
		pts = append(pts, l.A, l.B)
	}
	if len(pts) == 0 {
		return errNeed("fix", "a point or line")
	}
	for _, p := range pts {
		s.geom().AddFix(p)
	}
	return s.afterConstraint()
}

// applyDimension adds the dimension implied by the picked entities: a circle's radius,
// the angle between two lines, or a distance between two points (or a line's length),
// created at the current measured value in the document's units. The new dimension is
// held as the session's pending dimension so the UI can immediately prompt for a value
// to drive the geometry (Inventor's edit-on-place flow).
func applyDimension(s *Session, ents []sketch.Entity) error {
	if s.activeSketch == nil {
		return errors.New("no active sketch")
	}
	created, err := addDimensionFor(s.activeSketch.DimensionConstraints(), s.DocumentUnits(), ents)
	if err != nil {
		return err
	}
	s.pendingDim = created
	return s.afterConstraint()
}

// addDimensionFor creates the dimension implied by the picked entities — a circle's
// radius, the angle between two lines, or a distance between two points (or a line's
// length) — at the current measured value in the document's units.
func addDimensionFor(dims *sketch.DimensionConstraints, units param.UnitsOfMeasure, ents []sketch.Entity) (*sketch.DimensionConstraint, error) {
	switch {
	case len(filterCircles(ents)) >= 1:
		c := filterCircles(ents)[0]
		return dims.AddRadius(c, lengthExpr(units, c.Radius))
	case len(filterLines(ents)) >= 2:
		l := filterLines(ents)
		return dims.AddAngle(l[0], l[1], angleExpr(units, lineAngle(l[0], l[1])))
	default:
		a, b, ok := pointPairFrom(ents)
		if !ok {
			return nil, errNeed("dimension", "points, a line, a circle, or two lines")
		}
		return dims.AddDistance(a, b, lengthExpr(units, a.Position().DistanceTo(b.Position())))
	}
}

// lengthExpr formats a database-unit length as a parseable expression in the document's
// length unit (e.g. 2.5 cm → "25 mm").
func lengthExpr(units param.UnitsOfMeasure, dbValue float64) string {
	return units.Format(param.Quantity{Value: dbValue, Unit: param.Length})
}

// angleExpr formats a radian angle as a parseable expression in the document's angle
// unit (e.g. "30 deg").
func angleExpr(units param.UnitsOfMeasure, radians float64) string {
	return units.Format(param.Quantity{Value: radians, Unit: param.Angle})
}

// lineAngle returns the angle (radians, [0,π]) between two lines.
func lineAngle(l1, l2 *sketch.Line) float64 {
	d1 := l1.A.Position().VectorTo(l1.B.Position())
	d2 := l2.A.Position().VectorTo(l2.B.Position())
	return stdmath.Atan2(stdmath.Abs(d1.Cross(d2)), d1.Dot(d2))
}
