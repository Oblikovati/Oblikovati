// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/sketch"
)

// enumerateEntities lists a sketch's geometry: kind, construction flag, points, radius.
func enumerateEntities(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	ents := sk.Entities()
	moveable := sk.MoveableClassifier()
	guid, _ := activeDocumentGUID(s) // empty ⇒ omit keys; never fails for a real document
	out := make([]wire.SketchEntityInfo, len(ents))
	for i, e := range ents {
		kind, pts, radius := entityShape(e)
		out[i] = wire.SketchEntityInfo{
			Index:          i,
			ID:             uint64(e.EntityID()),
			Kind:           string(kind),
			Construction:   isConstruction(e),
			Points:         pts,
			Radius:         radius,
			ReferenceKey:   entityReferenceKey(guid, e),
			MoveableStatus: moveable.Of(e).String(),
			FitMethod:      splineFitSpelling(e),
		}
	}
	return json.Marshal(wire.EnumerateEntitiesResult{Entities: out})
}

// entityReferenceKey derives an entity's persistent reference key (#153); empty when the
// document guid is unavailable, so the field is simply omitted.
func entityReferenceKey(guid string, e sketch.Entity) string {
	if guid == "" {
		return ""
	}
	k, err := identity.SketchEntityKey(guid, uint64(e.EntityID()))
	if err != nil {
		return ""
	}
	return k
}

// enumerateConstraints lists a sketch's geometric constraints (kind + related entity ids).
func enumerateConstraints(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	cons := sk.GeometricConstraints()
	out := make([]wire.ConstraintInfo, cons.Count())
	for i := 0; i < cons.Count(); i++ {
		c := cons.Item(i)
		kind, refs := geometricShape(c)
		out[i] = wire.ConstraintInfo{
			Index: i, Kind: string(kind), Entities: refs,
			Deletable: constraintDeletable(c),
		}
		if custom, ok := c.(*sketch.CustomConstraint); ok {
			out[i].ClientID, out[i].Name = custom.ClientID, custom.Name
		}
	}
	return json.Marshal(wire.ListConstraintsResult{Constraints: out})
}

// enumerateDimensions lists a sketch's dimensional constraints.
func enumerateDimensions(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	dims := sk.DimensionConstraints()
	out := make([]wire.DimensionInfo, dims.Count())
	for i := 0; i < dims.Count(); i++ {
		out[i] = dimensionInfo(i, dims.Item(i))
	}
	return json.Marshal(wire.ListDimensionsResult{Dimensions: out})
}

// dimensionInfo renders one dimensional constraint as its wire summary.
func dimensionInfo(index int, d *sketch.DimensionConstraint) wire.DimensionInfo {
	info := wire.DimensionInfo{
		Index:  index,
		Kind:   string(dimensionKind(d.Kind())),
		Value:  d.Measured(),
		Driven: d.Driven(),
	}
	if p := d.Parameter(); p != nil {
		info.Name = p.Name()
		info.Expression = p.Expression()
	}
	return info
}

// entityShape returns an entity's wire kind, defining points ([x,y] each), and radius.
// Split into geometric curves and annotative entities to keep each switch small.
func entityShape(e sketch.Entity) (types.SketchEntityKind, [][]float64, float64) {
	switch v := e.(type) {
	case *sketch.Point:
		return types.SketchEntityPoint, [][]float64{pt(v)}, 0
	case *sketch.Line:
		return types.SketchEntityLine, [][]float64{pt(v.A), pt(v.B)}, 0
	case *sketch.Circle:
		return types.SketchEntityCircle, [][]float64{pt(v.Center)}, float64(v.Radius)
	case *sketch.Arc:
		return types.SketchEntityArc, [][]float64{pt(v.Center), pt(v.Start), pt(v.End)}, float64(v.Radius())
	case *sketch.Ellipse:
		return types.SketchEntityEllipse, [][]float64{pt(v.Center)}, 0
	case *sketch.EllipticalArc:
		return types.SketchEntityEllipticalArc, [][]float64{pt(v.Center)}, 0
	case *sketch.Spline:
		return splineKind(v), splinePts(v), 0
	case *sketch.SplineHandle:
		return types.SketchEntitySplineHandle, [][]float64{pt(v.Anchor), pt(v.End)}, 0
	default:
		return annotationShape(e)
	}
}

// splineKind reports a spline's wire kind: a fit (interpolating) spline is "spline", a
// control-point (approximating) spline is "controlPointSpline" (#150).
func splineKind(s *sketch.Spline) types.SketchEntityKind {
	if s.IsFitType() {
		return types.SketchEntitySpline
	}
	return types.SketchEntityControlPointSpline
}

