// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// addSketchEntity creates a sketch entity of the requested kind/variant and returns its
// id and defining-point ids.
func addSketchEntity(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddSketchEntityArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return nil, err
	}
	if isCompositeKind(in.Kind) {
		return addCompositeEntity(part, sk, in)
	}
	ent, pointIDs, err := buildSketchEntity(part, sk, in)
	if err != nil {
		return nil, err
	}
	applyConstruction(ent, in.Construction)
	out := wire.AddSketchEntityResult{
		EntityID: uint64(ent.EntityID()), Kind: in.Kind, PointIDs: pointIDs,
	}
	// Inference runs on commit and reports what it applied (M06-F10, #625).
	applyEntityInference(s, sk, ent, &out)
	return json.Marshal(out)
}

// isCompositeKind reports whether a kind builds several entities at once.
func isCompositeKind(kind string) bool {
	switch kind {
	case "rectangle", "polygon", "slot", "polyline":
		return true
	default:
		return false
	}
}

// addCompositeEntity builds a multi-entity composite (rectangle/polygon/slot), applies
// the construction flag to every created entity, and returns all their ids.
func addCompositeEntity(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs) (json.RawMessage, error) {
	ents, center, err := buildComposite(part, sk, in)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, len(ents))
	for i, e := range ents {
		applyConstruction(e, in.Construction)
		ids[i] = uint64(e.EntityID())
	}
	// For a polygon the centre point (the construction-circle centre) follows the
	// vertices in PointIDs, so a caller can ground it and dimension the circumradius.
	points := compositePointIDs(ents)
	if center != nil {
		points = append(points, uint64(center.EntityID()))
	}
	return json.Marshal(wire.AddSketchEntityResult{
		EntityID: ids[0], Kind: in.Kind, EntityIDs: ids, PointIDs: points,
	})
}

// pointBearer is the subset of line-like entities that expose their two defining
// endpoints — the corners a composite (rectangle/polygon) is built from.
type pointBearer interface {
	StartPoint() *sketch.Point
	EndPoint() *sketch.Point
}

// compositePointIDs collects the unique defining-point ids of a composite's
// member entities, in first-seen order. Without these a rectangle/polygon cannot
// be dimensioned or geometrically constrained over the API (constraints and 2D
// dimensions reference points), which blocked fully-constrained parametric
// sketches. Member entities that are not line-like (a slot's arcs) contribute no
// points here.
func compositePointIDs(ents []sketch.Entity) []uint64 {
	seen := map[sketch.ID]bool{}
	var out []uint64
	add := func(p *sketch.Point) {
		if p == nil || seen[p.EntityID()] {
			return
		}
		seen[p.EntityID()] = true
		out = append(out, uint64(p.EntityID()))
	}
	for _, e := range ents {
		if pb, ok := e.(pointBearer); ok {
			add(pb.StartPoint())
			add(pb.EndPoint())
		}
	}
	return out
}

// buildComposite dispatches a composite create request to its model builder.
func buildComposite(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs) ([]sketch.Entity, *sketch.Point, error) {
	pts, err := toPoint2s(in.Points)
	if err != nil {
		return nil, nil, err
	}
	switch in.Kind {
	case "rectangle":
		ents, err := buildRectangle(sk, in.Variant, pts)
		return ents, nil, err
	case "polygon":
		return buildPolygon(sk, in, pts)
	case "slot":
		ents, err := buildSlot(part, sk, in, pts)
		return ents, nil, err
	case "polyline":
		ents, err := buildPolyline(sk, in, pts)
		return ents, nil, err
	default:
		return nil, nil, fmt.Errorf("sketch.addEntity: %q is not a composite kind", in.Kind)
	}
}

