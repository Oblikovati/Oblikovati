// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati/addin/modelaccess"
	"oblikovati/api/types"
	"oblikovati/api/wire"
	"oblikovati/app"
	"oblikovati/math"
	"oblikovati/model/compdef"
	"oblikovati/model/param"
	"oblikovati/model/sketch"
)

// addSketch3DDimension is the discriminated 3D dimensional-constraint constructor.
func addSketch3DDimension(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.AddSketch3DDimensionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := activeSketch3DAt(s, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	dim, err := buildSketch3DDimension(sk, types.Dimension3DConstraintKind(in.Kind), in)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.AddSketch3DDimensionResult{
		Index: sk.DimensionConstraints3D().Count() - 1, Kind: in.Kind,
		Parameter: dim.Parameter().Name(), Value: dim.Measured(), DOF: sk.DegreesOfFreedom(),
	})
}

// buildSketch3DDimension resolves operands and applies the matching dimension factory.
func buildSketch3DDimension(sk *sketch.Sketch3D, kind types.Dimension3DConstraintKind, in wire.AddSketch3DDimensionArgs) (*sketch.DimensionConstraint3D, error) {
	switch kind {
	case types.Dim3DDistance:
		p, err := points3D(sk, in.Entities, 2)
		if err != nil {
			return nil, err
		}
		return sk.DimensionConstraints3D().AddDistance(p[0], p[1], in.Expression)
	case types.Dim3DPointPlaneDistance:
		return pointPlaneDimension3D(sk, in)
	default:
		return lineDimension3D(sk, kind, in)
	}
}

// lineDimension3D builds the line/circle dimensions (lineLength/radius/twoLineAngle).
func lineDimension3D(sk *sketch.Sketch3D, kind types.Dimension3DConstraintKind, in wire.AddSketch3DDimensionArgs) (*sketch.DimensionConstraint3D, error) {
	dc := sk.DimensionConstraints3D()
	switch kind {
	case types.Dim3DLineLength:
		l, err := lines3D(sk, in.Entities, 1)
		if err != nil {
			return nil, err
		}
		return dc.AddLineLength(l[0], in.Expression)
	case types.Dim3DRadius:
		c, err := circle3DRef(sk, in.Entities)
		if err != nil {
			return nil, err
		}
		return dc.AddRadius(c, in.Expression)
	case types.Dim3DTwoLineAngle:
		l, err := lines3D(sk, in.Entities, 2)
		if err != nil {
			return nil, err
		}
		return dc.AddTwoLineAngle(l[0], l[1], in.Expression)
	default:
		return nil, fmt.Errorf("sketch3d.addDimension: unsupported kind %q", in.Kind)
	}
}

// pointPlaneDimension3D builds a point-to-origin-plane distance dimension.
func pointPlaneDimension3D(sk *sketch.Sketch3D, in wire.AddSketch3DDimensionArgs) (*sketch.DimensionConstraint3D, error) {
	p, err := points3D(sk, in.Entities, 1)
	if err != nil {
		return nil, err
	}
	normal, err := planeNormal3D(in.Plane)
	if err != nil {
		return nil, err
	}
	return sk.DimensionConstraints3D().AddPointPlaneDistance(p[0], normal, in.Expression)
}

// planeNormal3D maps an origin-plane label to its unit normal.
func planeNormal3D(plane string) (math.Vector3, error) {
	switch plane {
	case "", "XY":
		return math.V3(0, 0, 1), nil
	case "XZ":
		return math.V3(0, 1, 0), nil
	case "YZ":
		return math.V3(1, 0, 0), nil
	default:
		return math.Vector3{}, fmt.Errorf("sketch3d.addDimension: unknown plane %q (want XY|XZ|YZ)", plane)
	}
}

// circle3DRef resolves a single ref to a 3D circle.
func circle3DRef(sk *sketch.Sketch3D, refs []uint64) (*sketch.Circle3D, error) {
	if len(refs) != 1 {
		return nil, fmt.Errorf("sketch3d.addDimension: radius needs 1 circle ref, got %d", len(refs))
	}
	e, ok := sk.EntityByID(sketch.ID(refs[0]))
	if !ok {
		return nil, fmt.Errorf("sketch3d: no entity with id %d", refs[0])
	}
	c, ok := e.(*sketch.Circle3D)
	if !ok {
		return nil, fmt.Errorf("sketch3d: entity %d is %T, want a 3D circle", refs[0], e)
	}
	return c, nil
}

// driveSketch3DDimension edits a 3D dimension's value and/or driving state.
func driveSketch3DDimension(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.DriveSketch3DDimensionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	dc := sk.DimensionConstraints3D()
	if in.DimensionIndex < 0 || in.DimensionIndex >= dc.Count() {
		return nil, fmt.Errorf("sketch3d.driveDimension: index %d out of range (%d dimensions)", in.DimensionIndex, dc.Count())
	}
	if err := applySketch3DDimensionEdit(part, dc.Item(in.DimensionIndex), in); err != nil {
		return nil, err
	}
	return json.Marshal(wire.OKResult{OK: true})
}

// applySketch3DDimensionEdit applies the driven toggle and/or value to a 3D dimension.
func applySketch3DDimensionEdit(part *compdef.PartComponentDefinition, dim *sketch.DimensionConstraint3D, in wire.DriveSketch3DDimensionArgs) error {
	if in.SetDriven {
		dim.SetDriven(in.Driven)
	}
	if in.Expression == "" {
		return nil
	}
	unit := param.Length
	if dim.KindName() == "twoLineAngle" {
		unit = param.Angle
	}
	v, err := part.Units().Parse(in.Expression, unit)
	if err != nil {
		return fmt.Errorf("sketch3d.driveDimension: value %q: %w", in.Expression, err)
	}
	return dim.Drive(float64(v.Value))
}