// splineFitSpelling reports a fit spline's fit-method wire spelling (the
// zero value resolves to the smooth default); empty for everything else.
func splineFitSpelling(e sketch.Entity) string {
	sp, ok := e.(*sketch.Spline)
	if !ok || !sp.IsFitType() {
		return ""
	}
	if sp.FitMethod == 0 {
		return types.SplineFitSmooth.String()
	}
	return sp.FitMethod.String()
}

// constraintDeletable reports whether explicit deletion is allowed (false for
// system-owned records like the text-box anchor — M06-F11, #626).
func constraintDeletable(c sketch.Constraint) bool {
	nd, ok := c.(sketch.NonDeletable)
	return !ok || nd.Deletable()
}

// annotationShape handles the non-curve (image/fill/text) entities.
func annotationShape(e sketch.Entity) (types.SketchEntityKind, [][]float64, float64) {
	switch v := e.(type) {
	case *sketch.SketchImage:
		return types.SketchEntityImage, [][]float64{{float64(v.Anchor.X), float64(v.Anchor.Y)}}, 0
	case *sketch.FillRegion:
		return types.SketchEntityFillRegion, [][]float64{{float64(v.Seed.X), float64(v.Seed.Y)}}, 0
	case *sketch.TextBox:
		return types.SketchEntityText, [][]float64{{float64(v.Anchor.X), float64(v.Anchor.Y)}}, 0
	case *sketch.EquationCurve:
		return types.SketchEntityEquationCurve, nil, 0
	case *sketch.FixedSpline:
		return types.SketchEntityFixedSpline, point2Slice(v.Pts), 0
	case *sketch.OffsetSpline:
		return types.SketchEntityOffsetSpline, nil, 0
	case *sketch.ProjectedPoint:
		return types.SketchEntityProjectedPoint, [][]float64{{float64(v.Position().X), float64(v.Position().Y)}}, 0
	case *sketch.ProjectedCurve:
		return types.SketchEntityProjectedCurve, point2Slice(v.Points()), 0
	default:
		return types.SketchEntityUnknown, nil, 0
	}
}

// point2Slice renders model points as [x,y] pairs for the wire DTOs.
func point2Slice(pts []math.Point2) [][]float64 {
	out := make([][]float64, len(pts))
	for i, p := range pts {
		out[i] = []float64{float64(p.X), float64(p.Y)}
	}
	return out
}

// geometricShape returns a geometric constraint's wire kind and the ids of the entities
// (points/lines/curves) it relates. Split into point- and curve-anchored groups to keep
// each type switch small.
func geometricShape(c sketch.Constraint) (types.GeometricConstraintKind, []uint64) {
	if k, refs, ok := pointConstraintShape(c); ok {
		return k, refs
	}
	if k, refs, ok := lineConstraintShape(c); ok {
		return k, refs
	}
	if k, refs, ok := circularConstraintShape(c); ok {
		return k, refs
	}
	if k, refs, ok := extraConstraintShape(c); ok {
		return k, refs
	}
	return types.GeoConstraintUnknown, nil
}

// extraConstraintShape handles the M21 constraints (ground/offset/pattern
// link) and the M06-F11 tag constraints (text-box anchor, custom).
func extraConstraintShape(c sketch.Constraint) (types.GeometricConstraintKind, []uint64, bool) {
	switch v := c.(type) {
	case *sketch.GroundConstraint:
		return types.GeoConstraintGround, ids(pointsAsEntities(v.Points())...), true
	case *sketch.OffsetConstraint:
		return types.GeoConstraintOffset, ids(v.L1, v.L2), true
	case *sketch.PatternConstraint:
		return types.GeoConstraintPattern, ids(v.Seed, v.Member), true
	case *sketch.TextBoxAnchorConstraint:
		return types.GeoConstraintTextBox, ids(v.Text), true
	case *sketch.CustomConstraint:
		return types.GeoConstraintCustom, ids(v.Entities...), true
	default:
		return "", nil, false
	}
}

// pointsAsEntities adapts a slice of points to the Entity interface for ids().
func pointsAsEntities(pts []*sketch.Point) []sketch.Entity {
	out := make([]sketch.Entity, len(pts))
	for i, p := range pts {
		out[i] = p
	}
	return out
}