// buildPolyline connects the given points with shared-endpoint lines (Closed ⇒ a closed
// profile). Unlike the regular polygon, the points are arbitrary — the way to author an
// L-bracket, a custom extrusion section, or any non-rectilinear outline over the API.
func buildPolyline(sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) ([]sketch.Entity, error) {
	min := 2
	if in.Closed {
		min = 3
	}
	if len(pts) < min {
		return nil, fmt.Errorf("sketch.addEntity: polyline needs at least %d points, got %d", min, len(pts))
	}
	return sk.AddPolyline(pts, in.Closed)
}

// buildRectangle builds the two-corner (default), center, or three-point rectangle.
func buildRectangle(sk *sketch.Sketch, variant string, pts []math.Point2) ([]sketch.Entity, error) {
	switch variant {
	case "center":
		if err := wantPoints("rectangle center", pts, 2); err != nil {
			return nil, err
		}
		return sk.AddRectangleByCenter(pts[0], pts[1]), nil
	case "threePoint":
		if err := wantPoints("rectangle threePoint", pts, 3); err != nil {
			return nil, err
		}
		return sk.AddRectangleByThreePoints(pts[0], pts[1], pts[2])
	default:
		if err := wantPoints("rectangle", pts, 2); err != nil {
			return nil, err
		}
		return sk.AddRectangleByCorners(pts[0], pts[1]), nil
	}
}

// buildPolygon builds a regular polygon (variant "circumscribed" ⇒ apothem, else inscribed).
func buildPolygon(sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) ([]sketch.Entity, *sketch.Point, error) {
	if err := wantPoints("polygon", pts, 2); err != nil {
		return nil, nil, err
	}
	return sk.AddPolygon(pts[0], pts[1], in.Sides, in.Variant != "circumscribed")
}

// buildSlot builds a straight (center-to-center, default) or arc (3-point) slot of the
// given unit-bearing width.
func buildSlot(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) ([]sketch.Entity, error) {
	w, err := part.Units().Parse(in.Width, param.Length)
	if err != nil {
		return nil, fmt.Errorf("sketch.addEntity: slot width %q: %w", in.Width, err)
	}
	if in.Variant == "arc" {
		if err := wantPoints("arc slot", pts, 3); err != nil {
			return nil, err
		}
		return sk.AddArcSlot(pts[0], pts[1], pts[2], math.Scalar(w.Value), in.CCW)
	}
	if err := wantPoints("slot", pts, 2); err != nil {
		return nil, err
	}
	return sk.AddStraightSlot(pts[0], pts[1], math.Scalar(w.Value))
}

// buildSketchEntity dispatches a create request to the matching model constructor,
// returning the entity and the ids of its defining points.
func buildSketchEntity(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs) (sketch.Entity, []uint64, error) {
	pts, err := toPoint2s(in.Points)
	if err != nil {
		return nil, nil, err
	}
	switch in.Kind {
	case "line":
		return buildLine(sk, pts)
	case "point":
		return buildPoint(sk, pts)
	case "circle":
		return buildCircle(part, sk, in, pts)
	case "arc":
		return buildArc(sk, in, pts)
	default:
		return buildCurvedEntity(part, sk, in, pts)
	}
}

// buildCurvedEntity dispatches the conic/spline and corner-blend kinds (split from
// buildSketchEntity to keep each switch small).
func buildCurvedEntity(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) (sketch.Entity, []uint64, error) {
	switch in.Kind {
	case "ellipse":
		return buildEllipse(part, sk, in, pts)
	case "ellipticalArc":
		return buildEllipticalArc(part, sk, in, pts)
	case "spline":
		return buildSpline(sk, in, pts)
	case "fillet":
		return buildFillet(part, sk, in)
	case "chamfer":
		return buildChamfer(part, sk, in)
	default:
		return buildDerivedCurve(part, sk, in, pts)
	}
}

