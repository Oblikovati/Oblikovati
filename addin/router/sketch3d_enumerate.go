// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// enumerateEntities3D lists a 3D sketch's geometry (kind, construction, points, radius).
func enumerateEntities3D(s *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.EnumerateEntities3DResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.EnumerateEntities3DResult{}, err
	}
	ents := sk.Entities()
	moveable := sk.MoveableClassifier()
	guid, _ := activeDocumentGUID(s) // empty ⇒ omit keys; never fails for a real document
	out := make([]wire.Sketch3DEntityInfo, 0, len(ents))
	for i, e := range ents {
		info := entity3DInfo(i, e)
		info.ReferenceKey = entityReferenceKey(guid, e)
		info.MoveableStatus = moveable.Of(e).String()
		out = append(out, info)
	}
	return wire.EnumerateEntities3DResult{Entities: out}, nil
}

// enumerateConstraints3D lists a 3D sketch's geometric constraints.
func enumerateConstraints3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.ListConstraints3DResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ListConstraints3DResult{}, err
	}
	cons := sk.GeometricConstraints3D().All()
	out := make([]wire.Constraint3DInfo, 0, len(cons))
	for i, c := range cons {
		kind, ids := constraint3DKind(c)
		out = append(out, wire.Constraint3DInfo{Index: i, Kind: string(kind), Entities: ids})
	}
	return wire.ListConstraints3DResult{Constraints: out}, nil
}

// enumerateDimensions3D lists a 3D sketch's dimensional constraints.
func enumerateDimensions3D(_ *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.ListDimensions3DResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ListDimensions3DResult{}, err
	}
	dims := sk.DimensionConstraints3D().All()
	out := make([]wire.Dimension3DInfo, 0, len(dims))
	for i, d := range dims {
		out = append(out, wire.Dimension3DInfo{
			Index:      i,
			Kind:       d.KindName(),
			Name:       d.Parameter().Name(),
			Expression: d.Parameter().Expression(),
			Value:      d.Measured(),
			Driven:     d.Driven(),
		})
	}
	return wire.ListDimensions3DResult{Dimensions: out}, nil
}

// entity3DInfo renders one 3D entity as its wire summary through the
// ShapedEntity3D capability: the model names its own kind, shape points, and
// radius; this adapter only translates vocabularies (#1624, audit I1).
func entity3DInfo(index int, e sketch.Entity) wire.Sketch3DEntityInfo {
	info := wire.Sketch3DEntityInfo{Index: index, ID: uint64(e.EntityID()), Kind: string(types.Sketch3DEntityUnknown)}
	se, ok := e.(sketch.ShapedEntity3D)
	if !ok {
		return info
	}
	info.Kind = string(wireSketch3DEntityKind(se.Kind()))
	info.Points = point3sCoords(se.ShapePoints3D())
	info.Radius = entityRadius(e)
	info.Construction = isConstruction(e)
	return info
}

// wireSketch3DEntityKinds maps the model's entity-kind vocabulary onto the 3D
// wire enum; kinds without a wire spelling enumerate as unknown.
var wireSketch3DEntityKinds = map[sketch.EntityKind]types.Sketch3DEntityKind{
	sketch.PointKind:                 types.Sketch3DEntityPoint,
	sketch.LineKind:                  types.Sketch3DEntityLine,
	sketch.CircleKind:                types.Sketch3DEntityCircle,
	sketch.ArcKind:                   types.Sketch3DEntityArc,
	sketch.EllipseKind:               types.Sketch3DEntityEllipse,
	sketch.EllipticalArcKind:         types.Sketch3DEntityEllipticalArc,
	sketch.SplineKind:                types.Sketch3DEntitySpline,
	sketch.ControlPointSplineKind:    types.Sketch3DEntityControlPointSpline,
	sketch.SplineHandleKind:          types.Sketch3DEntitySplineHandle,
	sketch.FixedSplineKind:           types.Sketch3DEntityFixedSpline,
	sketch.EquationCurveKind:         types.Sketch3DEntityEquationCurve,
	sketch.HelicalKind:               types.Sketch3DEntityHelical,
	sketch.IntersectionCurveKind:     types.Sketch3DEntityIntersection,
	sketch.SilhouetteCurveKind:       types.Sketch3DEntitySilhouette,
	sketch.ProjectToSurfaceCurveKind: types.Sketch3DEntityProjectToSurface,
	sketch.OnFaceCurveKind:           types.Sketch3DEntityOnFace,
	sketch.OffsetCurveKind:           types.Sketch3DEntityOffset,
	sketch.IncludedPointKind:         types.Sketch3DEntityIncludedPoint,
	sketch.IncludedCurveKind:         types.Sketch3DEntityIncludedCurve,
}

func wireSketch3DEntityKind(k sketch.EntityKind) types.Sketch3DEntityKind {
	if w, ok := wireSketch3DEntityKinds[k]; ok {
		return w
	}
	return types.Sketch3DEntityUnknown
}

// point3sCoords flattens a polyline to [[x,y,z],…] for enumeration (nil in,
// nil out — shapeless kinds keep omitting the field).
func point3sCoords(pts []math.Point3) [][]float64 {
	if len(pts) == 0 {
		return nil
	}
	out := make([][]float64, len(pts))
	for i, p := range pts {
		out[i] = p3coords(p)
	}
	return out
}

// p3coords flattens a model point to [x,y,z].
func p3coords(p math.Point3) []float64 {
	return []float64{float64(p.X), float64(p.Y), float64(p.Z)}
}

// constraint3DKind maps a 3D constraint to its wire kind and the session ids
// it relates, both self-reported by the KindedConstraint capability (#1625).
// The 3D wire enum spells every kind exactly like the model's persisted
// vocabulary (by construction), so the kind maps by conversion; a constraint
// without the capability enumerates honestly as unknown.
func constraint3DKind(c sketch.Constraint) (types.Geometric3DConstraintKind, []uint64) {
	kc, ok := c.(sketch.KindedConstraint)
	if !ok {
		return types.Geo3DUnknown, nil
	}
	return types.Geometric3DConstraintKind(kc.ConstraintKind()), ids(kc.RelatedEntities()...)
}
