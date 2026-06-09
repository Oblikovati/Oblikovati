// SPDX-License-Identifier: GPL-2.0-only

// Package geommap maps STEP geometric entities (CARTESIAN_POINT, DIRECTION,
// AXIS2_PLACEMENT_3D, the surfaces and curves) onto kernel/geom values, in both
// directions. The *_from_step.go files read STEP into geom; the *_to_step.go files
// emit geom as STEP. It depends only on kernel/geom and the part21 entity graph.
package geommap

import (
	"fmt"

	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/math"
)

// CartesianPoint reads a CARTESIAN_POINT entity into a Point3, applying the unit
// scale (file length unit → database mm). 2D points (length 2) get z = 0.
func CartesianPoint(g *part21.EntityGraph, id int, scale float64) (math.Point3, error) {
	ent, err := g.Lookup(id)
	if err != nil {
		return math.Point3{}, err
	}
	if ent.Keyword != "CARTESIAN_POINT" {
		return math.Point3{}, fmt.Errorf("geommap: #%d is %s, want CARTESIAN_POINT", id, ent.Keyword)
	}
	coords, err := floatList(ent.Params, 1)
	if err != nil {
		return math.Point3{}, fmt.Errorf("geommap: CARTESIAN_POINT #%d: %w", id, err)
	}
	x, y, z := pad3(coords)
	return math.P3(x*scale, y*scale, z*scale), nil
}

// Direction reads a DIRECTION entity into a Vector3 (unscaled — directions are
// dimensionless). Errors on a zero-length direction.
func Direction(g *part21.EntityGraph, id int) (math.Vector3, error) {
	ent, err := g.Lookup(id)
	if err != nil {
		return math.Vector3{}, err
	}
	if ent.Keyword != "DIRECTION" {
		return math.Vector3{}, fmt.Errorf("geommap: #%d is %s, want DIRECTION", id, ent.Keyword)
	}
	coords, err := floatList(ent.Params, 1)
	if err != nil {
		return math.Vector3{}, fmt.Errorf("geommap: DIRECTION #%d: %w", id, err)
	}
	x, y, z := pad3(coords)
	v := math.V3(x, y, z)
	if v.LengthSquared() == 0 {
		return math.Vector3{}, fmt.Errorf("geommap: DIRECTION #%d is zero-length", id)
	}
	return v, nil
}

// floatList decodes the parameter at index i as a list of reals.
func floatList(params []part21.Value, i int) ([]float64, error) {
	if i >= len(params) {
		return nil, fmt.Errorf("missing parameter %d (have %d)", i, len(params))
	}
	items, err := params[i].AsList()
	if err != nil {
		return nil, err
	}
	out := make([]float64, len(items))
	for j, item := range items {
		if out[j], err = item.AsFloat(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// pad3 returns the first three coordinates, padding missing ones with 0.
func pad3(c []float64) (x, y, z float64) {
	if len(c) > 0 {
		x = c[0]
	}
	if len(c) > 1 {
		y = c[1]
	}
	if len(c) > 2 {
		z = c[2]
	}
	return x, y, z
}

// refParam returns the entity id referenced by parameter i.
func refParam(params []part21.Value, i int) (int, error) {
	if i >= len(params) {
		return 0, fmt.Errorf("missing reference parameter %d (have %d)", i, len(params))
	}
	return params[i].AsRef()
}

// floatParam returns parameter i as a float.
func floatParam(params []part21.Value, i int) (float64, error) {
	if i >= len(params) {
		return 0, fmt.Errorf("missing numeric parameter %d (have %d)", i, len(params))
	}
	return params[i].AsFloat()
}
