// SPDX-License-Identifier: GPL-2.0-only

package brepfixture

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// HalfDiskFace builds a single planar face bounded by a half-circle of radius r: the top arc from
// angle 0 to π, closed by the diameter. It is the smallest face with one curved and one straight
// boundary edge, which is what the discretizer and the pcurve reconstructor are specified against.
//
// Example: f := brepfixture.HalfDiskFace(t, 5) // one arc edge, one segment edge, one loop
func HalfDiskFace(tb testing.TB, r float64) *topo.Face {
	tb.Helper()
	lin := topo.NewLineage(topo.Tok("fixture", "halfdisk", 0))
	bld := topo.NewBuilder(false, lin)
	a, b := math.P3(r, 0, 0), math.P3(-r, 0, 0)
	va, vb := bld.AddVertex(a, lin), bld.AddVertex(b, lin)
	arc, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r, 0, stdmath.Pi)
	if err != nil {
		tb.Fatalf("half-disk arc of radius %g: %v", r, err)
	}
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		tb.Fatalf("half-disk plane: %v", err)
	}
	eArc := bld.AddEdge(arc, va, vb, lin)                       // A→B along the top arc
	eSeg := bld.AddEdge(geom.NewLineSegment(b, a), vb, va, lin) // B→A along the diameter
	bld.AddFace(plane, lin, topo.OuterLoop(topo.Use{Edge: eArc}, topo.Use{Edge: eSeg}))
	return bld.Build().Faces()[0]
}
