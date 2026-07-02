// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// enumerateEntities3D lists a 3D sketch's geometry (kind, construction, points, radius).
func enumerateEntities3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
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
	return json.Marshal(wire.EnumerateEntities3DResult{Entities: out})
}

// enumerateConstraints3D lists a 3D sketch's geometric constraints.
func enumerateConstraints3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	cons := sk.GeometricConstraints3D().All()
	out := make([]wire.Constraint3DInfo, 0, len(cons))
	for i, c := range cons {
		kind, ids := constraint3DKind(c)
		out = append(out, wire.Constraint3DInfo{Index: i, Kind: string(kind), Entities: ids})
	}
	return json.Marshal(wire.ListConstraints3DResult{Constraints: out})
}

// enumerateDimensions3D lists a 3D sketch's dimensional constraints.
func enumerateDimensions3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
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
	return json.Marshal(wire.ListDimensions3DResult{Dimensions: out})
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

// constraint3DKind maps a 3D constraint to its wire kind and the session ids it relates.
func constraint3DKind(c sketch.Constraint) (types.Geometric3DConstraintKind, []uint64) {
	switch v := c.(type) {
	case *sketch.Coincident3D:
		return types.Geo3DCoincident, []uint64{uint64(v.A.EntityID()), uint64(v.B.EntityID())}
	case *sketch.Collinear3D:
		return types.Geo3DCollinear, []uint64{uint64(v.A.EntityID()), uint64(v.B.EntityID()), uint64(v.C.EntityID())}
	case *sketch.Concentric3D:
		return types.Geo3DConcentric, []uint64{uint64(v.Center1.EntityID()), uint64(v.Center2.EntityID())}
	case *sketch.Equal3D:
		return types.Geo3DEqual, nil
	default:
		return lineConstraint3DKind(c)
	}
}

// lineConstraint3DKind maps the line/point-operand constraints (M22-F05) to their kind.
func lineConstraint3DKind(c sketch.Constraint) (types.Geometric3DConstraintKind, []uint64) {
	switch v := c.(type) {
	case *sketch.Parallel3D:
		return types.Geo3DParallel, []uint64{uint64(v.L1.EntityID()), uint64(v.L2.EntityID())}
	case *sketch.Perpendicular3D:
		return types.Geo3DPerpendicular, []uint64{uint64(v.L1.EntityID()), uint64(v.L2.EntityID())}
	case *sketch.Midpoint3D:
		return types.Geo3DMidpoint, []uint64{uint64(v.P.EntityID()), uint64(v.L.EntityID())}
	case *sketch.Ground3D:
		return types.Geo3DGround, []uint64{uint64(v.P.EntityID())}
	case *sketch.ParallelToAxis3D:
		return axisConstraintKind(v), []uint64{uint64(v.L.EntityID())}
	case *sketch.ParallelToPlane3D:
		return planeConstraintKind(v), []uint64{uint64(v.L.EntityID())}
	default:
		return curveConstraint3DKind(c)
	}
}

// curveConstraint3DKind maps the curve-join constraints (issue #142) to their kind.
func curveConstraint3DKind(c sketch.Constraint) (types.Geometric3DConstraintKind, []uint64) {
	switch v := c.(type) {
	case *sketch.Tangent3D:
		return types.Geo3DTangent, []uint64{uint64(v.C1.EntityID()), uint64(v.C2.EntityID())}
	case *sketch.Smooth3D:
		return types.Geo3DSmooth, []uint64{uint64(v.C1.EntityID()), uint64(v.C2.EntityID())}
	case *sketch.SplineFitPoints3D:
		return types.Geo3DSplineFitPoints, []uint64{uint64(v.Spline.EntityID()), uint64(v.P.EntityID())}
	case *sketch.Helical3D:
		return types.Geo3DHelical, []uint64{uint64(v.H.EntityID()), uint64(v.C.EntityID())}
	case *sketch.Bend3D:
		return types.Geo3DBend, []uint64{uint64(v.Arc.EntityID()), uint64(v.L1.EntityID()), uint64(v.L2.EntityID())}
	default:
		return types.Geo3DUnknown, nil
	}
}

// axisConstraintKind names a parallel-to-axis constraint by which world axis it pins to.
func axisConstraintKind(c *sketch.ParallelToAxis3D) types.Geometric3DConstraintKind {
	switch {
	case c.Axis.X != 0:
		return types.Geo3DParallelToXAxis
	case c.Axis.Y != 0:
		return types.Geo3DParallelToYAxis
	default:
		return types.Geo3DParallelToZAxis
	}
}

// planeConstraintKind names a parallel-to-plane constraint by its plane normal.
func planeConstraintKind(c *sketch.ParallelToPlane3D) types.Geometric3DConstraintKind {
	switch {
	case c.Normal.Z != 0:
		return types.Geo3DParallelToXYPlane
	case c.Normal.Y != 0:
		return types.Geo3DParallelToXZPlane
	default:
		return types.Geo3DParallelToYZPlane
	}
}
