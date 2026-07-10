// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/model/sketch"
)

// twoLinesOf resolves exactly two line refs, returning ok=false (with an error) on a bad
// count so callers can surface it.
func twoLinesOf(sk *sketch.Sketch, refs []uint64) (*sketch.Line, *sketch.Line, bool, error) {
	if len(refs) != 2 {
		return nil, nil, false, fmt.Errorf("sketch.addConstraint: need 2 line refs, got %d", len(refs))
	}
	l1, err := lineRef(sk, refs[0])
	if err != nil {
		return nil, nil, false, err
	}
	l2, err := lineRef(sk, refs[1])
	if err != nil {
		return nil, nil, false, err
	}
	return l1, l2, true, nil
}

// currentPerpDistance is the signed perpendicular distance of line l2's start point from
// line l1 (using l1's left normal) — the value an offset constraint freezes.
func currentPerpDistance(l1, l2 *sketch.Line) float64 {
	a, b := l1.StartPoint().Position(), l1.EndPoint().Position()
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	length := stdmath.Hypot(dx, dy)
	if length == 0 {
		return 0
	}
	nx, ny := -dy/length, dx/length
	w := l2.StartPoint().Position()
	return float64(w.X-a.X)*nx + float64(w.Y-a.Y)*ny
}

// singleLineOrAlign routes the plain horizontal/vertical kinds by operand count: one line ref
// makes that line horizontal/vertical (the single-entity form), two point refs level the points
// (the align form) — Inventor's distinct AddHorizontal(line) vs AddHorizontalAlign(p,p) (#1871).
func singleLineOrAlign(sk *sketch.Sketch, refs []uint64, addLine func(*sketch.Line) sketch.Constraint, addAlign func(a, b *sketch.Point) sketch.Constraint) (sketch.Constraint, bool, error) {
	switch len(refs) {
	case 1:
		l, err := lineRef(sk, refs[0])
		if err != nil {
			return nil, true, fmt.Errorf("sketch.addConstraint: single-entity horizontal/vertical needs a line ref: %w", err)
		}
		return addLine(l), true, nil
	case 2:
		a, b, err := twoPointRefs(sk, refs)
		if err != nil {
			return nil, true, err
		}
		return addAlign(a, b), true, nil
	default:
		return nil, true, fmt.Errorf("sketch.addConstraint: horizontal/vertical needs 1 line ref or 2 point refs, got %d", len(refs))
	}
}

// singleEntityHVOrAlign routes horizontal/vertical by operand: a single ellipse ref rotates that
// ellipse's axis to horizontal/vertical (major/minor per the flag), otherwise it falls to the
// line-or-align form (#1879 AC2, #1871).
func singleEntityHVOrAlign(sk *sketch.Sketch, refs []uint64, major bool, addEllipse func(sketch.AxisOperand) sketch.Constraint, addLine func(*sketch.Line) sketch.Constraint, addAlign func(a, b *sketch.Point) sketch.Constraint) (sketch.Constraint, bool, error) {
	if len(refs) == 1 && isEllipseRef(sk, refs[0]) {
		op, err := axisOperandRef(sk, refs[0], major)
		if err != nil {
			return nil, true, err
		}
		return addEllipse(op), true, nil
	}
	return singleLineOrAlign(sk, refs, addLine, addAlign)
}

// axisOrLineRelation applies an ellipse-axis relation when either operand is an ellipse
// (per-operand major/minor from the flags), else the plain two-line relation (#1879).
func axisOrLineRelation(sk *sketch.Sketch, refs []uint64, flags axisFlags, addAxis func(a, b sketch.AxisOperand) sketch.Constraint, addLine func(a, b *sketch.Line) sketch.Constraint) (sketch.Constraint, bool, error) {
	if len(refs) != 2 || (!isEllipseRef(sk, refs[0]) && !isEllipseRef(sk, refs[1])) {
		return withTwoLines(sk, refs, addLine)
	}
	a, err := axisOperandRef(sk, refs[0], flags.one)
	if err != nil {
		return nil, true, err
	}
	b, err := axisOperandRef(sk, refs[1], flags.two)
	if err != nil {
		return nil, true, err
	}
	return addAxis(a, b), true, nil
}

