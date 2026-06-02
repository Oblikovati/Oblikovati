// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/math"
)

// This file serializes a part's USER work features (planes/axes/points) into the
// recipe. The origin coordinate system is regenerated, never serialized — only its
// stable references are recorded. User features are stored in global creation order so
// a feature that references an earlier one restores after it; their definitions name
// the geometry they are built on by WorkRef (origin well-known keys, or earlier user
// features by position), which re-resolve on recompute.

// WorkFeatureData is the recipe form of one user work feature: which collection it
// belongs to, its definition kind, the references it is built on, and any parameter.
type WorkFeatureData struct {
	Collection string    `yaml:"collection"` // plane | axis | point
	Kind       string    `yaml:"kind"`
	Refs       []string  `yaml:"refs,omitempty"`
	Offset     float64   `yaml:"offset,omitempty"`   // plane-offset
	Position   []float64 `yaml:"position,omitempty"` // point position [x,y,z]
}

// MarshalWork projects the user work features into the recipe, in creation order.
func MarshalWork(g *WorkGeometry) ([]WorkFeatureData, error) {
	out := make([]WorkFeatureData, 0, len(g.userSeq))
	for i, e := range g.userSeq {
		d, err := serializeWorkFeature(g, e)
		if err != nil {
			return nil, fmt.Errorf("work feature %d (%s): %w", i, e.collection, err)
		}
		out = append(out, d)
	}
	return out, nil
}

func serializeWorkFeature(g *WorkGeometry, e userEntry) (WorkFeatureData, error) {
	switch e.collection {
	case "plane":
		return serializePlaneDef(g.planes.Item(e.index).def)
	case "axis":
		return serializeAxisDef(g.axes.Item(e.index).def)
	case "point":
		return serializePointDef(g.points.Item(e.index).def)
	default:
		return WorkFeatureData{}, fmt.Errorf("unknown work collection %q", e.collection)
	}
}

func serializePlaneDef(def planeDefinition) (WorkFeatureData, error) {
	d := WorkFeatureData{Collection: "plane", Kind: def.kindName(), Refs: refStrings(def.refs())}
	switch v := def.(type) {
	case offsetPlaneDef:
		d.Offset = v.offset()
	case threePointPlaneDef:
		// references only
	default:
		return WorkFeatureData{}, fmt.Errorf("no codec for work plane definition %q", def.kindName())
	}
	return d, nil
}

func serializeAxisDef(def axisDefinition) (WorkFeatureData, error) {
	switch def.(type) {
	case twoPointsAxisDef, planeIntersectionAxisDef:
		return WorkFeatureData{Collection: "axis", Kind: def.kindName(), Refs: refStrings(def.refs())}, nil
	default:
		return WorkFeatureData{}, fmt.Errorf("no codec for work axis definition %q", def.kindName())
	}
}

func serializePointDef(def pointDefinition) (WorkFeatureData, error) {
	d := WorkFeatureData{Collection: "point", Kind: def.kindName(), Refs: refStrings(def.refs())}
	switch v := def.(type) {
	case positionPointDef:
		p := v.at()
		d.Position = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
	case planeAxisPointDef:
		// references only
	default:
		return WorkFeatureData{}, fmt.Errorf("no codec for work point definition %q", def.kindName())
	}
	return d, nil
}

// ApplyWork rebuilds the user work features onto g (which already holds the origin
// frame), in order, resolving each one's references as it goes.
func ApplyWork(g *WorkGeometry, data []WorkFeatureData) error {
	for i, d := range data {
		if err := restoreWorkFeature(g, d); err != nil {
			return fmt.Errorf("work feature %d (%s/%s): %w", i, d.Collection, d.Kind, err)
		}
	}
	return nil
}

func restoreWorkFeature(g *WorkGeometry, d WorkFeatureData) error {
	switch d.Collection + "/" + d.Kind {
	case "plane/plane-offset":
		base, err := workRefAt(d.Refs, 0)
		if err != nil {
			return err
		}
		off := d.Offset
		g.WorkPlanes().AddByPlaneAndOffset(base, func() float64 { return off })
	case "plane/three-points":
		r, err := workRefs(d.Refs, 3)
		if err != nil {
			return err
		}
		g.WorkPlanes().AddByThreePoints(r[0], r[1], r[2])
	case "axis/two-points":
		r, err := workRefs(d.Refs, 2)
		if err != nil {
			return err
		}
		g.WorkAxes().AddByTwoPoints(r[0], r[1])
	case "axis/plane-intersection":
		r, err := workRefs(d.Refs, 2)
		if err != nil {
			return err
		}
		g.WorkAxes().AddByPlaneIntersection(r[0], r[1])
	case "point/position":
		if len(d.Position) != 3 {
			return fmt.Errorf("position point needs 3 coordinates, got %d", len(d.Position))
		}
		pos := math.P3(d.Position[0], d.Position[1], d.Position[2])
		g.WorkPoints().AddByPosition(func() math.Point3 { return pos })
	case "point/plane-axis-intersection":
		r, err := workRefs(d.Refs, 2)
		if err != nil {
			return err
		}
		g.WorkPoints().AddByPlaneAndAxisIntersection(r[0], r[1])
	default:
		return fmt.Errorf("no restore codec for work feature %s/%s", d.Collection, d.Kind)
	}
	return nil
}

// RevolveData is a revolve's recipe: the sketch profile, the revolution axis (a
// WorkRef, typically an origin axis or a user work axis), the swept angle, and the
// boolean operation. Generation is still deferred, but the definition round-trips.
type RevolveData struct {
	Sketch    int     `yaml:"sketch"`
	Profile   int     `yaml:"profile"`
	Axis      string  `yaml:"axis"`
	Angle     float64 `yaml:"angle,omitempty"` // 0 ⇒ full revolution
	Operation string  `yaml:"operation"`
}

func serializeRevolve(def *RevolveDefinition, sk SketchIndexer) (*RevolveData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("revolve references a sketch that is not in the part")
	}
	if def.Axis == nil {
		return nil, fmt.Errorf("revolve has no axis")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	return &RevolveData{
		Sketch:    idx,
		Profile:   def.ProfileIndex,
		Axis:      string(def.Axis.Key()),
		Angle:     evalFloat(def.Angle),
		Operation: op,
	}, nil
}

func restoreRevolve(fs *PartFeatures, d *RevolveData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("revolve feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("revolve references sketch index %d, which does not exist", d.Sketch)
	}
	if work == nil {
		return nil, fmt.Errorf("revolve needs the part's work geometry to resolve its axis")
	}
	axis, err := work.axis(WorkRef(d.Axis))
	if err != nil {
		return nil, fmt.Errorf("revolve axis: %w", err)
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	angle := d.Angle
	return NewRevolveFeatures(fs).Add(skt, d.Profile, axis, func() float64 { return angle }, op), nil
}

// refStrings renders work references as their string form for YAML.
func refStrings(refs []WorkRef) []string {
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = string(r)
	}
	return out
}

// workRefs requires exactly n references; workRefAt fetches one by position.
func workRefs(refs []string, n int) ([]WorkRef, error) {
	if len(refs) != n {
		return nil, fmt.Errorf("expected %d references, got %d", n, len(refs))
	}
	out := make([]WorkRef, n)
	for i, r := range refs {
		out[i] = WorkRef(r)
	}
	return out, nil
}

func workRefAt(refs []string, i int) (WorkRef, error) {
	if i >= len(refs) {
		return "", fmt.Errorf("missing reference %d (have %d)", i, len(refs))
	}
	return WorkRef(refs[i]), nil
}
