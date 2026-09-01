// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// dPrismBody builds a "D" prism: a major circular arc of radius r (cocylindrical with a
// SolidCylinder of the same radius/axis) closed by a chord, extruded z0→z1. It is the
// #2167 piston-head tool, built directly as an analytic B-rep (arc wall = true cylinder,
// chord wall = plane, two D caps) so ops tests can reproduce the cocylindrical cap-on-wall
// boolean without the model/feature extrude layer. theta is the chord's half-angle at the
// centre; material is the major-segment side (containing -x).
func dPrismBody(r, theta, z0, z1 float64, feat string) *topo.Body {
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(feat, role, i)) }
	cx, sy := r*stdmath.Cos(theta), r*stdmath.Sin(theta)
	a0, b0 := math.P3(cx, -sy, z0), math.P3(cx, sy, z0)
	a1, b1 := math.P3(cx, -sy, z1), math.P3(cx, sy, z1)
	arcB, _ := geom.NewArc3d(math.P3(0, 0, z0), math.V3(0, 0, 1), math.V3(1, 0, 0), r, theta, 2*stdmath.Pi-2*theta)
	arcT, _ := geom.NewArc3d(math.P3(0, 0, z1), math.V3(0, 0, 1), math.V3(1, 0, 0), r, theta, 2*stdmath.Pi-2*theta)
	side, _ := geom.NewCylinder(math.P3(0, 0, z0), math.V3(0, 0, 1), r)
	capB, _ := geom.NewPlane(math.P3(0, 0, z0), math.V3(0, 0, -1))
	capT, _ := geom.NewPlane(math.P3(0, 0, z1), math.V3(0, 0, 1))
	chordPl, _ := geom.NewPlane(math.P3(cx, 0, z0), math.V3(1, 0, 0))

	bld := topo.NewBuilder(true, lin("body", 0))
	va0, vb0 := bld.AddVertex(a0, lin("v", 0)), bld.AddVertex(b0, lin("v", 1))
	va1, vb1 := bld.AddVertex(a1, lin("v", 2)), bld.AddVertex(b1, lin("v", 3))
	eArcB := bld.AddEdge(arcB, vb0, va0, lin("e", 0))                          // B0→A0 (major arc)
	eArcT := bld.AddEdge(arcT, vb1, va1, lin("e", 1))                          // B1→A1
	eChordB := bld.AddEdge(geom.NewLineSegment(a0, b0), va0, vb0, lin("e", 2)) // A0→B0
	eChordT := bld.AddEdge(geom.NewLineSegment(a1, b1), va1, vb1, lin("e", 3)) // A1→B1
	eVA := bld.AddEdge(geom.NewLineSegment(a0, a1), va0, va1, lin("e", 4))     // A0→A1
	eVB := bld.AddEdge(geom.NewLineSegment(b0, b1), vb0, vb1, lin("e", 5))     // B0→B1

	bld.AddFace(capB, lin("f", 0), topo.OuterLoop(topo.Rev(eArcB), topo.Rev(eChordB)))
	bld.AddFace(capT, lin("f", 1), topo.OuterLoop(topo.Fwd(eChordT), topo.Fwd(eArcT)))
	bld.AddFace(side, lin("f", 2), topo.OuterLoop(topo.Fwd(eArcB), topo.Fwd(eVA), topo.Rev(eArcT), topo.Rev(eVB)))
	bld.AddFace(chordPl, lin("f", 3), topo.OuterLoop(topo.Fwd(eChordB), topo.Fwd(eVB), topo.Rev(eChordT), topo.Rev(eVA)))
	return bld.Build()
}
