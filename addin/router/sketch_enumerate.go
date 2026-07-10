// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/sketch"
)

// enumerateEntities lists a sketch's geometry: kind, construction flag, points, radius.
func enumerateEntities(s *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.EnumerateEntitiesResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.EnumerateEntitiesResult{}, err
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
	return wire.EnumerateEntitiesResult{Entities: out}, nil
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
func enumerateConstraints(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.ListConstraintsResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ListConstraintsResult{}, err
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
	return wire.ListConstraintsResult{Constraints: out}, nil
}

// enumerateDimensions lists a sketch's dimensional constraints.
func enumerateDimensions(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.ListDimensionsResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ListDimensionsResult{}, err
	}
	return wire.ListDimensionsResult{Dimensions: projectAll(sk.DimensionConstraints(), dimensionInfo)}, nil
}

// dimensionInfo renders one dimensional constraint as its wire summary.
func dimensionInfo(index int, d *sketch.DimensionConstraint) wire.DimensionInfo {
	info := wire.DimensionInfo{
		Index:          index,
		Kind:           string(dimensionKind(d.Kind())),
		Value:          d.Measured(),
		Driven:         d.Driven(),
		Orientation:    sketch.DistanceOrientationName(d.Orientation()),
		LinearDiameter: d.LinearDiameter(),
	}
	if p := d.Parameter(); p != nil {
		info.Name = p.Name()
		info.Expression = p.Expression()
	}
	if tp, ok := d.TextPoint(); ok {
		info.TextPoint = []float64{float64(tp.X), float64(tp.Y)}
	}
	return info
}

// entityShape returns an entity's wire kind, defining points ([x,y] each), and
// radius, through the ShapedEntity capability — the model names its own kind
// and shape, so this adapter only translates vocabularies (#1624, audit I1).
func entityShape(e sketch.Entity) (types.SketchEntityKind, [][]float64, float64) {
	se, ok := e.(sketch.ShapedEntity)
	if !ok {
		return types.SketchEntityUnknown, nil, 0
	}
	return wireSketchEntityKind(se.Kind()), point2Slice(se.ShapePoints()), entityRadius(e)
}

// wireSketchEntityKinds maps the model's entity-kind vocabulary onto the wire
// enum. Kinds without a wire spelling (block instances) enumerate as unknown,
// as they always have.
var wireSketchEntityKinds = map[sketch.EntityKind]types.SketchEntityKind{
	sketch.PointKind:              types.SketchEntityPoint,
	sketch.LineKind:               types.SketchEntityLine,
	sketch.CircleKind:             types.SketchEntityCircle,
	sketch.ArcKind:                types.SketchEntityArc,
	sketch.EllipseKind:            types.SketchEntityEllipse,
	sketch.EllipticalArcKind:      types.SketchEntityEllipticalArc,
	sketch.SplineKind:             types.SketchEntitySpline,
	sketch.ControlPointSplineKind: types.SketchEntityControlPointSpline,
	sketch.SplineHandleKind:       types.SketchEntitySplineHandle,
	sketch.ImageKind:              types.SketchEntityImage,
	sketch.FillRegionKind:         types.SketchEntityFillRegion,
	sketch.TextKind:               types.SketchEntityText,
	sketch.EquationCurveKind:      types.SketchEntityEquationCurve,
	sketch.FixedSplineKind:        types.SketchEntityFixedSpline,
	sketch.OffsetSplineKind:       types.SketchEntityOffsetSpline,
	sketch.ProjectedPointKind:     types.SketchEntityProjectedPoint,
	sketch.ProjectedCurveKind:     types.SketchEntityProjectedCurve,
}

func wireSketchEntityKind(k sketch.EntityKind) types.SketchEntityKind {
	if w, ok := wireSketchEntityKinds[k]; ok {
		return w
	}
	return types.SketchEntityUnknown
}