// isEllipseRef reports whether an id resolves to an ellipse or elliptical arc.
func isEllipseRef(sk *sketch.Sketch, ref uint64) bool {
	e, ok := sk.EntityByID(sketch.ID(ref))
	if !ok {
		return false
	}
	_, ok = sketch.EllipseAxisOf(e, true)
	return ok
}

// axisOperandRef resolves an id to an axis-relation operand: an ellipse's major/minor axis, or a
// line's direction (a single typed pick, not a type switch — archguard I1).
func axisOperandRef(sk *sketch.Sketch, ref uint64, major bool) (sketch.AxisOperand, error) {
	e, ok := sk.EntityByID(sketch.ID(ref))
	if !ok {
		return nil, fmt.Errorf("sketch.addConstraint: no entity with id %d", ref)
	}
	if op, ok := sketch.EllipseAxisOf(e, major); ok {
		return op, nil
	}
	l, ok := e.(*sketch.Line)
	if !ok {
		return nil, fmt.Errorf("sketch.addConstraint: entity %d (%T) is neither a line nor an ellipse", ref, e)
	}
	return sketch.LineAxis(l), nil
}

// withTwoPoints resolves two point refs and applies a two-point constraint factory.
func withTwoPoints(sk *sketch.Sketch, refs []uint64, add func(a, b *sketch.Point) sketch.Constraint) (sketch.Constraint, bool, error) {
	a, b, err := twoPointRefs(sk, refs)
	if err != nil {
		return nil, true, err
	}
	return add(a, b), true, nil
}

// withTwoLines resolves two line refs and applies a two-line constraint factory.
func withTwoLines(sk *sketch.Sketch, refs []uint64, add func(a, b *sketch.Line) sketch.Constraint) (sketch.Constraint, bool, error) {
	if len(refs) != 2 {
		return nil, true, fmt.Errorf("sketch.addConstraint: need 2 line refs, got %d", len(refs))
	}
	a, err := lineRef(sk, refs[0])
	if err != nil {
		return nil, true, err
	}
	b, err := lineRef(sk, refs[1])
	if err != nil {
		return nil, true, err
	}
	return add(a, b), true, nil
}

// withTwoCirculars resolves two circular-curve refs and applies the factory.
func withTwoCirculars(sk *sketch.Sketch, refs []uint64, add func(a, b sketch.CircularCurve) sketch.Constraint) (sketch.Constraint, bool, error) {
	if len(refs) != 2 {
		return nil, true, fmt.Errorf("sketch.addConstraint: need 2 circular-curve refs, got %d", len(refs))
	}
	a, err := circularRef(sk, refs[0])
	if err != nil {
		return nil, true, err
	}
	b, err := circularRef(sk, refs[1])
	if err != nil {
		return nil, true, err
	}
	return add(a, b), true, nil
}

// pointLineConstraint resolves a point + line ref and applies the factory.
func pointLineConstraint(sk *sketch.Sketch, refs []uint64, add func(p *sketch.Point, l *sketch.Line) sketch.Constraint) (sketch.Constraint, bool, error) {
	if len(refs) != 2 {
		return nil, true, fmt.Errorf("sketch.addConstraint: need a point ref + a line ref, got %d", len(refs))
	}
	p, err := pointRef(sk, refs[0])
	if err != nil {
		return nil, true, err
	}
	l, err := lineRef(sk, refs[1])
	if err != nil {
		return nil, true, err
	}
	return add(p, l), true, nil
}

// pointCircleConstraint applies a point-on-circle constraint (point ref + circular ref).
func pointCircleConstraint(sk *sketch.Sketch, refs []uint64) (sketch.Constraint, bool, error) {
	if len(refs) != 2 {
		return nil, true, fmt.Errorf("sketch.addConstraint: pointOnCircle needs a point + circular ref, got %d", len(refs))
	}
	p, err := pointRef(sk, refs[0])
	if err != nil {
		return nil, true, err
	}
	c, err := circularRef(sk, refs[1])
	if err != nil {
		return nil, true, err
	}
	return sk.GeometricConstraints().AddPointOnCircle(p, c), true, nil
}

