// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// The ONE face census this package's result gates read (#3527).
//
// "Result gates are per-face (area, surface type, loop count) against the oracle" is a ground rule, and
// four separate loops in this package were tallying faces by surface kind — three of them by hand, in
// the test that needed them, so each one carried its own idea of what a census is and only one of them
// counted loops at all. A body that measures right can still be the wrong shape; the shape is the kinds
// AND the loops each face is bounded by.

// faceKindCensus is a result's faces tallied by analytic surface kind, with the total number of
// boundary loops over all of them. loops is what separates two bodies with the same kinds: a drilled
// ring's torus face carries the bore's two seams as holes, and a mesher or a stitch that dropped one
// leaves the kinds untouched.
type faceKindCensus struct {
	tori, cylinders, cones, spheres, planes, faces, loops int
}

// faceKindCensusOf tallies a body's faces by analytic surface kind and totals their boundary loops.
//
//	got := faceKindCensusOf(res) // faceKindCensus{tori: 1, cylinders: 1, faces: 2, loops: 4}
func faceKindCensusOf(b *topo.Body) faceKindCensus {
	c := faceKindCensus{faces: len(b.Faces())}
	for _, f := range b.Faces() {
		c.loops += len(f.Loops())
		switch f.Geometry().(type) {
		case geom.Torus:
			c.tori++
		case geom.Cylinder:
			c.cylinders++
		case geom.Cone:
			c.cones++
		case geom.Sphere:
			c.spheres++
		case geom.Plane:
			c.planes++
		}
	}
	return c
}