// pointConstraintShape handles the constraints anchored on points (and point↔line).
func pointConstraintShape(c sketch.Constraint) (types.GeometricConstraintKind, []uint64, bool) {
	switch v := c.(type) {
	case *sketch.CoincidentConstraint:
		return types.GeoConstraintCoincident, ids(v.A, v.B), true
	case *sketch.PointOnLineConstraint:
		return types.GeoConstraintPointOnLine, ids(v.P, v.L), true
	case *sketch.MidpointConstraint:
		return types.GeoConstraintMidpoint, ids(v.P, v.L), true
	case *sketch.PointOnCircleConstraint:
		return types.GeoConstraintPointOnCircle, append(ids(v.P), curveID(v.C)...), true
	case *sketch.HorizontalConstraint:
		return types.GeoConstraintHorizontal, ids(v.A, v.B), true
	case *sketch.VerticalConstraint:
		return types.GeoConstraintVertical, ids(v.A, v.B), true
	case *sketch.SymmetryConstraint:
		return types.GeoConstraintSymmetry, ids(v.A, v.B, v.About), true
	case *sketch.FixConstraint:
		return types.GeoConstraintFix, ids(v.P), true
	default:
		return "", nil, false
	}
}

// lineConstraintShape handles the constraints anchored on straight lines.
func lineConstraintShape(c sketch.Constraint) (types.GeometricConstraintKind, []uint64, bool) {
	switch v := c.(type) {
	case *sketch.ParallelConstraint:
		return types.GeoConstraintParallel, ids(v.L1, v.L2), true
	case *sketch.PerpendicularConstraint:
		return types.GeoConstraintPerpendicular, ids(v.L1, v.L2), true
	case *sketch.CollinearConstraint:
		return types.GeoConstraintCollinear, ids(v.L1, v.L2), true
	case *sketch.EqualLengthConstraint:
		return types.GeoConstraintEqualLength, ids(v.L1, v.L2), true
	case *sketch.TangentConstraint:
		return types.GeoConstraintTangent, append(ids(v.L), curveID(v.C)...), true
	default:
		return "", nil, false
	}
}

// circularConstraintShape handles the constraints anchored on circular/smooth curves.
func circularConstraintShape(c sketch.Constraint) (types.GeometricConstraintKind, []uint64, bool) {
	switch v := c.(type) {
	case *sketch.ConcentricConstraint:
		return types.GeoConstraintConcentric, append(curveID(v.C1), curveID(v.C2)...), true
	case *sketch.EqualRadiusConstraint:
		return types.GeoConstraintEqualRadius, append(curveID(v.C1), curveID(v.C2)...), true
	case *sketch.CircularTangentConstraint:
		return types.GeoConstraintTangent, append(curveID(v.C1), curveID(v.C2)...), true
	case *sketch.SmoothConstraint:
		return types.GeoConstraintSmooth, ids(v.C1, v.C2), true
	default:
		return "", nil, false
	}
}

// dimensionKind maps a model DimKind to its wire kind.
func dimensionKind(k sketch.DimKind) types.DimensionConstraintKind {
	switch k {
	case sketch.DistanceDim:
		return types.DimConstraintDistance
	case sketch.AngleDim:
		return types.DimConstraintAngle
	case sketch.RadiusDim:
		return types.DimConstraintRadius
	case sketch.DiameterDim:
		return types.DimConstraintDiameter
	case sketch.ArcLengthDim:
		return types.DimConstraintArcLength
	default:
		return advancedDimensionKind(k)
	}
}

// advancedDimensionKind maps the M21+ dimension kinds (offset/three-point-angle/ellipse-radius/
// tangent-distance); split from dimensionKind to keep each switch small.
func advancedDimensionKind(k sketch.DimKind) types.DimensionConstraintKind {
	switch k {
	case sketch.OffsetDim:
		return types.DimConstraintOffset
	case sketch.ThreePointAngleDim:
		return types.DimConstraintThreePointAngle
	case sketch.EllipseRadiusDim:
		return types.DimConstraintEllipseRadius
	case sketch.TangentDistanceDim:
		return types.DimConstraintTangentDistance
	default:
		return types.DimConstraintUnknown
	}
}

// isConstruction reports whether a curve entity is construction geometry.
func isConstruction(e sketch.Entity) bool {
	c, ok := e.(interface{ IsConstruction() bool })
	return ok && c.IsConstruction()
}

// ids collects the entity ids of the given entities (nil-safe).
func ids(es ...sketch.Entity) []uint64 {
	out := make([]uint64, 0, len(es))
	for _, e := range es {
		if e != nil {
			out = append(out, uint64(e.EntityID()))
		}
	}
	return out
}

// curveID returns the id of a circular curve (a *Circle or *Arc, both Entity), or nil.
func curveID(c sketch.CircularCurve) []uint64 {
	if e, ok := c.(sketch.Entity); ok {
		return []uint64{uint64(e.EntityID())}
	}
	return nil
}

// pt renders a sketch point as [x,y].
func pt(p *sketch.Point) []float64 { return []float64{float64(p.X), float64(p.Y)} }

// splinePts renders a spline's defining points.
func splinePts(s *sketch.Spline) [][]float64 {
	out := make([][]float64, len(s.Points))
	for i, p := range s.Points {
		out[i] = pt(p)
	}
	return out
}
