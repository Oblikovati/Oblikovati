// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// addDimension adds a dimensional constraint of the requested kind and reports the
// backing parameter, the measured value, and the sketch's resulting DOF.
func addDimension(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.AddDimensionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := activeSketchAt(s, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	dim, err := buildDimension(sk, types.DimensionConstraintKind(in.Kind), in.Entities, in.Expression)
	if err != nil {
		return nil, err
	}
	dc := sk.DimensionConstraints()
	return json.Marshal(wire.AddDimensionResult{
		Index: dc.Count() - 1, Kind: in.Kind, Parameter: dim.Parameter().Name(),
		Value: dim.Measured(), DOF: sk.DegreesOfFreedom(),
	})
}

// driveDimension edits a dimension: its value (expression), driven flag, and/or limits.
func driveDimension(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.DriveDimensionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	dim, err := dimensionAt(part, in.SketchIndex, in.DimensionIndex)
	if err != nil {
		return nil, err
	}
	return applyDimensionEdit(part, dim, in)
}

// applyDimensionEdit applies the optional value/driven/limits edits to a dimension.
func applyDimensionEdit(part *compdef.PartComponentDefinition, dim *sketch.DimensionConstraint, in wire.DriveDimensionArgs) (json.RawMessage, error) {
	if in.SetLimits {
		dim.SetLimits(in.Min, in.Max)
	}
	if in.SetDriven {
		dim.SetDriven(in.Driven)
	}
	if in.Expression != "" {
		v, err := part.Units().Parse(in.Expression, dimensionUnit(dim.Kind()))
		if err != nil {
			return nil, fmt.Errorf("sketch.driveDimension: value %q: %w", in.Expression, err)
		}
		if err := dim.Drive(v.Value); err != nil {
			return nil, fmt.Errorf("sketch.driveDimension: %w", err)
		}
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// buildDimension resolves references and applies the matching model dimension factory.
func buildDimension(sk *sketch.Sketch, kind types.DimensionConstraintKind, refs []uint64, expr string) (*sketch.DimensionConstraint, error) {
	dc := sk.DimensionConstraints()
	switch kind {
	case types.DimConstraintDistance:
		a, b, err := twoPointRefs(sk, refs)
		if err != nil {
			return nil, err
		}
		return dc.AddDistance(a, b, expr)
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
		return buildAdvancedDimension(sk, kind, refs, expr)
	}
}

// buildAdvancedDimension handles the M21 dimension kinds (offset/three-point-angle/
// ellipse-radius); split out of buildDimension to keep that switch small.
func buildAdvancedDimension(sk *sketch.Sketch, kind types.DimensionConstraintKind, refs []uint64, expr string) (*sketch.DimensionConstraint, error) {
	dc := sk.DimensionConstraints()
	switch kind {
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
