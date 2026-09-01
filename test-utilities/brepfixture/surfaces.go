// SPDX-License-Identifier: GPL-2.0-only

package brepfixture

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// SurfaceFaceBody wraps a B-spline surface in a one-face body whose boundary is the four straight
// segments between its corner points — a face carrying the surface with the SIMPLEST possible trim.
// It is deliberately not the surface's own iso-curve boundary (retopo.FullDomainBody): a test that
// wants to see whether an operation reads the surface or the trim needs the two to differ.
//
// Example: b := brepfixture.SurfaceFaceBody(t, s) // one face, four segment edges
// surfaceFaceBody wraps a B-spline surface in a single-face surface body with straight boundary
// edges at the domain corners. The loop geometry is incidental — RebuildFaceSurfaces reads only
// the face surface and preserves the loops.
func SurfaceFaceBody(tb testing.TB, s geom.BSplineSurface) *topo.Body {
	tb.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("fixture", "body", 0)))
	corners := [4]math.Point3{s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)}
	v := make([]*topo.Vertex, 4)
	for i, p := range corners {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("fixture", "v", i)))
	}
	uses := make([]topo.Use, 4)
	for i := range 4 {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(corners[i], corners[j]), v[i], v[j], topo.NewLineage(topo.Tok("fixture", "e", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(s, topo.NewLineage(topo.Tok("fixture", "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}
