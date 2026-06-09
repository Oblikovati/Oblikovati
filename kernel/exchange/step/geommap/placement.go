// SPDX-License-Identifier: GPL-2.0-only

package geommap

import (
	"fmt"

	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/math"
)

// Frame is the orthonormal coordinate frame an AXIS2_PLACEMENT_3D names: its
// Origin (location, scaled to mm), its AxisZ (the surface/curve axis), and its
// AxisX (ref_direction, made exactly perpendicular to AxisZ by Gram-Schmidt). It
// is the kernel-side analogue of STEP's placement, feeding every analytic
// surface/curve constructor.
type Frame struct {
	Origin math.Point3
	AxisZ  math.Vector3
	AxisX  math.Vector3
}

// Placement reads an AXIS2_PLACEMENT_3D (params: name, location, axis, ref_dir).
// A null axis defaults to +Z, a null ref_direction to a direction orthogonal to
// the axis. ref_direction is orthogonalized so AxisX⊥AxisZ exactly.
func Placement(g *part21.EntityGraph, id int, scale float64) (Frame, error) {
	ent, err := g.Lookup(id)
	if err != nil {
		return Frame{}, err
	}
	if ent.Keyword != "AXIS2_PLACEMENT_3D" {
		return Frame{}, fmt.Errorf("geommap: #%d is %s, want AXIS2_PLACEMENT_3D", id, ent.Keyword)
	}
	return buildFrame(g, ent, scale)
}

// buildFrame assembles a Frame from a placement entity's parameters.
func buildFrame(g *part21.EntityGraph, ent *part21.RawEntity, scale float64) (Frame, error) {
	origin, err := pointParam(g, ent.Params, 1, scale)
	if err != nil {
		return Frame{}, err
	}
	axisZ, err := optionalDirection(g, ent.Params, 2, math.V3(0, 0, 1))
	if err != nil {
		return Frame{}, err
	}
	axisX, err := refXAxis(g, ent.Params, axisZ)
	if err != nil {
		return Frame{}, err
	}
	return Frame{Origin: origin, AxisZ: axisZ, AxisX: axisX}, nil
}

// pointParam resolves a CARTESIAN_POINT referenced by parameter i.
func pointParam(g *part21.EntityGraph, params []part21.Value, i int, scale float64) (math.Point3, error) {
	ref, err := refParam(params, i)
	if err != nil {
		return math.Point3{}, err
	}
	return CartesianPoint(g, ref, scale)
}

// optionalDirection resolves the DIRECTION at parameter i, or returns the default
// when the parameter is the '$' null marker.
func optionalDirection(g *part21.EntityGraph, params []part21.Value, i int, fallback math.Vector3) (math.Vector3, error) {
	if i >= len(params) || params[i].IsNull() {
		return fallback, nil
	}
	ref, err := params[i].AsRef()
	if err != nil {
		return math.Vector3{}, err
	}
	return Direction(g, ref)
}

// refXAxis derives an X axis perpendicular to axisZ: the orthogonalized
// ref_direction when present, else any vector orthogonal to the axis.
func refXAxis(g *part21.EntityGraph, params []part21.Value, axisZ math.Vector3) (math.Vector3, error) {
	raw, err := optionalDirection(g, params, 3, anyPerpendicular(axisZ))
	if err != nil {
		return math.Vector3{}, err
	}
	ortho := orthogonalize(raw, axisZ)
	if ortho.LengthSquared() == 0 {
		return anyPerpendicular(axisZ), nil
	}
	return ortho, nil
}

// orthogonalize removes the component of v parallel to axis (Gram-Schmidt).
func orthogonalize(v, axis math.Vector3) math.Vector3 {
	a := axis.Scale(1 / axis.Length())
	return v.Sub(a.Scale(v.Dot(a)))
}

// anyPerpendicular returns an arbitrary nonzero vector perpendicular to axis.
func anyPerpendicular(axis math.Vector3) math.Vector3 {
	trial := math.V3(1, 0, 0)
	if axis.Cross(trial).LengthSquared() < 1e-12 {
		trial = math.V3(0, 1, 0)
	}
	return axis.Cross(trial)
}