// tangentConstraint applies a line↔circle tangency, or a circle↔circle tangency when both
// refs are circular curves.
func tangentConstraint(sk *sketch.Sketch, refs []uint64) (sketch.Constraint, bool, error) {
	if len(refs) != 2 {
		return nil, true, fmt.Errorf("sketch.addConstraint: tangent needs 2 refs, got %d", len(refs))
	}
	g := sk.GeometricConstraints()
	if l, err := lineRef(sk, refs[0]); err == nil {
		c, err := circularRef(sk, refs[1])
		if err != nil {
			return nil, true, err
		}
		return g.AddTangent(l, c), true, nil
	}
	c1, err := circularRef(sk, refs[0])
	if err != nil {
		return nil, true, err
	}
	c2, err := circularRef(sk, refs[1])
	if err != nil {
		return nil, true, err
	}
	return g.AddCircularTangent(c1, c2), true, nil
}

// symmetryConstraint makes two entities symmetric about a mirror line: two points, two lines,
// or two circular curves (#1870). The operand kind is discovered by trial resolution (a single
// typed pick each, not a type switch — archguard I1) and dispatched to the matching model
// factory; refs are [entityOne, entityTwo, mirrorLine], matching the enumerate order.
func symmetryConstraint(sk *sketch.Sketch, refs []uint64) (sketch.Constraint, bool, error) {
	if len(refs) != 3 {
		return nil, true, fmt.Errorf("sketch.addConstraint: symmetry needs 2 entity refs + a mirror-line ref, got %d", len(refs))
	}
	about, err := lineRef(sk, refs[2])
	if err != nil {
		return nil, true, fmt.Errorf("sketch.addConstraint: symmetry mirror axis: %w", err)
	}
	g := sk.GeometricConstraints()
	if a, b, ok := twoPointsOpt(sk, refs[0], refs[1]); ok {
		return g.AddSymmetry(a, b, about), true, nil
	}
	if l1, l2, ok := twoLinesOpt(sk, refs[0], refs[1]); ok {
		return g.AddLineSymmetry(l1, l2, about), true, nil
	}
	if c1, c2, ok := twoCircularsOpt(sk, refs[0], refs[1]); ok {
		return g.AddCircularSymmetry(c1, c2, about), true, nil
	}
	return nil, true, fmt.Errorf("sketch.addConstraint: symmetry operands %d and %d are not a matching pair of points, lines, or circular curves", refs[0], refs[1])
}

// midpointConstraint places a point at the midpoint of a line (AddMidpoint) or the arc-length
// midpoint of an arc (AddMidpointToArc, #1872). A circle has no defined midpoint and errors.
func midpointConstraint(sk *sketch.Sketch, refs []uint64) (sketch.Constraint, bool, error) {
	if len(refs) != 2 {
		return nil, true, fmt.Errorf("sketch.addConstraint: midpoint needs a point ref + a line or arc ref, got %d", len(refs))
	}
	p, err := pointRef(sk, refs[0])
	if err != nil {
		return nil, true, err
	}
	g := sk.GeometricConstraints()
	if l, err := lineRef(sk, refs[1]); err == nil {
		return g.AddMidpoint(p, l), true, nil
	}
	a, err := arcRef(sk, refs[1])
	if err != nil {
		return nil, true, fmt.Errorf("sketch.addConstraint: midpoint host %d must be a line or an arc (a circle has no midpoint): %w", refs[1], err)
	}
	return g.AddMidpointToArc(p, a), true, nil
}

// twoPointsOpt resolves two ids as points, reporting ok=false (no error) when either is not a
// point — the trial used to discover a symmetry operand kind.
func twoPointsOpt(sk *sketch.Sketch, r0, r1 uint64) (*sketch.Point, *sketch.Point, bool) {
	a, e0 := pointRef(sk, r0)
	b, e1 := pointRef(sk, r1)
	if e0 != nil || e1 != nil {
		return nil, nil, false
	}
	return a, b, true
}

// twoLinesOpt resolves two ids as lines, reporting ok=false when either is not a line.
func twoLinesOpt(sk *sketch.Sketch, r0, r1 uint64) (*sketch.Line, *sketch.Line, bool) {
	a, e0 := lineRef(sk, r0)
	b, e1 := lineRef(sk, r1)
	if e0 != nil || e1 != nil {
		return nil, nil, false
	}
	return a, b, true
}

// twoCircularsOpt resolves two ids as circular curves, reporting ok=false when either is not one.
func twoCircularsOpt(sk *sketch.Sketch, r0, r1 uint64) (sketch.CircularCurve, sketch.CircularCurve, bool) {
	a, e0 := circularRef(sk, r0)
	b, e1 := circularRef(sk, r1)
	if e0 != nil || e1 != nil {
		return nil, nil, false
	}
	return a, b, true
}

