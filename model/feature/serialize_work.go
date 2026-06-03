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
	Angle      float64   `yaml:"angle,omitempty"`    // line-plane-angle
	Position   []float64 `yaml:"position,omitempty"` // point position / fixed-frame origin [x,y,z]
	XAxis      []float64 `yaml:"xaxis,omitempty"`    // fixed-frame X axis [x,y,z]
	YAxis      []float64 `yaml:"yaxis,omitempty"`    // fixed-frame Y axis [x,y,z]
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
	case fixedFramePlaneDef:
		p := v.origin()
		d.Position = []float64{float64(p.X), float64(p.Y), float64(p.Z)}
		d.XAxis, d.YAxis = unitSlice(v.x), unitSlice(v.y)
	case linePlaneAnglePlaneDef:
		d.Angle = v.angle()
	case threePointPlaneDef, planeAndPointPlaneDef, twoPlanesPlaneDef, twoLinesPlaneDef,
		normalToCurvePlaneDef, torusMidPlaneDef, pointAndTangentPlaneDef,
		planeAndTangentPlaneDef, lineAndTangentPlaneDef:
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
	switch d.Collection {
	case "plane":
		return restorePlaneFeature(g.WorkPlanes(), d)
	case "axis":
		return restoreAxisFeature(g.WorkAxes(), d)
	case "point":
		return restorePointFeature(g.WorkPoints(), d)
	default:
		return fmt.Errorf("unknown work collection %q", d.Collection)
	}
}

// restorePlaneFeature rebuilds one user work plane from its recipe. Reference-only kinds
// resolve through workRefs; fixed-frame/offset/angle kinds also carry scalar parameters,
// re-installed as closures so a recompute re-reads them.
func restorePlaneFeature(c *WorkPlanes, d WorkFeatureData) error {
	switch d.Kind {
	case "plane-offset":
		return restoreRefPlane(d, 1, func(r []WorkRef) {
			off := d.Offset
			c.AddByPlaneAndOffset(r[0], func() float64 { return off })
		})
	case "three-points":
		return restoreRefPlane(d, 3, func(r []WorkRef) { c.AddByThreePoints(r[0], r[1], r[2]) })
	case "fixed-frame":
		return restoreFixedFrame(c, d)
	case "plane-point":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByPlaneAndPoint(r[0], r[1]) })
	case "two-planes":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByTwoPlanes(r[0], r[1]) })
	case "line-plane-angle":
		return restoreRefPlane(d, 2, func(r []WorkRef) {
			ang := d.Angle
			c.AddByLinePlaneAndAngle(r[0], r[1], func() float64 { return ang })
		})
	case "two-lines":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByTwoLines(r[0], r[1]) })
	case "normal-to-curve":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByNormalToCurve(r[0], r[1]) })
	case "torus-midplane":
		return restoreRefPlane(d, 1, func(r []WorkRef) { c.AddByTorusMidPlane(r[0]) })
	case "point-tangent":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByPointAndTangent(r[0], r[1]) })
	case "plane-tangent":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByPlaneAndTangent(r[0], r[1]) })
	case "line-tangent":
		return restoreRefPlane(d, 2, func(r []WorkRef) { c.AddByLineAndTangent(r[0], r[1]) })
	default:
		return fmt.Errorf("no restore codec for work plane kind %q", d.Kind)
	}
}

// restoreRefPlane resolves d's n references and calls add with them, centralizing the
// arity check so each plane kind above stays a single line.
func restoreRefPlane(d WorkFeatureData, n int, add func([]WorkRef)) error {
	r, err := workRefs(d.Refs, n)
	if err != nil {
		return err
	}
	add(r)
	return nil
}

// restoreFixedFrame rebuilds an AddFixed plane from its origin and two in-plane axes.
func restoreFixedFrame(c *WorkPlanes, d WorkFeatureData) error {
	origin, err := point3From(d.Position, "fixed-frame origin")
	if err != nil {
		return err
	}
	x, err := unit3From(d.XAxis, "fixed-frame X axis")
	if err != nil {
		return err
	}
	y, err := unit3From(d.YAxis, "fixed-frame Y axis")
	if err != nil {
		return err
	}
	c.AddFixed(func() math.Point3 { return origin }, x, y)
	return nil
}