// buildDerivedCurve dispatches the M21 derived curves (equation/fixed/offset spline).
func buildDerivedCurve(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) (sketch.Entity, []uint64, error) {
	switch in.Kind {
	case "equationCurve":
		e, err := sk.EquationCurves().Add(in.XExpr, in.YExpr, in.T0, in.T1)
		if err != nil {
			return nil, nil, err
		}
		return e, nil, nil
	case "fixedSpline":
		if len(pts) < 2 {
			return nil, nil, fmt.Errorf("sketch.addEntity: fixedSpline needs at least 2 points, got %d", len(pts))
		}
		return sk.FixedSplines().Add(pts), nil, nil
	case "offsetSpline":
		return buildOffsetSpline(part, sk, in)
	default:
		return nil, nil, fmt.Errorf("sketch.addEntity: unknown kind %q", in.Kind)
	}
}

// buildOffsetSpline offsets a referenced parent spline by a unit-bearing distance.
func buildOffsetSpline(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs) (sketch.Entity, []uint64, error) {
	if len(in.EntityRefs) != 1 {
		return nil, nil, fmt.Errorf("sketch.addEntity: offsetSpline needs 1 parent spline ref, got %d", len(in.EntityRefs))
	}
	e, ok := sk.EntityByID(sketch.ID(in.EntityRefs[0]))
	if !ok {
		return nil, nil, fmt.Errorf("sketch.addEntity: no entity with id %d", in.EntityRefs[0])
	}
	parent, ok := e.(*sketch.Spline)
	if !ok {
		return nil, nil, fmt.Errorf("sketch.addEntity: entity %d is %T, want a spline", in.EntityRefs[0], e)
	}
	d, err := part.Units().Parse(in.Radius, param.Length)
	if err != nil {
		return nil, nil, fmt.Errorf("sketch.addEntity: offset distance %q: %w", in.Radius, err)
	}
	return sk.OffsetSplines().Add(parent, d.Value), nil, nil
}

// buildFillet rounds the corner between two referenced lines with a tangent arc.
func buildFillet(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs) (sketch.Entity, []uint64, error) {
	l1, l2, err := twoLineRefs(sk, in.EntityRefs)
	if err != nil {
		return nil, nil, err
	}
	r, err := part.Units().Parse(in.Radius, param.Length)
	if err != nil {
		return nil, nil, fmt.Errorf("sketch.addEntity: fillet radius %q: %w", in.Radius, err)
	}
	arc, err := sk.AddFillet(l1, l2, math.Scalar(r.Value))
	if err != nil {
		return nil, nil, err
	}
	return arc, arcPointIDs(arc), nil
}

// buildChamfer bevels the corner between two referenced lines (Distance2 defaults to Radius).
func buildChamfer(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs) (sketch.Entity, []uint64, error) {
	l1, l2, err := twoLineRefs(sk, in.EntityRefs)
	if err != nil {
		return nil, nil, err
	}
	d1, err := part.Units().Parse(in.Radius, param.Length)
	if err != nil {
		return nil, nil, fmt.Errorf("sketch.addEntity: chamfer distance %q: %w", in.Radius, err)
	}
	d2 := d1
	if in.Distance2 != "" {
		if d2, err = part.Units().Parse(in.Distance2, param.Length); err != nil {
			return nil, nil, fmt.Errorf("sketch.addEntity: chamfer distance2 %q: %w", in.Distance2, err)
		}
	}
	line, err := sk.AddChamfer(l1, l2, math.Scalar(d1.Value), math.Scalar(d2.Value))
	if err != nil {
		return nil, nil, err
	}
	return line, []uint64{uint64(line.A.EntityID()), uint64(line.B.EntityID())}, nil
}

// twoLineRefs resolves exactly two entity-reference ids to sketch lines.
func twoLineRefs(sk *sketch.Sketch, refs []uint64) (*sketch.Line, *sketch.Line, error) {
	if len(refs) != 2 {
		return nil, nil, fmt.Errorf("sketch.addEntity: corner blend needs 2 line refs, got %d", len(refs))
	}
	l1, err := lineRef(sk, refs[0])
	if err != nil {
		return nil, nil, err
	}
	l2, err := lineRef(sk, refs[1])
	if err != nil {
		return nil, nil, err
	}
	return l1, l2, nil
}

