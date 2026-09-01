// SPDX-License-Identifier: GPL-2.0-only

package brepfixture

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// QuadBody builds a single planar quadrilateral face as its own one-face body, wound p0→p1→p2→p3.
// It is the operand a stitch, a sew or a self-intersection test is built from: independent sheets
// that only become one body through the operation under test.
//
// Example: f := brepfixture.QuadBody("wall", p(0,0,0), p(1,0,0), p(1,0,1), p(0,0,1))
// quadBody builds a one-face surface body from four points wound CCW as seen from
// outside, so the plane normal points outward. Each face is an independent body —
// stitching must weld their coincident corners/edges.
func QuadBody(feat string, p0, p1, p2, p3 math.Point3) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	normal := p0.VectorTo(p1).Cross(p1.VectorTo(p2))
	surf, _ := geom.NewPlane(p0, normal)
	pts := []math.Point3{p0, p1, p2, p3}
	v := make([]*topo.Vertex, 4)
	for i, p := range pts {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	uses := make([]topo.Use, 4)
	for i := range 4 {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(pts[i], pts[j]), v[i], v[j], topo.NewLineage(topo.Tok(feat, "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// CubeFaces returns the six independent quad sheets of the unit cube, each wound outward. Stitching
// or sewing them must produce one closed solid; that is the acceptance for both.
//
// Example: body, err := heal.Stitch(brepfixture.CubeFaces(), 0, false, "stitch")
// cubeFaces returns the six outward-oriented quad surface bodies of the unit cube.
func CubeFaces() []*topo.Body {
	p := math.P3
	return []*topo.Body{
		QuadBody("bottom", p(0, 0, 0), p(0, 1, 0), p(1, 1, 0), p(1, 0, 0)), // -Z
		QuadBody("top", p(0, 0, 1), p(1, 0, 1), p(1, 1, 1), p(0, 1, 1)),    // +Z
		QuadBody("front", p(0, 0, 0), p(1, 0, 0), p(1, 0, 1), p(0, 0, 1)),  // -Y
		QuadBody("back", p(0, 1, 0), p(0, 1, 1), p(1, 1, 1), p(1, 1, 0)),   // +Y
		QuadBody("left", p(0, 0, 0), p(0, 0, 1), p(0, 1, 1), p(0, 1, 0)),   // -X
		QuadBody("right", p(1, 0, 0), p(1, 1, 0), p(1, 1, 1), p(1, 0, 1)),  // +X
	}
}
