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

// symmetryConstraint makes two points symmetric about a mirror line (point A, point B,
// mirror line), matching the enumerate mapping (A, B, About) in sketch_enumerate.go.
func symmetryConstraint(sk *sketch.Sketch, refs []uint64) (sketch.Constraint, bool, error) {
	if len(refs) != 3 {
		return nil, true, fmt.Errorf("sketch.addConstraint: symmetry needs 2 point refs + a mirror-line ref, got %d", len(refs))
	}
	a, err := pointRef(sk, refs[0])
	if err != nil {
		return nil, true, err
	}
	b, err := pointRef(sk, refs[1])
	if err != nil {
		return nil, true, err
	}
	about, err := lineRef(sk, refs[2])
	if err != nil {
		return nil, true, err
	}
	return sk.GeometricConstraints().AddSymmetry(a, b, about), true, nil
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
