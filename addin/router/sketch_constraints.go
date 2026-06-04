// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// addConstraint applies a geometric constraint of the requested kind to the referenced
// geometry and reports the sketch's resulting DOF.
func addConstraint(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.AddConstraintArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := activeSketchAt(s, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	if _, err := buildGeometricConstraint(sk, types.GeometricConstraintKind(in.Kind), in.Entities); err != nil {
		return nil, err
	}
	g := sk.GeometricConstraints()
	return json.Marshal(wire.AddConstraintResult{Index: g.Count() - 1, Kind: in.Kind, DOF: sk.DegreesOfFreedom()})
}

// deleteConstraint removes a geometric constraint by index.
func deleteConstraint(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.DeleteConstraintArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := activeSketchAt(s, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	g := sk.GeometricConstraints()
	if in.ConstraintIndex < 0 || in.ConstraintIndex >= g.Count() {
		return nil, fmt.Errorf("sketch.deleteConstraint: index %d out of range (%d constraints)", in.ConstraintIndex, g.Count())
	}
	g.Delete(g.Item(in.ConstraintIndex))
	return json.Marshal(wire.OKResult{OK: true})
}

// buildGeometricConstraint resolves the references and applies the matching model
// constraint. The kinds are split into small groups to keep each switch readable.
func buildGeometricConstraint(sk *sketch.Sketch, kind types.GeometricConstraintKind, refs []uint64) (sketch.Constraint, error) {
	if c, ok, err := pointPairConstraint(sk, kind, refs); ok {
		return c, err
	}
	if c, ok, err := lineConstraint(sk, kind, refs); ok {
		return c, err
	}
	if c, ok, err := circularConstraint(sk, kind, refs); ok {
		return c, err
	}
	if c, ok, err := mixedConstraint(sk, kind, refs); ok {
		return c, err
	}
	if c, ok, err := newConstraint212(sk, kind, refs); ok {
		return c, err
	}
	return nil, fmt.Errorf("sketch.addConstraint: unsupported kind %q", kind)
}

// newConstraint212 handles the M21 additions (ground/offset/pattern link) and the
// horizontal/vertical *align* aliases. Each freezes the current geometry where it needs a
// value, so no extra argument is required.
func newConstraint212(sk *sketch.Sketch, kind types.GeometricConstraintKind, refs []uint64) (sketch.Constraint, bool, error) {
	g := sk.GeometricConstraints()
	switch kind {
	case types.GeoConstraintGround:
		return groundConstraint(sk, refs)
	case types.GeoConstraintOffset:
		return offsetConstraintFromRefs(sk, refs)
	case types.GeoConstraintPattern:
		return withTwoPoints(sk, refs, func(a, b *sketch.Point) sketch.Constraint { return g.AddPatternLink(a, b) })
	default:
		return nil, false, nil
	}
}

// groundConstraint grounds a single referenced entity (fixes all its points).
func groundConstraint(sk *sketch.Sketch, refs []uint64) (sketch.Constraint, bool, error) {
	if len(refs) != 1 {
		return nil, true, fmt.Errorf("sketch.addConstraint: ground needs 1 entity ref, got %d", len(refs))
	}
	e, ok := sk.EntityByID(sketch.ID(refs[0]))
	if !ok {
		return nil, true, fmt.Errorf("sketch.addConstraint: no entity with id %d", refs[0])
	}
	return sk.GeometricConstraints().AddGround(e), true, nil
}

// offsetConstraintFromRefs holds two lines parallel at their current perpendicular
// distance (frozen from the present geometry).
func offsetConstraintFromRefs(sk *sketch.Sketch, refs []uint64) (sketch.Constraint, bool, error) {
	l1, l2, ok, err := twoLinesOf(sk, refs)
	if !ok {
		return nil, true, err
	}
	return sk.GeometricConstraints().AddOffset(l1, l2, currentPerpDistance(l1, l2)), true, nil
}

// pointPairConstraint handles the constraints between two points.
func pointPairConstraint(sk *sketch.Sketch, kind types.GeometricConstraintKind, refs []uint64) (sketch.Constraint, bool, error) {
	g := sk.GeometricConstraints()
	switch kind {
	case types.GeoConstraintCoincident:
		return withTwoPoints(sk, refs, func(a, b *sketch.Point) sketch.Constraint { return g.AddCoincident(a, b) })
	case types.GeoConstraintHorizontal:
		return withTwoPoints(sk, refs, func(a, b *sketch.Point) sketch.Constraint { return g.AddHorizontal(a, b) })
	case types.GeoConstraintVertical:
		return withTwoPoints(sk, refs, func(a, b *sketch.Point) sketch.Constraint { return g.AddVertical(a, b) })
	default:
		return nil, false, nil
	}
}

// lineConstraint handles the constraints between two lines.
func lineConstraint(sk *sketch.Sketch, kind types.GeometricConstraintKind, refs []uint64) (sketch.Constraint, bool, error) {
	g := sk.GeometricConstraints()
	switch kind {
	case types.GeoConstraintParallel:
		return withTwoLines(sk, refs, func(a, b *sketch.Line) sketch.Constraint { return g.AddParallel(a, b) })
	case types.GeoConstraintPerpendicular:
		return withTwoLines(sk, refs, func(a, b *sketch.Line) sketch.Constraint { return g.AddPerpendicular(a, b) })
	case types.GeoConstraintCollinear:
		return withTwoLines(sk, refs, func(a, b *sketch.Line) sketch.Constraint { return g.AddCollinear(a, b) })
	case types.GeoConstraintEqualLength:
		return withTwoLines(sk, refs, func(a, b *sketch.Line) sketch.Constraint { return g.AddEqualLength(a, b) })
	default:
		return nil, false, nil
	}
}

// circularConstraint handles the constraints between two circular curves.
func circularConstraint(sk *sketch.Sketch, kind types.GeometricConstraintKind, refs []uint64) (sketch.Constraint, bool, error) {
	g := sk.GeometricConstraints()
	switch kind {
	case types.GeoConstraintConcentric:
		return withTwoCirculars(sk, refs, func(a, b sketch.CircularCurve) sketch.Constraint { return g.AddConcentric(a, b) })
	case types.GeoConstraintEqualRadius:
		return withTwoCirculars(sk, refs, func(a, b sketch.CircularCurve) sketch.Constraint { return g.AddEqualRadius(a, b) })
	default:
		return nil, false, nil
	}
}

// mixedConstraint handles the constraints whose operands are of mixed types.
func mixedConstraint(sk *sketch.Sketch, kind types.GeometricConstraintKind, refs []uint64) (sketch.Constraint, bool, error) {
	g := sk.GeometricConstraints()
	switch kind {
	case types.GeoConstraintPointOnLine:
		return pointLineConstraint(sk, refs, func(p *sketch.Point, l *sketch.Line) sketch.Constraint { return g.AddPointOnLine(p, l) })
	case types.GeoConstraintMidpoint:
		return pointLineConstraint(sk, refs, func(p *sketch.Point, l *sketch.Line) sketch.Constraint { return g.AddMidpoint(p, l) })
	case types.GeoConstraintPointOnCircle:
		return pointCircleConstraint(sk, refs)
	case types.GeoConstraintTangent:
		return tangentConstraint(sk, refs)
	case types.GeoConstraintFix:
		return fixConstraint(sk, refs)
	default:
		return nil, false, nil
	}
}
