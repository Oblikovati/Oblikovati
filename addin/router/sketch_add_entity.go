// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/param"
	"github.com/Oblikovati/oblikovati/model/sketch"
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
	ent, pointIDs, err := buildSketchEntity(part, sk, in)
	if err != nil {
		return nil, err
	}
	applyConstruction(ent, in.Construction)
	return json.Marshal(wire.AddSketchEntityResult{
		EntityID: uint64(ent.EntityID()), Kind: in.Kind, PointIDs: pointIDs,
	})
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
	case "ellipse":
		return buildEllipse(part, sk, in, pts)
	case "ellipticalArc":
		return buildEllipticalArc(part, sk, in, pts)
	case "spline":
		return buildSpline(sk, in, pts)
	default:
		return nil, nil, fmt.Errorf("sketch.addEntity: unknown kind %q (want line|point|circle|arc|ellipse|ellipticalArc|spline)", in.Kind)
	}
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
