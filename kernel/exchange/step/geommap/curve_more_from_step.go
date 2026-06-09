// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"
	"strings"

	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/geom"
)

// ellipseFromStep maps ELLIPSE(name, position, semi_axis_1, semi_axis_2) to its analytic
// parameters (the edge later trims it to an EllipticalArc by its vertices). The position's
// X axis is the major-axis direction.
func ellipseFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (MappedCurve, error) {
	ref, err := refParam(ent.Params, 1)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: ELLIPSE placement: %w", err)
	}
	f, err := Placement(g, ref, scale)
	if err != nil {
		return MappedCurve{}, err
	}
	major, err := floatParam(ent.Params, 2)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: ELLIPSE semi_axis_1: %w", err)
	}
	minor, err := floatParam(ent.Params, 3)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: ELLIPSE semi_axis_2: %w", err)
	}
	return MappedCurve{Kind: CurveEllipse, Ellipse: EllipseParams{
		Center: f.Origin, Normal: f.AxisZ, RefDir: f.AxisX, Major: major * scale, Minor: minor * scale,
	}}, nil
}

// plainBSplineCurveFromStep maps the knot-less B-spline curve forms — B_SPLINE_CURVE,
// BEZIER_CURVE, UNIFORM_CURVE, QUASI_UNIFORM_CURVE — whose knot vector is IMPLIED by the
// subtype rather than listed. Parameters: 0 name, 1 degree, 2 control_points_list,
// 3 curve_form, 4 closed, 5 self_intersect. Non-rational (unit weights).
func plainBSplineCurveFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (MappedCurve, error) {
	degree, err := intParam(ent.Params, 1)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: %s degree: %w", ent.Keyword, err)
	}
	ctrl, err := pointRefList(g, ent.Params, 2, scale)
	if err != nil {
		return MappedCurve{}, err
	}
	curve, err := geom.NewBSplineCurveUniformWeights(degree, ctrl, implicitKnots(ent.Keyword, degree, len(ctrl)))
	return MappedCurve{Kind: CurveBSpline, BSpline: curve}, err
}

// polylineFromStep maps POLYLINE(name, (points)) to a bounded geom.Polyline.
func polylineFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (MappedCurve, error) {
	pts, err := pointRefList(g, ent.Params, 1, scale)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: POLYLINE points: %w", err)
	}
	pl, err := geom.NewPolyline(pts)
	return MappedCurve{Kind: CurvePolyline, Polyline: pl}, err
}

// implicitKnots builds the knot vector a knot-less B-spline subtype implies, for n control
// points of the given degree (ISO 10303-42 b_spline_*_form). Works for both curve and
// surface keywords (BEZIER_CURVE/BEZIER_SURFACE, …) by matching the form prefix:
//   - BEZIER: a single clamped Bézier span — degree+1 zeros then degree+1 ones.
//   - UNIFORM: a uniform integer vector 0..n+degree (interpolates no endpoints).
//   - QUASI_UNIFORM / plain B_SPLINE: clamped uniform — end multiplicities degree+1,
//     interior knots single (the de-facto default and what interpolates the end points).
func implicitKnots(form string, degree, n int) []float64 {
	switch {
	case strings.HasPrefix(form, "BEZIER"):
		return clampedEnds(degree, 1) // one interior span ⇒ no interior knots
	case strings.HasPrefix(form, "UNIFORM"):
		out := make([]float64, n+degree+1)
		for i := range out {
			out[i] = float64(i)
		}
		return out
	default: // QUASI_UNIFORM_*, B_SPLINE_*
		return clampedEnds(degree, n-degree)
	}
}

// clampedEnds returns a clamped knot vector with degree+1 multiplicity at each end and
// (spans-1) evenly-spaced single interior knots, spanning [0, spans].
func clampedEnds(degree, spans int) []float64 {
	if spans < 1 {
		spans = 1
	}
	var out []float64
	for i := 0; i <= degree; i++ {
		out = append(out, 0)
	}
	for k := 1; k < spans; k++ {
		out = append(out, float64(k))
	}
	for i := 0; i <= degree; i++ {
		out = append(out, float64(spans))
	}
	return out
}
