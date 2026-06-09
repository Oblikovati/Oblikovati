// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"

	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// bsplineSurfaceFromStep maps B_SPLINE_SURFACE_WITH_KNOTS into a geom.BSplineSurface.
// Parameters (0-indexed, including the leading entity name): 0 name, 1 u_degree,
// 2 v_degree, 3 control_points_list, 4 surface_form, 5 u_closed, 6 v_closed,
// 7 self_intersect, 8 u_multiplicities, 9 v_multiplicities, 10 u_knots, 11 v_knots.
// Knots are expanded from the (knot, multiplicity) compact form the kernel does not
// use. Weights are unit (non-rational); RATIONAL surfaces are deferred (warned by
// the caller via the unsupported path until the rational complex form is handled).
func bsplineSurfaceFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (geom.Surface, error) {
	uDeg, vDeg, err := degreePair(ent.Params)
	if err != nil {
		return nil, err
	}
	ctrl, err := controlNet(g, ent.Params, 3, scale)
	if err != nil {
		return nil, err
	}
	uKnots, vKnots, err := surfaceKnots(ent.Params, uDeg, vDeg)
	if err != nil {
		return nil, err
	}
	weights := unitNet(len(ctrl), len(ctrl[0]))
	return geom.NewBSplineSurface(uDeg, vDeg, ctrl, weights, uKnots, vKnots)
}

// degreePair reads the u/v degrees (parameters 1 and 2, after the entity name).
func degreePair(params []part21.Value) (uDeg, vDeg int, err error) {
	if uDeg, err = intParam(params, 1); err != nil {
		return 0, 0, fmt.Errorf("geommap: B_SPLINE_SURFACE u_degree: %w", err)
	}
	if vDeg, err = intParam(params, 2); err != nil {
		return 0, 0, fmt.Errorf("geommap: B_SPLINE_SURFACE v_degree: %w", err)
	}
	return uDeg, vDeg, nil
}

// surfaceKnots reads and expands the u/v knot vectors (multiplicities at 8/9,
// distinct knots at 10/11 — after the leading entity name).
func surfaceKnots(params []part21.Value, _, _ int) (uKnots, vKnots []float64, err error) {
	if uKnots, err = expandedKnots(params, 8, 10); err != nil {
		return nil, nil, fmt.Errorf("geommap: B_SPLINE_SURFACE u knots: %w", err)
	}
	if vKnots, err = expandedKnots(params, 9, 11); err != nil {
		return nil, nil, fmt.Errorf("geommap: B_SPLINE_SURFACE v knots: %w", err)
	}
	return uKnots, vKnots, nil
}

// controlNet reads the rectangular CARTESIAN_POINT reference grid at parameter i.
func controlNet(g *part21.EntityGraph, params []part21.Value, i int, scale float64) ([][]math.Point3, error) {
	rows, err := params[i].AsList()
	if err != nil {
		return nil, err
	}
	net := make([][]math.Point3, len(rows))
	for r, row := range rows {
		refs, err := row.AsList()
		if err != nil {
			return nil, err
		}
		net[r], err = pointRow(g, refs, scale)
		if err != nil {
			return nil, err
		}
	}
	return net, nil
}

// pointRow resolves one row of control-point references.
func pointRow(g *part21.EntityGraph, refs []part21.Value, scale float64) ([]math.Point3, error) {
	pts := make([]math.Point3, len(refs))
	for c, ref := range refs {
		id, err := ref.AsRef()
		if err != nil {
			return nil, err
		}
		if pts[c], err = CartesianPoint(g, id, scale); err != nil {
			return nil, err
		}
	}
	return pts, nil
}

// expandedKnots reads the multiplicities at multIdx and distinct knots at knotIdx
// and expands them into the kernel's flat knot vector.
func expandedKnots(params []part21.Value, multIdx, knotIdx int) ([]float64, error) {
	mults, err := intList(params, multIdx)
	if err != nil {
		return nil, err
	}
	distinct, err := floatList(params, knotIdx)
	if err != nil {
		return nil, err
	}
	if len(mults) != len(distinct) {
		return nil, fmt.Errorf("knots/multiplicities length mismatch: %d vs %d", len(distinct), len(mults))
	}
	return repeatKnots(distinct, mults), nil
}

// repeatKnots expands distinct knots by their multiplicities into a flat vector.
func repeatKnots(distinct []float64, mults []int) []float64 {
	var out []float64
	for i, k := range distinct {
		for j := 0; j < mults[i]; j++ {
			out = append(out, k)
		}
	}
	return out
}

// unitNet builds a rows×cols weight net of all-ones (non-rational).
func unitNet(rows, cols int) [][]float64 {
	out := make([][]float64, rows)
	for r := range out {
		out[r] = make([]float64, cols)
		for c := range out[r] {
			out[r][c] = 1
		}
	}
	return out
}

// intList decodes parameter i as a list of integers.
func intList(params []part21.Value, i int) ([]int, error) {
	if i >= len(params) {
		return nil, fmt.Errorf("missing integer-list parameter %d (have %d)", i, len(params))
	}
	items, err := params[i].AsList()
	if err != nil {
		return nil, err
	}
	out := make([]int, len(items))
	for j, item := range items {
		if out[j], err = item.AsInt(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// intParam decodes parameter i as an integer.
func intParam(params []part21.Value, i int) (int, error) {
	if i >= len(params) {
		return 0, fmt.Errorf("missing integer parameter %d (have %d)", i, len(params))
	}
	return params[i].AsInt()
}