// entityRadius reports the optional radius capability (0 for radius-free kinds).
func entityRadius(e sketch.Entity) float64 {
	if r, ok := e.(sketch.RadiusedEntity); ok {
		return r.ShapeRadius()
	}
	return 0
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

// point2Slice renders model points as [x,y] pairs for the wire DTOs (nil in,
// nil out — shapeless kinds keep omitting the field).
func point2Slice(pts []math.Point2) [][]float64 {
	if len(pts) == 0 {
		return nil
	}
	out := make([][]float64, len(pts))
	for i, p := range pts {
		out[i] = []float64{float64(p.X), float64(p.Y)}
	}
	return out
}

// geometricShape returns a geometric constraint's wire kind and the ids of the
// entities it relates, both self-reported by the constraint's KindedConstraint
// capability (#1625) — the per-consumer type switches this replaces are the
// structure that shipped #1574's enumerable-but-not-creatable Symmetry.
func geometricShape(c sketch.Constraint) (types.GeometricConstraintKind, []uint64) {
	kc, ok := c.(sketch.KindedConstraint)
	if !ok {
		return types.GeoConstraintUnknown, nil
	}
	return wireConstraintKind(kc.ConstraintKind()), ids(kc.RelatedEntities()...)
}

// wireConstraintKind maps the model's persisted constraint vocabulary to the
// wire enum at the router boundary (the two follow different compatibility
// contracts: .obk stability vs API SemVer). The wire enum is coarser: a
// circular tangent enumerates as the wire "tangent".
func wireConstraintKind(kind sketch.ConstraintKind) types.GeometricConstraintKind {
	if kind == sketch.CircularTangentKind {
		return types.GeoConstraintTangent
	}
	if wire, ok := wireConstraintKinds[kind]; ok {
		return wire
	}
	return types.GeoConstraintUnknown
}

// wireConstraintKinds is the 1:1 part of the model→wire kind mapping.
var wireConstraintKinds = map[sketch.ConstraintKind]types.GeometricConstraintKind{
	sketch.CoincidentKind:    types.GeoConstraintCoincident,
	sketch.PointOnLineKind:   types.GeoConstraintPointOnLine,
	sketch.MidpointKind:      types.GeoConstraintMidpoint,
	sketch.PointOnCircleKind: types.GeoConstraintPointOnCircle,
	// The two-point Horizontal/Vertical are Inventor's *align* forms; the single-line forms
	// carry the plain horizontal/vertical wire kinds (#1871).
	sketch.HorizontalKind:           types.GeoConstraintHorizontalAlign,
	sketch.VerticalKind:             types.GeoConstraintVerticalAlign,
	sketch.SingleLineHorizontalKind: types.GeoConstraintHorizontal,
	sketch.SingleLineVerticalKind:   types.GeoConstraintVertical,
	sketch.SymmetryKind:             types.GeoConstraintSymmetry,
	sketch.LineSymmetryKind:         types.GeoConstraintSymmetry,
	sketch.CircularSymmetryKind:     types.GeoConstraintSymmetry,
	sketch.ArcMidpointKind:          types.GeoConstraintMidpoint,
	// Ellipse-axis relations enumerate as their plain wire relation (#1879); the horizontal/
	// vertical forms enumerate as the single-operand horizontal/vertical (#1879 AC2).
	sketch.EllipseParallelKind:      types.GeoConstraintParallel,
	sketch.EllipsePerpendicularKind: types.GeoConstraintPerpendicular,
	sketch.EllipseCollinearKind:     types.GeoConstraintCollinear,
	sketch.EllipseHorizontalKind:    types.GeoConstraintHorizontal,
	sketch.EllipseVerticalKind:      types.GeoConstraintVertical,
	sketch.FixKind:                  types.GeoConstraintFix,
	sketch.ParallelKind:             types.GeoConstraintParallel,
	sketch.PerpendicularKind:        types.GeoConstraintPerpendicular,
	sketch.CollinearKind:            types.GeoConstraintCollinear,
	sketch.EqualLengthKind:          types.GeoConstraintEqualLength,
	sketch.TangentKind:              types.GeoConstraintTangent,
	sketch.ConcentricKind:           types.GeoConstraintConcentric,
	sketch.EqualRadiusKind:          types.GeoConstraintEqualRadius,
	sketch.SmoothKind:               types.GeoConstraintSmooth,
	sketch.GroundKind:               types.GeoConstraintGround,
	sketch.OffsetKind:               types.GeoConstraintOffset,
	sketch.PatternLinkKind:          types.GeoConstraintPattern,
	sketch.TextBoxAnchorKind:        types.GeoConstraintTextBox,
	sketch.CustomKind:               types.GeoConstraintCustom,
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