// arcRef resolves an id to an arc (a single typed pick, not a type switch — archguard I1).
func arcRef(sk *sketch.Sketch, id uint64) (*sketch.Arc, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("sketch.addConstraint: no entity with id %d", id)
	}
	a, ok := e.(*sketch.Arc)
	if !ok {
		return nil, fmt.Errorf("sketch.addConstraint: entity %d is %T, want an arc", id, e)
	}
	return a, nil
}

// smoothConstraintFromRefs joins two smooth-capable curves (line/arc/spline, at least
// one spline) with a G2 constraint at their nearest endpoints — the same join policy as
// the app tool (sketch.NearestSmoothJoin). Closed the enum-vs-wire drift of #1643: the
// kind was enumerable and persistable but not wire-creatable.
func smoothConstraintFromRefs(sk *sketch.Sketch, refs []uint64) (sketch.Constraint, bool, error) {
	if len(refs) != 2 {
		return nil, true, fmt.Errorf("sketch.addConstraint: smooth needs 2 curve refs, got %d", len(refs))
	}
	c1, err := smoothCurveRef(sk, refs[0])
	if err != nil {
		return nil, true, err
	}
	c2, err := smoothCurveRef(sk, refs[1])
	if err != nil {
		return nil, true, err
	}
	if !sketch.HasSplineCurve([]sketch.SmoothCurve{c1, c2}) {
		return nil, true, fmt.Errorf("sketch.addConstraint: smooth needs at least one spline, got %T and %T", c1, c2)
	}
	p1, p2, ok := sketch.NearestSmoothJoin(c1, c2)
	if !ok {
		return nil, true, fmt.Errorf("sketch.addConstraint: smooth refs %v have no usable endpoints (degenerate curve)", refs)
	}
	return sk.GeometricConstraints().AddSmooth(c1, c2, p1, p2), true, nil
}

// smoothCurveRef resolves one entity ref to a smooth-capable curve (line/arc/spline).
func smoothCurveRef(sk *sketch.Sketch, ref uint64) (sketch.SmoothCurve, error) {
	e, ok := sk.EntityByID(sketch.ID(ref))
	if !ok {
		return nil, fmt.Errorf("sketch.addConstraint: no entity with id %d", ref)
	}
	c, ok := e.(sketch.SmoothCurve)
	if !ok {
		return nil, fmt.Errorf("sketch.addConstraint: entity %d (%T) is not a smooth-capable curve (line/arc/spline)", ref, e)
	}
	return c, nil
}

// fixConstraint grounds a single point.
func fixConstraint(sk *sketch.Sketch, refs []uint64) (sketch.Constraint, bool, error) {
	if len(refs) != 1 {
		return nil, true, fmt.Errorf("sketch.addConstraint: fix needs 1 point ref, got %d", len(refs))
	}
	p, err := pointRef(sk, refs[0])
	if err != nil {
		return nil, true, err
	}
	return sk.GeometricConstraints().AddFix(p), true, nil
}

// twoPointRefs resolves exactly two point refs.
func twoPointRefs(sk *sketch.Sketch, refs []uint64) (*sketch.Point, *sketch.Point, error) {
	if len(refs) != 2 {
		return nil, nil, fmt.Errorf("sketch.addConstraint: need 2 point refs, got %d", len(refs))
	}
	a, err := pointRef(sk, refs[0])
	if err != nil {
		return nil, nil, err
	}
	b, err := pointRef(sk, refs[1])
	if err != nil {
		return nil, nil, err
	}
	return a, b, nil
}

// pointRef resolves an id to a constrainable point (a curve endpoint/center or a
// standalone sketch point).
func pointRef(sk *sketch.Sketch, id uint64) (*sketch.Point, error) {
	if p, ok := sk.PointByID(sketch.ID(id)); ok {
		return p, nil
	}
	return nil, fmt.Errorf("sketch.addConstraint: no point with id %d", id)
}

// circularRef resolves an id to a circular curve (a circle or an arc).
func circularRef(sk *sketch.Sketch, id uint64) (sketch.CircularCurve, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("sketch.addConstraint: no entity with id %d", id)
	}
	c, ok := e.(sketch.CircularCurve)
	if !ok {
		return nil, fmt.Errorf("sketch.addConstraint: entity %d is %T, want a circle or arc", id, e)
	}
	return c, nil
}
