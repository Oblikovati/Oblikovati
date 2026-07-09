// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// addDimension adds a dimensional constraint of the requested kind and reports the
// backing parameter, the measured value, and the sketch's resulting DOF.
func addDimension(_ *app.Session, part *compdef.PartComponentDefinition, in wire.AddDimensionArgs) (wire.AddDimensionResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.AddDimensionResult{}, err
	}
	orientation, ok := sketch.ParseDistanceOrientation(in.Orientation)
	if !ok {
		return wire.AddDimensionResult{}, fmt.Errorf("sketch.addDimension: unknown orientation %q (want aligned|horizontal|vertical)", in.Orientation)
	}
	dim, err := buildDimension(sk, types.DimensionConstraintKind(in.Kind), in.Entities, in.Expression, in.FarSide, orientation)
	if err != nil {
		return wire.AddDimensionResult{}, err
	}
	dc := sk.DimensionConstraints()
	return wire.AddDimensionResult{
		Index: dc.Count() - 1, Kind: in.Kind, Parameter: dim.Parameter().Name(),
		Value: dim.Measured(), DOF: sk.DegreesOfFreedom(),
	}, nil
}

// driveDimension edits a dimension: its value (expression), driven flag, and/or limits.
func driveDimension(_ *app.Session, part *compdef.PartComponentDefinition, in wire.DriveDimensionArgs) (wire.OKResult, error) {
	dim, err := dimensionAt(part, in.SketchIndex, in.DimensionIndex)
	if err != nil {
		return wire.OKResult{}, err
	}
	return applyDimensionEdit(part, dim, in)
}

// applyDimensionEdit applies the optional value/driven/limits edits to a dimension.
func applyDimensionEdit(part *compdef.PartComponentDefinition, dim *sketch.DimensionConstraint, in wire.DriveDimensionArgs) (wire.OKResult, error) {
	if in.SetLimits {
		dim.SetLimits(in.Min, in.Max)
	}
	if in.SetDriven {
		dim.SetDriven(in.Driven)
	}
	if in.Expression != "" {
		v, err := resolveQuantity(part, in.Expression, dimensionUnit(dim.Kind()))
		if err != nil {
			return wire.OKResult{}, fmt.Errorf("sketch.driveDimension: value %q: %w", in.Expression, err)
		}
		if err := dim.Drive(v.Value); err != nil {
			return wire.OKResult{}, fmt.Errorf("sketch.driveDimension: %w", err)
		}
	}
	return wire.OKResult{OK: true}, nil
}

// buildDimension resolves references and applies the matching model dimension factory. orientation
// applies only to the distance kind (aligned/horizontal/vertical); other kinds ignore it.
func buildDimension(sk *sketch.Sketch, kind types.DimensionConstraintKind, refs []uint64, expr string, farSide bool, orientation sketch.DistanceOrientation) (*sketch.DimensionConstraint, error) {
	dc := sk.DimensionConstraints()
	switch kind {
	case types.DimConstraintDistance:
		a, b, err := twoPointRefs(sk, refs)
		if err != nil {
			return nil, err
		}
		return dc.AddDistanceOriented(a, b, expr, orientation)
	case types.DimConstraintAngle:
		a, b, err := twoLineRefs(sk, refs)
		if err != nil {
			return nil, err
		}
		return dc.AddAngle(a, b, expr)
	case types.DimConstraintRadius:
		return radiusDimension(sk, dc.AddRadius, refs, expr)
	case types.DimConstraintDiameter:
		return radiusDimension(sk, dc.AddDiameter, refs, expr)
	case types.DimConstraintArcLength:
		return arcLengthDimension(sk, refs, expr)
	default:
		return buildAdvancedDimension(sk, kind, refs, expr, farSide)
	}
}

// buildAdvancedDimension handles the M21 dimension kinds (offset/three-point-angle/
// ellipse-radius) plus the tangent-distance dimension (#152); split out of buildDimension to
// keep that switch small.
func buildAdvancedDimension(sk *sketch.Sketch, kind types.DimensionConstraintKind, refs []uint64, expr string, farSide bool) (*sketch.DimensionConstraint, error) {
	dc := sk.DimensionConstraints()
	switch kind {
	case types.DimConstraintTangentDistance:
		return tangentDistanceDimension(sk, refs, expr, farSide)
	case types.DimConstraintOffset:
		return offsetDimension(sk, refs, expr)
	case types.DimConstraintThreePointAngle:
		v, a, b, err := threePointRefs(sk, refs)
		if err != nil {
			return nil, err
		}
		return dc.AddThreePointAngle(v, a, b, expr)
	case types.DimConstraintEllipseRadius:
		e, err := ellipseRef(sk, refs)
		if err != nil {
			return nil, err
		}
		return dc.AddEllipseRadius(e, expr)
	default:
		return nil, fmt.Errorf("sketch.addDimension: unsupported kind %q", kind)
	}
}