func restoreAxisFeature(c *WorkAxes, d WorkFeatureData) error {
	r, err := workRefs(d.Refs, 2)
	if err != nil {
		return err
	}
	switch d.Kind {
	case "two-points":
		c.AddByTwoPoints(r[0], r[1])
	case "plane-intersection":
		c.AddByPlaneIntersection(r[0], r[1])
	default:
		return fmt.Errorf("no restore codec for work axis kind %q", d.Kind)
	}
	return nil
}

func restorePointFeature(c *WorkPoints, d WorkFeatureData) error {
	switch d.Kind {
	case "position":
		pos, err := point3From(d.Position, "position point")
		if err != nil {
			return err
		}
		c.AddByPosition(func() math.Point3 { return pos })
		return nil
	case "plane-axis-intersection":
		r, err := workRefs(d.Refs, 2)
		if err != nil {
			return err
		}
		c.AddByPlaneAndAxisIntersection(r[0], r[1])
		return nil
	default:
		return fmt.Errorf("no restore codec for work point kind %q", d.Kind)
	}
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

// CoilData is a coil's recipe: the sketch profile, the helix axis (a WorkRef), the
// pitch (per revolution), the number of revolutions, the taper, and the operation.
type CoilData struct {
	Sketch      int     `yaml:"sketch"`
	Profile     int     `yaml:"profile"`
	Axis        string  `yaml:"axis"`
	Pitch       float64 `yaml:"pitch"`
	Revolutions float64 `yaml:"revolutions"`
	Taper       float64 `yaml:"taper,omitempty"`
	Operation   string  `yaml:"operation"`
}

func serializeCoil(def *CoilDefinition, sk SketchIndexer) (*CoilData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("coil references a sketch that is not in the part")
	}
	if def.Axis == nil {
		return nil, fmt.Errorf("coil has no axis")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	return &CoilData{
		Sketch: idx, Profile: def.ProfileIndex, Axis: string(def.Axis.Key()),
		Pitch: evalFloat(def.Pitch), Revolutions: evalFloat(def.Revolutions),
		Taper: def.Taper, Operation: op,
	}, nil
}

func restoreCoil(fs *PartFeatures, d *CoilData, sk SketchIndexer, work *WorkGeometry) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("coil feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("coil references sketch index %d, which does not exist", d.Sketch)
	}
	if work == nil {
		return nil, fmt.Errorf("coil needs the part's work geometry to resolve its axis")
	}
	axis, err := work.axis(WorkRef(d.Axis))
	if err != nil {
		return nil, fmt.Errorf("coil axis: %w", err)
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	pitch, revs := d.Pitch, d.Revolutions
	return NewCoilFeatures(fs).Add(skt, d.Profile, axis,
		func() float64 { return pitch }, func() float64 { return revs }, d.Taper, op), nil
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

// workRefs requires exactly n references, converting them from their string form.
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

// unitSlice renders a unit vector as its [x,y,z] components for YAML.
func unitSlice(u math.UnitVector3) []float64 {
	v := u.AsVector()
	return []float64{float64(v.X), float64(v.Y), float64(v.Z)}
}

// point3From reads a 3-component coordinate slice into a point, naming what for errors.
func point3From(s []float64, what string) (math.Point3, error) {
	if len(s) != 3 {
		return math.Point3{}, fmt.Errorf("%s needs 3 coordinates, got %d", what, len(s))
	}
	return math.P3(s[0], s[1], s[2]), nil
}

// unit3From reads a 3-component slice into a unit vector (erroring on a zero vector).
func unit3From(s []float64, what string) (math.UnitVector3, error) {
	if len(s) != 3 {
		return math.UnitVector3{}, fmt.Errorf("%s needs 3 components, got %d", what, len(s))
	}
	return math.NewUnitVector3(s[0], s[1], s[2])
}
