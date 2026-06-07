// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"

	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// CurveKind discriminates a mapped STEP curve so the topology layer can trim it to
// an edge's vertices (an EDGE_CURVE bounds a curve by two vertices).
type CurveKind uint8

const (
	// CurveLine is an infinite STEP LINE — trimmed to a LineSegment between vertices.
	CurveLine CurveKind = iota
	// CurveCircle is a STEP CIRCLE — trimmed to an Arc3d (or kept full) by vertices.
	CurveCircle
	// CurveBSpline is a STEP B_SPLINE_CURVE_WITH_KNOTS — already bounded.
	CurveBSpline
)

// MappedCurve carries a STEP curve's analytic parameters so the topology layer can
// build the correctly-trimmed kernel edge curve. Only the fields for Kind matter.
type MappedCurve struct {
	Kind CurveKind
	// CurveCircle: defining frame + radius (Center=Frame.Origin, Normal=AxisZ, Ref=AxisX).
	Circle CircleParams
	// CurveBSpline: the fully-bounded curve (no trimming needed).
	BSpline geom.BSplineCurve
}

// CircleParams holds a STEP CIRCLE's geometry for arc trimming.
type CircleParams struct {
	Center math.Point3
	Normal math.Vector3
	RefDir math.Vector3
	Radius float64
}

// ErrUnsupportedCurve signals a STEP curve type with no kernel analogue.
type ErrUnsupportedCurve struct {
	Keyword string
	ID      int
}

func (e ErrUnsupportedCurve) Error() string {
	return fmt.Sprintf("geommap: unsupported curve %s (#%d)", e.Keyword, e.ID)
}

// Curve maps a STEP curve entity to a MappedCurve. LINE/CIRCLE expose analytic
// parameters for trimming; B_SPLINE_CURVE_WITH_KNOTS is bounded as-is. Other curve
// types return ErrUnsupportedCurve.
func Curve(g *part21.EntityGraph, id int, scale float64) (MappedCurve, error) {
	ent, err := g.Lookup(id)
	if err != nil {
		return MappedCurve{}, err
	}
	if len(ent.Components) > 0 { // complex instance, e.g. a rational (weighted) B-spline
		return rationalBSplineCurve(g, ent, scale)
	}
	switch ent.Keyword {
	case "LINE":
		return MappedCurve{Kind: CurveLine}, nil
	case "CIRCLE":
		return circleFromStep(g, ent, scale)
	case "B_SPLINE_CURVE_WITH_KNOTS":
		return bsplineCurveFromStep(g, ent, scale)
	default:
		return MappedCurve{}, ErrUnsupportedCurve{Keyword: ent.Keyword, ID: id}
	}
}

// circleFromStep maps CIRCLE(name, placement, radius) to its analytic parameters.
func circleFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (MappedCurve, error) {
	ref, err := refParam(ent.Params, 1)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: CIRCLE placement: %w", err)
	}
	f, err := Placement(g, ref, scale)
	if err != nil {
		return MappedCurve{}, err
	}
	radius, err := floatParam(ent.Params, 2)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: CIRCLE radius: %w", err)
	}
	return MappedCurve{Kind: CurveCircle, Circle: CircleParams{
		Center: f.Origin, Normal: f.AxisZ, RefDir: f.AxisX, Radius: radius * scale,
	}}, nil
}

// bsplineCurveFromStep maps B_SPLINE_CURVE_WITH_KNOTS into a bounded geom.BSplineCurve.
// Parameters (0-indexed, including the leading entity name): 0 name, 1 degree,
// 2 control_points_list, 3 curve_form, 4 closed, 5 self_intersect, 6 knot_multiplicities,
// 7 knots. Non-rational (unit weights); RATIONAL is deferred.
func bsplineCurveFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (MappedCurve, error) {
	degree, err := intParam(ent.Params, 1)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: B_SPLINE_CURVE degree: %w", err)
	}
	ctrl, err := pointRefList(g, ent.Params, 2, scale)
	if err != nil {
		return MappedCurve{}, err
	}
	knots, err := expandedKnots(ent.Params, 6, 7)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: B_SPLINE_CURVE knots: %w", err)
	}
	curve, err := geom.NewBSplineCurveUniformWeights(degree, ctrl, knots)
	return MappedCurve{Kind: CurveBSpline, BSpline: curve}, err
}

// pointRefList resolves a flat list of CARTESIAN_POINT references at parameter i.
func pointRefList(g *part21.EntityGraph, params []part21.Value, i int, scale float64) ([]math.Point3, error) {
	if i >= len(params) {
		return nil, fmt.Errorf("missing point-list parameter %d (have %d)", i, len(params))
	}
	refs, err := params[i].AsList()
	if err != nil {
		return nil, err
	}
	return pointRow(g, refs, scale)
}
