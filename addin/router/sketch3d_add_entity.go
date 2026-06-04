// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/param"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// addSketch3DEntity is the discriminated 3D entity constructor (point/line/circle/arc).
func addSketch3DEntity(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddSketch3DEntityArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	return buildSketch3DEntity(part, sk, in)
}

// buildSketch3DEntity resolves the requested kind and applies the matching model factory.
func buildSketch3DEntity(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DEntityArgs) (json.RawMessage, error) {
	switch types.Sketch3DEntityKind(in.Kind) {
	case types.Sketch3DEntityPoint:
		p, err := point3At(in.Points, 0)
		if err != nil {
			return nil, err
		}
		pt := sk.AddPoint3D(p)
		return entityResult(uint64(pt.EntityID()), in.Kind, pt.EntityID())
	case types.Sketch3DEntityLine:
		return buildLine3D(sk, in)
	case types.Sketch3DEntityCircle:
		return buildCircle3D(part, sk, in)
	case types.Sketch3DEntityArc:
		return buildArc3D(sk, in)
	default:
		return nil, fmt.Errorf("sketch3d.addEntity: unsupported kind %q", in.Kind)
	}
}

func buildLine3D(sk *sketch.Sketch3D, in wire.AddSketch3DEntityArgs) (json.RawMessage, error) {
	a, b, err := twoPoint3(in.Points)
	if err != nil {
		return nil, err
	}
	l := sk.AddLine3D(a, b)
	l.SetConstruction(in.Construction)
	return entityResult(uint64(l.EntityID()), in.Kind, l.A.EntityID(), l.B.EntityID())
}

func buildCircle3D(part *compdef.PartComponentDefinition, sk *sketch.Sketch3D, in wire.AddSketch3DEntityArgs) (json.RawMessage, error) {
	center, err := point3At(in.Points, 0)
	if err != nil {
		return nil, err
	}
	axis, err := axisOrZ(in.Axis)
	if err != nil {
		return nil, err
	}
	r, err := part.Units().Parse(in.Radius, param.Length)
	if err != nil {
		return nil, fmt.Errorf("sketch3d.addEntity: circle radius %q: %w", in.Radius, err)
	}
	c := sk.AddCircle3D(center, axis, float64(r.Value))
	c.SetConstruction(in.Construction)
	return entityResult(uint64(c.EntityID()), in.Kind, c.Center.EntityID())
}

func buildArc3D(sk *sketch.Sketch3D, in wire.AddSketch3DEntityArgs) (json.RawMessage, error) {
	if len(in.Points) != 3 {
		return nil, fmt.Errorf("sketch3d.addEntity: arc needs 3 points (center,start,end), got %d", len(in.Points))
	}
	center, _ := point3At(in.Points, 0)
	start, _ := point3At(in.Points, 1)
	end, _ := point3At(in.Points, 2)
	a := sk.AddArc3D(center, start, end, in.CCW)
	a.SetConstruction(in.Construction)
	return entityResult(uint64(a.EntityID()), in.Kind, a.Center.EntityID(), a.Start.EntityID(), a.End.EntityID())
}

// entityResult marshals the created entity's id, kind, and defining point ids.
func entityResult(id uint64, kind string, pointIDs ...sketch.ID) (json.RawMessage, error) {
	ids := make([]uint64, len(pointIDs))
	for i, p := range pointIDs {
		ids[i] = uint64(p)
	}
	return json.Marshal(wire.AddSketch3DEntityResult{EntityID: id, Kind: kind, PointIDs: ids})
}

// twoPoint3 resolves exactly two [x,y,z] points.
func twoPoint3(points [][]float64) (math.Point3, math.Point3, error) {
	if len(points) != 2 {
		return math.Point3{}, math.Point3{}, fmt.Errorf("sketch3d.addEntity: need 2 points, got %d", len(points))
	}
	a, err := point3At(points, 0)
	if err != nil {
		return math.Point3{}, math.Point3{}, err
	}
	b, err := point3At(points, 1)
	if err != nil {
		return math.Point3{}, math.Point3{}, err
	}
	return a, b, nil
}

// point3At reads the i-th [x,y,z] coordinate triple as a model point.
func point3At(points [][]float64, i int) (math.Point3, error) {
	if i >= len(points) || len(points[i]) != 3 {
		return math.Point3{}, fmt.Errorf("sketch3d.addEntity: point %d must be [x,y,z]", i)
	}
	p := points[i]
	return math.P3(math.Scalar(p[0]), math.Scalar(p[1]), math.Scalar(p[2])), nil
}

// axisOrZ reads an axis [x,y,z], defaulting to +Z when empty.
func axisOrZ(axis []float64) (math.UnitVector3, error) {
	if len(axis) == 0 {
		return math.NewUnitVector3(0, 0, 1)
	}
	if len(axis) != 3 {
		return math.UnitVector3{}, fmt.Errorf("sketch3d.addEntity: axis must be [x,y,z], got %d components", len(axis))
	}
	return math.NewUnitVector3(math.Scalar(axis[0]), math.Scalar(axis[1]), math.Scalar(axis[2]))
}
