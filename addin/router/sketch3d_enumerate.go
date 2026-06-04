// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// enumerateEntities3D lists a 3D sketch's geometry (kind, construction, points, radius).
func enumerateEntities3D(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	ents := sk.Entities()
	out := make([]wire.Sketch3DEntityInfo, 0, len(ents))
	for i, e := range ents {
		out = append(out, entity3DInfo(i, e))
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

// entity3DInfo renders one 3D entity as its wire summary (kind, defining points in
// model cm, radius for circular kinds, construction flag).
func entity3DInfo(index int, e sketch.Entity) wire.Sketch3DEntityInfo {
	info := wire.Sketch3DEntityInfo{Index: index, ID: uint64(e.EntityID()), Kind: string(types.Sketch3DEntityUnknown)}
	if p, ok := e.(*sketch.Point3D); ok {
		info.Kind = string(types.Sketch3DEntityPoint)
		info.Points = [][]float64{p3coords(p.Position())}
		return info
	}
	if fillSegmentCurve3DInfo(&info, e) {
		return info
	}
	fillRoundCurve3DInfo(&info, e)
	if info.Kind == string(types.Sketch3DEntityUnknown) {
		fillDerivedCurve3DInfo(&info, e)
	}
	return info
}

// fillDerivedCurve3DInfo renders the surface-derived curves (F11) and included reference
// geometry (F08) by kind; their geometry is recompute-derived, so only identity is shown.
func fillDerivedCurve3DInfo(info *wire.Sketch3DEntityInfo, e sketch.Entity) {
	switch v := e.(type) {
	case *sketch.IntersectionCurve3D:
		info.Kind, info.Construction = string(types.Sketch3DEntityIntersection), v.IsConstruction()
	case *sketch.SilhouetteCurve3D:
		info.Kind, info.Construction = string(types.Sketch3DEntitySilhouette), v.IsConstruction()
	case *sketch.ProjectToSurfaceCurve3D:
		info.Kind, info.Construction = string(types.Sketch3DEntityProjectToSurface), v.IsConstruction()
	case *sketch.OnFaceCurve3D:
		info.Kind, info.Construction = string(types.Sketch3DEntityOnFace), v.IsConstruction()
	case *sketch.OffsetCurve3:
		info.Kind, info.Construction = string(types.Sketch3DEntityOffset), v.IsConstruction()
	case *sketch.IncludedPoint3D:
		info.Kind, info.Construction = string(types.Sketch3DEntityIncludedPoint), true
		info.Points = [][]float64{p3coords(v.Position())}
	case *sketch.IncludedCurve3D:
		info.Kind, info.Construction = string(types.Sketch3DEntityIncludedCurve), true
		info.Points = point3sCoords(v.Points())
	}
}

// fillSegmentCurve3DInfo renders the straight/poly curve families (line/arc) and the
// plain circle into info, reporting whether it matched.
func fillSegmentCurve3DInfo(info *wire.Sketch3DEntityInfo, e sketch.Entity) bool {
	switch v := e.(type) {
	case *sketch.Line3D:
		info.Kind = string(types.Sketch3DEntityLine)
		info.Points = [][]float64{p3coords(v.A.Position()), p3coords(v.B.Position())}
		info.Construction = v.IsConstruction()
	case *sketch.Circle3D:
		info.Kind = string(types.Sketch3DEntityCircle)
		info.Points = [][]float64{p3coords(v.Center.Position())}
		info.Radius = float64(v.Radius)
		info.Construction = v.IsConstruction()
	case *sketch.Arc3D:
		info.Kind = string(types.Sketch3DEntityArc)
		info.Points = [][]float64{p3coords(v.Center.Position()), p3coords(v.Start.Position()), p3coords(v.End.Position())}
		info.Radius = float64(v.Radius())
		info.Construction = v.IsConstruction()
	default:
		return false
	}
	return true
}

// fillRoundCurve3DInfo renders the helix and conic families (centered, radius-bearing).
func fillRoundCurve3DInfo(info *wire.Sketch3DEntityInfo, e sketch.Entity) {
	switch v := e.(type) {
	case *sketch.HelicalCurve3D:
		info.Kind = string(types.Sketch3DEntityHelical)
		info.Points = [][]float64{p3coords(v.Origin.Position())}
		info.Radius = float64(v.StartRadius)
		info.Construction = v.IsConstruction()
	case *sketch.Ellipse3D:
		info.Kind = string(types.Sketch3DEntityEllipse)
		info.Points = [][]float64{p3coords(v.Center.Position())}
		info.Radius = float64(v.MajorRadius)
		info.Construction = v.IsConstruction()
	case *sketch.EllipticalArc3D:
		info.Kind = string(types.Sketch3DEntityEllipticalArc)
		info.Points = [][]float64{p3coords(v.Center.Position())}
		info.Radius = float64(v.MajorRadius)
		info.Construction = v.IsConstruction()
	default:
		fillSplineCurve3DInfo(info, e)
	}
}

// fillSplineCurve3DInfo renders the spline family (interpolation/control/fixed spline +
// equation curve) into info.
func fillSplineCurve3DInfo(info *wire.Sketch3DEntityInfo, e sketch.Entity) {
	switch v := e.(type) {
	case *sketch.Spline3D:
		if v.IsFitType() {
			info.Kind = string(types.Sketch3DEntitySpline)
		} else {
			info.Kind = string(types.Sketch3DEntityControlPointSpline)
		}
		info.Points = point3sCoords(v.Sample())
		info.Construction = v.IsConstruction()
	case *sketch.FixedSpline3D:
		info.Kind = string(types.Sketch3DEntityFixedSpline)
		info.Points = point3sCoords(v.Sample())
		info.Construction = v.IsConstruction()
	case *sketch.EquationCurve3D:
		info.Kind = string(types.Sketch3DEntityEquationCurve)
		info.Points = point3sCoords(v.Sample(16))
		info.Construction = v.IsConstruction()
	}
}

// point3sCoords flattens a polyline to [[x,y,z],…] for enumeration.
func point3sCoords(pts []math.Point3) [][]float64 {
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