// tangentDistanceDimension resolves a line + circle/arc ref and dimensions the distance from
// the line to the curve's near (default) or far tangent point (#152).
func tangentDistanceDimension(sk *sketch.Sketch, refs []uint64, expr string, farSide bool) (*sketch.DimensionConstraint, error) {
	if len(refs) != 2 {
		return nil, fmt.Errorf("sketch.addDimension: tangentDistance needs a line ref and a circle/arc ref, got %d", len(refs))
	}
	l, err := lineRef(sk, refs[0])
	if err != nil {
		return nil, err
	}
	c, err := circularRef(sk, refs[1])
	if err != nil {
		return nil, err
	}
	return sk.DimensionConstraints().AddTangentDistance(l, c, farSide, expr)
}

// offsetDimension resolves a point + line ref and dimensions their perpendicular distance.
func offsetDimension(sk *sketch.Sketch, refs []uint64, expr string) (*sketch.DimensionConstraint, error) {
	if len(refs) != 2 {
		return nil, fmt.Errorf("sketch.addDimension: offsetDim needs a point + line ref, got %d", len(refs))
	}
	p, err := pointRef(sk, refs[0])
	if err != nil {
		return nil, err
	}
	l, err := lineRef(sk, refs[1])
	if err != nil {
		return nil, err
	}
	return sk.DimensionConstraints().AddOffsetDim(p, l, expr)
}

// threePointRefs resolves three point refs (vertex, a, b).
func threePointRefs(sk *sketch.Sketch, refs []uint64) (*sketch.Point, *sketch.Point, *sketch.Point, error) {
	if len(refs) != 3 {
		return nil, nil, nil, fmt.Errorf("sketch.addDimension: threePointAngle needs 3 point refs, got %d", len(refs))
	}
	v, err := pointRef(sk, refs[0])
	if err != nil {
		return nil, nil, nil, err
	}
	a, err := pointRef(sk, refs[1])
	if err != nil {
		return nil, nil, nil, err
	}
	b, err := pointRef(sk, refs[2])
	if err != nil {
		return nil, nil, nil, err
	}
	return v, a, b, nil
}

// ellipseRef resolves a single ref to an *Ellipse.
func ellipseRef(sk *sketch.Sketch, refs []uint64) (*sketch.Ellipse, error) {
	if len(refs) != 1 {
		return nil, fmt.Errorf("sketch.addDimension: ellipseRadius needs 1 ellipse ref, got %d", len(refs))
	}
	e, ok := sk.EntityByID(sketch.ID(refs[0]))
	if !ok {
		return nil, fmt.Errorf("sketch.addDimension: no entity with id %d", refs[0])
	}
	el, ok := e.(*sketch.Ellipse)
	if !ok {
		return nil, fmt.Errorf("sketch.addDimension: entity %d is %T, want an ellipse", refs[0], e)
	}
	return el, nil
}

// radiusDimension resolves a single circle-or-arc ref and applies a radius/diameter
// factory. Inventor dimensions the radius/diameter of arcs as well as circles.
func radiusDimension(sk *sketch.Sketch, add func(sketch.CircularCurve, string) (*sketch.DimensionConstraint, error), refs []uint64, expr string) (*sketch.DimensionConstraint, error) {
	if len(refs) != 1 {
		return nil, fmt.Errorf("sketch.addDimension: radius/diameter needs 1 circle or arc ref, got %d", len(refs))
	}
	c, err := circularRef(sk, refs[0])
	if err != nil {
		return nil, err
	}
	return add(c, expr)
}

// arcLengthDimension resolves a single arc ref and applies the arc-length factory.
func arcLengthDimension(sk *sketch.Sketch, refs []uint64, expr string) (*sketch.DimensionConstraint, error) {
	if len(refs) != 1 {
		return nil, fmt.Errorf("sketch.addDimension: arcLength needs 1 arc ref, got %d", len(refs))
	}
	e, ok := sk.EntityByID(sketch.ID(refs[0]))
	if !ok {
		return nil, fmt.Errorf("sketch.addDimension: no entity with id %d", refs[0])
	}
	a, ok := e.(*sketch.Arc)
	if !ok {
		return nil, fmt.Errorf("sketch.addDimension: entity %d is %T, want an arc", refs[0], e)
	}
	return sk.DimensionConstraints().AddArcLength(a, expr)
}

// dimensionAt returns the dimensional constraint at (sketchIndex, dimIndex).
func dimensionAt(part *compdef.PartComponentDefinition, sketchIndex, dimIndex int) (*sketch.DimensionConstraint, error) {
	sk, err := sketchAtIndex(part, sketchIndex)
	if err != nil {
		return nil, err
	}
	dc := sk.DimensionConstraints()
	if dimIndex < 0 || dimIndex >= dc.Count() {
		return nil, fmt.Errorf("sketch.driveDimension: index %d out of range (%d dimensions)", dimIndex, dc.Count())
	}
	return dc.Item(dimIndex), nil
}

// dimensionUnit returns the unit a dimension's value expression is parsed in.
func dimensionUnit(kind sketch.DimKind) param.Unit {
	if kind == sketch.AngleDim {
		return param.Angle
	}
	return param.Length
}