// lineRef resolves an entity id to a sketch line.
func lineRef(sk *sketch.Sketch, id uint64) (*sketch.Line, error) {
	e, ok := sk.EntityByID(sketch.ID(id))
	if !ok {
		return nil, fmt.Errorf("sketch.addEntity: no entity with id %d", id)
	}
	l, ok := e.(*sketch.Line)
	if !ok {
		return nil, fmt.Errorf("sketch.addEntity: entity %d is %T, want a line", id, e)
	}
	return l, nil
}

// buildLine creates a line between two points.
func buildLine(sk *sketch.Sketch, pts []math.Point2) (sketch.Entity, []uint64, error) {
	if err := wantPoints("line", pts, 2); err != nil {
		return nil, nil, err
	}
	l := sk.Lines().AddByTwoPoints(pts[0], pts[1])
	return l, []uint64{uint64(l.A.EntityID()), uint64(l.B.EntityID())}, nil
}

// buildPoint creates a standalone sketch point.
func buildPoint(sk *sketch.Sketch, pts []math.Point2) (sketch.Entity, []uint64, error) {
	if err := wantPoints("point", pts, 1); err != nil {
		return nil, nil, err
	}
	p := sk.Points().Add(pts[0])
	return p, []uint64{uint64(p.EntityID())}, nil
}

// buildCircle creates a circle by center+radius (default) or through three points.
func buildCircle(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) (sketch.Entity, []uint64, error) {
	if in.Variant == "threePoint" {
		if err := wantPoints("circle threePoint", pts, 3); err != nil {
			return nil, nil, err
		}
		c, err := sk.Circles().AddByThreePoints(pts[0], pts[1], pts[2])
		if err != nil {
			return nil, nil, err
		}
		return c, []uint64{uint64(c.Center.EntityID())}, nil
	}
	if err := wantPoints("circle centerRadius", pts, 1); err != nil {
		return nil, nil, err
	}
	r, err := part.Units().Parse(in.Radius, param.Length)
	if err != nil {
		return nil, nil, fmt.Errorf("sketch.addEntity: circle radius %q: %w", in.Radius, err)
	}
	c := sk.Circles().AddByCenterRadius(pts[0], math.Scalar(r.Value))
	return c, []uint64{uint64(c.Center.EntityID())}, nil
}

// buildArc creates an arc by center-start-end (default) or through three points.
func buildArc(sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) (sketch.Entity, []uint64, error) {
	if err := wantPoints("arc", pts, 3); err != nil {
		return nil, nil, err
	}
	if in.Variant == "threePoint" {
		a, err := sk.Arcs().AddByThreePoints(pts[0], pts[1], pts[2])
		if err != nil {
			return nil, nil, err
		}
		return a, arcPointIDs(a), nil
	}
	a := sk.Arcs().AddByCenterStartEnd(pts[0], pts[1], pts[2], in.CCW)
	return a, arcPointIDs(a), nil
}

// buildEllipse creates a full ellipse from a center, major-axis direction, and two radii.
func buildEllipse(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) (sketch.Entity, []uint64, error) {
	if err := wantPoints("ellipse", pts, 1); err != nil {
		return nil, nil, err
	}
	axis, majorR, minorR, err := ellipseFrame(part, in)
	if err != nil {
		return nil, nil, err
	}
	e := sk.Ellipses().Add(pts[0], axis, majorR, minorR)
	return e, []uint64{uint64(e.Center.EntityID())}, nil
}

// buildEllipticalArc creates an elliptical arc bounded by parametric start/end angles.
func buildEllipticalArc(part *compdef.PartComponentDefinition, sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) (sketch.Entity, []uint64, error) {
	if err := wantPoints("ellipticalArc", pts, 1); err != nil {
		return nil, nil, err
	}
	axis, majorR, minorR, err := ellipseFrame(part, in)
	if err != nil {
		return nil, nil, err
	}
	start, err := modelAngle(part, in.StartAngle)
	if err != nil {
		return nil, nil, err
	}
	end, err := modelAngle(part, in.EndAngle)
	if err != nil {
		return nil, nil, err
	}
	e := sk.EllipticalArcs().Add(pts[0], axis, majorR, minorR, math.Scalar(start), math.Scalar(end))
	return e, []uint64{uint64(e.Center.EntityID())}, nil
}

