// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"

	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/geom"
)

// plainBSplineSurfaceFromStep maps the knot-less B-spline surface forms — B_SPLINE_SURFACE,
// BEZIER_SURFACE, UNIFORM_SURFACE, QUASI_UNIFORM_SURFACE — whose u/v knot vectors are IMPLIED
// by the subtype. Parameters: 0 name, 1 u_degree, 2 v_degree, 3 control_points net,
// 4 surface_form, 5 u_closed, 6 v_closed, 7 self_intersect. Non-rational (unit weights).
func plainBSplineSurfaceFromStep(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (geom.Surface, error) {
	uDeg, vDeg, err := degreePair(ent.Params)
	if err != nil {
		return nil, err
	}
	ctrl, err := controlNet(g, ent.Params, 3, scale)
	if err != nil {
		return nil, err
	}
	nU, nV := len(ctrl), len(ctrl[0])
	uKnots := implicitKnots(ent.Keyword, uDeg, nU)
	vKnots := implicitKnots(ent.Keyword, vDeg, nV)
	return geom.NewBSplineSurface(uDeg, vDeg, ctrl, unitNet(nU, nV), uKnots, vKnots)
}

// wrappedSurface maps a carrier surface (RECTANGULAR_TRIMMED_SURFACE, CURVE_BOUNDED_SURFACE)
// by resolving its basis surface at parameter index i. The face's own boundary loops already
// trim the surface, so the underlying basis surface is what we need; the entity's own trim
// parameters are ignored. (OFFSET_SURFACE is NOT handled here — it changes the geometry.)
func wrappedSurface(g *part21.EntityGraph, ent *part21.RawEntity, i int, scale float64) (geom.Surface, error) {
	ref, err := refParam(ent.Params, i)
	if err != nil {
		return nil, fmt.Errorf("geommap: %s basis surface: %w", ent.Keyword, err)
	}
	return Surface(g, ref, scale)
}
