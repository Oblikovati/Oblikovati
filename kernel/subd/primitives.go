// SPDX-License-Identifier: GPL-2.0-only

package subd

import "oblikovati/math"

// Primitive free-form starts (PBI-113). Each returns a control cage; callers refine
// it with [SubdivideN] and convert with [ToBody].

// Box returns a closed six-quad box cage of the given dimensions at the origin, with
// every face wound outward (so the converted body is a consistently oriented solid).
func Box(sx, sy, sz float64) Mesh {
	v := []math.Point3{
		math.P3(0, 0, 0), math.P3(sx, 0, 0), math.P3(sx, sy, 0), math.P3(0, sy, 0),
		math.P3(0, 0, sz), math.P3(sx, 0, sz), math.P3(sx, sy, sz), math.P3(0, sy, sz),
	}
	f := [][]int{
		{0, 3, 2, 1}, // bottom −Z
		{4, 5, 6, 7}, // top +Z
		{0, 1, 5, 4}, // front −Y
		{3, 7, 6, 2}, // back +Y
		{0, 4, 7, 3}, // left −X
		{1, 2, 6, 5}, // right +X
	}
	return Mesh{Verts: v, Faces: f}
}

// Plane returns an open single-quad cage on the z=0 plane (an n×n grid is a future
// convenience; one quad suffices as a refinable starting surface).
func Plane(sx, sy float64) Mesh {
	return Mesh{
		Verts: []math.Point3{math.P3(0, 0, 0), math.P3(sx, 0, 0), math.P3(sx, sy, 0), math.P3(0, sy, 0)},
		Faces: [][]int{{0, 1, 2, 3}},
	}
}

// QuadBall returns a sphere-like cage: a cube refined twice, then scaled so its limit
// surface approximates a sphere of the given radius (a common sub-D primitive start).
func QuadBall(radius float64) Mesh {
	m := SubdivideN(Box(2, 2, 2), 2)
	center := math.P3(1, 1, 1)
	for i, p := range m.Verts {
		dir, err := math.UnitVector3FromVector(center.VectorTo(p))
		if err != nil {
			continue
		}
		m.Verts[i] = center.TranslateBy(dir.AsVector().Scale(radius))
	}
	return m
}
