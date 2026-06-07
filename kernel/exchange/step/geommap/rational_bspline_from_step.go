// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"
	"strings"

	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/geom"
)

// Rational (weighted) B-splines arrive as STEP complex instances — a single #id that
// combines several supertype components, e.g.
//
//	#id =( BOUNDED_CURVE() B_SPLINE_CURVE(deg, ctrl, …)
//	       B_SPLINE_CURVE_WITH_KNOTS(mults, knots, …) CURVE()
//	       GEOMETRIC_REPRESENTATION_ITEM() RATIONAL_B_SPLINE_CURVE(weights)
//	       REPRESENTATION_ITEM(name) )
//
// The geometry is split across the parts: B_SPLINE_CURVE carries degree + control points,
// B_SPLINE_CURVE_WITH_KNOTS the knot vector, and RATIONAL_B_SPLINE_CURVE the per-control-
// point weights. Unlike a simple instance, a component's parameters carry NO leading entity
// name (the name lives on REPRESENTATION_ITEM), so each part is indexed from 0. SolidWorks
// and most kernels export every freeform curve/surface this way, so a real-world STEP is
// unimportable without it.

// complexPartParams returns the parameters of a complex instance's component by keyword,
// or false when the instance has no such part.
func complexPartParams(ent *part21.RawEntity, keyword string) ([]part21.Value, bool) {
	for i := range ent.Components {
		if ent.Components[i].Keyword == keyword {
			return ent.Components[i].Params, true
		}
	}
	return nil, false
}

// rationalBSplineCurve maps a RATIONAL_B_SPLINE_CURVE complex instance to a weighted
// geom.BSplineCurve. Returns ErrUnsupportedCurve if the instance is some other complex type.
func rationalBSplineCurve(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (MappedCurve, error) {
	bc, ok1 := complexPartParams(ent, "B_SPLINE_CURVE")
	bk, ok2 := complexPartParams(ent, "B_SPLINE_CURVE_WITH_KNOTS")
	rb, ok3 := complexPartParams(ent, "RATIONAL_B_SPLINE_CURVE")
	if !ok1 || !ok2 || !ok3 {
		return MappedCurve{}, ErrUnsupportedCurve{Keyword: complexKeyword(ent), ID: ent.ID}
	}
	degree, err := intParam(bc, 0)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: rational B_SPLINE_CURVE degree: %w", err)
	}
	ctrl, err := pointRefList(g, bc, 1, scale)
	if err != nil {
		return MappedCurve{}, err
	}
	knots, err := expandedKnots(bk, 0, 1)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: rational B_SPLINE_CURVE knots: %w", err)
	}
	weights, err := floatList(rb, 0)
	if err != nil {
		return MappedCurve{}, fmt.Errorf("geommap: rational B_SPLINE_CURVE weights: %w", err)
	}
	curve, err := geom.NewBSplineCurve(degree, ctrl, weights, knots)
	return MappedCurve{Kind: CurveBSpline, BSpline: curve}, err
}

// rationalBSplineSurface maps a RATIONAL_B_SPLINE_SURFACE complex instance to a weighted
// geom.BSplineSurface. Returns ErrUnsupportedSurface if the instance is some other complex type.
func rationalBSplineSurface(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (geom.Surface, error) {
	bs, ok1 := complexPartParams(ent, "B_SPLINE_SURFACE")
	bk, ok2 := complexPartParams(ent, "B_SPLINE_SURFACE_WITH_KNOTS")
	rb, ok3 := complexPartParams(ent, "RATIONAL_B_SPLINE_SURFACE")
	if !ok1 || !ok2 || !ok3 {
		return nil, ErrUnsupportedSurface{Keyword: complexKeyword(ent), ID: ent.ID}
	}
	uDeg, vDeg, err := rationalSurfaceDegrees(bs)
	if err != nil {
		return nil, err
	}
	ctrl, err := controlNet(g, bs, 2, scale)
	if err != nil {
		return nil, err
	}
	uKnots, vKnots, err := rationalSurfaceKnots(bk)
	if err != nil {
		return nil, err
	}
	weights, err := floatNet(rb, 0)
	if err != nil {
		return nil, fmt.Errorf("geommap: rational B_SPLINE_SURFACE weights: %w", err)
	}
	return geom.NewBSplineSurface(uDeg, vDeg, ctrl, weights, uKnots, vKnots)
}

// rationalSurfaceDegrees reads the u/v degrees from the B_SPLINE_SURFACE component
// (parameters 0 and 1 — a component carries no leading name).
func rationalSurfaceDegrees(bs []part21.Value) (uDeg, vDeg int, err error) {
	if uDeg, err = intParam(bs, 0); err != nil {
		return 0, 0, fmt.Errorf("geommap: rational B_SPLINE_SURFACE u_degree: %w", err)
	}
	if vDeg, err = intParam(bs, 1); err != nil {
		return 0, 0, fmt.Errorf("geommap: rational B_SPLINE_SURFACE v_degree: %w", err)
	}
	return uDeg, vDeg, nil
}

// rationalSurfaceKnots reads and expands the u/v knot vectors from the
// B_SPLINE_SURFACE_WITH_KNOTS component (multiplicities at 0/1, distinct knots at 2/3).
func rationalSurfaceKnots(bk []part21.Value) (uKnots, vKnots []float64, err error) {
	if uKnots, err = expandedKnots(bk, 0, 2); err != nil {
		return nil, nil, fmt.Errorf("geommap: rational B_SPLINE_SURFACE u knots: %w", err)
	}
	if vKnots, err = expandedKnots(bk, 1, 3); err != nil {
		return nil, nil, fmt.Errorf("geommap: rational B_SPLINE_SURFACE v knots: %w", err)
	}
	return uKnots, vKnots, nil
}

// floatNet reads a rectangular grid of reals (a list of lists) at parameter i — the
// rational surface's weight net.
func floatNet(params []part21.Value, i int) ([][]float64, error) {
	if i >= len(params) {
		return nil, fmt.Errorf("missing float-grid parameter %d (have %d)", i, len(params))
	}
	rows, err := params[i].AsList()
	if err != nil {
		return nil, err
	}
	net := make([][]float64, len(rows))
	for r, row := range rows {
		items, listErr := row.AsList()
		if listErr != nil {
			return nil, listErr
		}
		net[r] = make([]float64, len(items))
		for c, item := range items {
			if net[r][c], err = item.AsFloat(); err != nil {
				return nil, err
			}
		}
	}
	return net, nil
}

// complexKeyword renders a complex instance's component keywords for an error message
// (e.g. "(BOUNDED_CURVE B_SPLINE_CURVE …)"), or the plain keyword for a simple instance.
func complexKeyword(ent *part21.RawEntity) string {
	if len(ent.Components) == 0 {
		return ent.Keyword
	}
	names := make([]string, len(ent.Components))
	for i := range ent.Components {
		names[i] = ent.Components[i].Keyword
	}
	return "(" + strings.Join(names, " ") + ")"
}
