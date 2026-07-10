// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// addConstraint applies a geometric constraint of the requested kind to the referenced
// geometry and reports the sketch's resulting DOF.
func addConstraint(_ *app.Session, part *compdef.PartComponentDefinition, in wire.AddConstraintArgs) (wire.AddConstraintResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.AddConstraintResult{}, err
	}
	// Report the kind the CREATED constraint actually enumerates as, not the request kind:
	// a two-point horizontal is created as (and reports as) horizontalAlign, a single line as
	// horizontal (#1871). The custom kind returns no constraint, so it echoes its request kind.
	kind := in.Kind
	if types.GeometricConstraintKind(in.Kind) == types.GeoConstraintCustom {
		if err := addCustomConstraint(sk, in); err != nil {
			return wire.AddConstraintResult{}, err
		}
	} else {
		c, err := buildGeometricConstraint(sk, types.GeometricConstraintKind(in.Kind), in.Entities)
		if err != nil {
			return wire.AddConstraintResult{}, err
		}
		if wk, _ := geometricShape(c); wk != types.GeoConstraintUnknown {
			kind = string(wk)
		}
	}
	g := sk.GeometricConstraints()
	return wire.AddConstraintResult{Index: g.Count() - 1, Kind: kind, DOF: sk.DegreesOfFreedom()}, nil
}

// deleteConstraint removes a geometric constraint by index.
func deleteConstraint(_ *app.Session, part *compdef.PartComponentDefinition, in wire.DeleteConstraintArgs) (wire.OKResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.OKResult{}, err
	}
	g := sk.GeometricConstraints()
	if in.ConstraintIndex < 0 || in.ConstraintIndex >= g.Count() {
		return wire.OKResult{}, fmt.Errorf("sketch.deleteConstraint: index %d out of range (%d constraints)", in.ConstraintIndex, g.Count())
	}
	// System-owned records (the text-box anchor) refuse deletion (M06-F11).
	if err := g.DeleteAllowed(g.Item(in.ConstraintIndex)); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// addCustomConstraint applies the add-in tag constraint: arbitrary entity
// operands, owned by ClientID (M06-F11, #626).
func addCustomConstraint(sk *sketch.Sketch, in wire.AddConstraintArgs) error {
	ents := make([]sketch.Entity, 0, len(in.Entities))
	for _, id := range in.Entities {
		e, ok := sk.EntityByID(sketch.ID(id))
		if !ok {
			return fmt.Errorf("sketch.addConstraint: entity %d not found", id)
		}
		ents = append(ents, e)
	}
	_, err := sk.GeometricConstraints().AddCustom(in.ClientID, in.Name, ents)
	return err
}

// buildGeometricConstraint resolves the references and applies the matching model
// constraint. The kinds are split into small groups to keep each switch readable.
func buildGeometricConstraint(sk *sketch.Sketch, kind types.GeometricConstraintKind, refs []uint64) (sketch.Constraint, error) {
	if c, ok, err := pointPairConstraint(sk, kind, refs); ok {
		return c, err
	}
	if c, ok, err := horizontalVerticalConstraint(sk, kind, refs); ok {
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
	// Accept a top-level entity (line/circle/standalone point) or a sub-point such
	// as a circle's center or a line endpoint — a *Point is itself an Entity, and
	// pinning a single defining point is a common, valid ground. EntityByID only
	// sees top-level entities, so fall back to PointByID.
	e, ok := sk.EntityByID(sketch.ID(refs[0]))
	if !ok {
		if p, pok := sk.PointByID(sketch.ID(refs[0])); pok {
			e = p
		} else {
			return nil, true, fmt.Errorf("sketch.addConstraint: no entity or point with id %d", refs[0])
		}
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
	default:
		return nil, false, nil
	}
}

// horizontalVerticalConstraint routes the horizontal/vertical family (#1871): the plain
// horizontal/vertical make a single line horizontal/vertical (one line ref) or level two
// points (the align form, two point refs); horizontalAlign/verticalAlign are the explicit
// two-point align kinds.
func horizontalVerticalConstraint(sk *sketch.Sketch, kind types.GeometricConstraintKind, refs []uint64) (sketch.Constraint, bool, error) {
	g := sk.GeometricConstraints()
	switch kind {
	case types.GeoConstraintHorizontal:
		return singleLineOrAlign(sk, refs,
			func(l *sketch.Line) sketch.Constraint { return g.AddLineHorizontal(l) },
			func(a, b *sketch.Point) sketch.Constraint { return g.AddHorizontal(a, b) })
	case types.GeoConstraintVertical:
		return singleLineOrAlign(sk, refs,
			func(l *sketch.Line) sketch.Constraint { return g.AddLineVertical(l) },
			func(a, b *sketch.Point) sketch.Constraint { return g.AddVertical(a, b) })
	case types.GeoConstraintHorizontalAlign:
		return withTwoPoints(sk, refs, func(a, b *sketch.Point) sketch.Constraint { return g.AddHorizontal(a, b) })
	case types.GeoConstraintVerticalAlign:
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
		return midpointConstraint(sk, refs)
	case types.GeoConstraintPointOnCircle:
		return pointCircleConstraint(sk, refs)
	case types.GeoConstraintTangent:
		return tangentConstraint(sk, refs)
	case types.GeoConstraintSymmetry:
		return symmetryConstraint(sk, refs)
	case types.GeoConstraintSmooth:
		return smoothConstraintFromRefs(sk, refs)
	case types.GeoConstraintFix:
		return fixConstraint(sk, refs)
	default:
		return nil, false, nil
	}
}