// ellipseFrame parses the shared ellipse inputs: a 2-component major-axis direction and
// the two unit-bearing radii.
func ellipseFrame(part *compdef.PartComponentDefinition, in wire.AddSketchEntityArgs) (math.Vector2, math.Scalar, math.Scalar, error) {
	if len(in.Axis) != 2 {
		return math.Vector2{}, 0, 0, fmt.Errorf("sketch.addEntity: ellipse axis needs 2 components, got %d", len(in.Axis))
	}
	majorR, err := part.Units().Parse(in.MajorRadius, param.Length)
	if err != nil {
		return math.Vector2{}, 0, 0, fmt.Errorf("sketch.addEntity: majorRadius %q: %w", in.MajorRadius, err)
	}
	minorR, err := part.Units().Parse(in.MinorRadius, param.Length)
	if err != nil {
		return math.Vector2{}, 0, 0, fmt.Errorf("sketch.addEntity: minorRadius %q: %w", in.MinorRadius, err)
	}
	return math.V2(in.Axis[0], in.Axis[1]), math.Scalar(majorR.Value), math.Scalar(minorR.Value), nil
}

// buildSpline creates a spline interpolating (default) or controlling its points.
func buildSpline(sk *sketch.Sketch, in wire.AddSketchEntityArgs, pts []math.Point2) (sketch.Entity, []uint64, error) {
	if len(pts) < 2 {
		return nil, nil, fmt.Errorf("sketch.addEntity: spline needs at least 2 points, got %d", len(pts))
	}
	var sp *sketch.Spline
	if in.Variant == "controlPoint" {
		sp = sk.Splines().AddByControlPoints(pts, in.Closed)
	} else {
		sp = sk.Splines().AddByPoints(pts, in.Closed)
	}
	if in.FitMethod != "" {
		m, ok := types.ParseSplineFitMethod(in.FitMethod)
		if !ok {
			return nil, nil, fmt.Errorf("sketch.addEntity: unknown fit method %q (want smooth|sweet|chord)", in.FitMethod)
		}
		sp.FitMethod = m
	}
	ids := make([]uint64, len(sp.Points))
	for i, p := range sp.Points {
		ids[i] = uint64(p.EntityID())
	}
	return sp, ids, nil
}

// arcPointIDs returns an arc's center/start/end point ids.
func arcPointIDs(a *sketch.Arc) []uint64 {
	return []uint64{uint64(a.Center.EntityID()), uint64(a.Start.EntityID()), uint64(a.End.EntityID())}
}

// toPoint2s converts wire [x,y] pairs (cm) to model points, validating each has 2 coords.
func toPoint2s(in [][]float64) ([]math.Point2, error) {
	out := make([]math.Point2, len(in))
	for i, p := range in {
		if len(p) != 2 {
			return nil, fmt.Errorf("sketch.addEntity: point %d has %d coords, want [x,y]", i, len(p))
		}
		out[i] = math.P2(math.Scalar(p[0]), math.Scalar(p[1]))
	}
	return out, nil
}

// wantPoints errors unless exactly n points were supplied for the named constructor.
func wantPoints(what string, pts []math.Point2, n int) error {
	if len(pts) != n {
		return fmt.Errorf("sketch.addEntity: %s needs %d points, got %d", what, n, len(pts))
	}
	return nil
}

// applyConstruction toggles an entity's construction flag when it supports one.
func applyConstruction(e sketch.Entity, construction bool) {
	if c, ok := e.(interface{ SetConstruction(bool) }); ok {
		c.SetConstruction(construction)
	}
}
