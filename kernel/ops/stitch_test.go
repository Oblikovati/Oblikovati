// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// quadBody builds a one-face surface body from four points wound CCW as seen from
// outside, so the plane normal points outward. Each face is an independent body —
// stitching must weld their coincident corners/edges.
func quadBody(feat string, p0, p1, p2, p3 math.Point3) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	normal := p0.VectorTo(p1).Cross(p1.VectorTo(p2))
	surf, _ := geom.NewPlane(p0, normal)
	pts := []math.Point3{p0, p1, p2, p3}
	v := make([]*topo.Vertex, 4)
	for i, p := range pts {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	uses := make([]topo.Use, 4)
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(pts[i], pts[j]), v[i], v[j], topo.NewLineage(topo.Tok(feat, "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// cubeFaces returns the six outward-oriented quad surface bodies of the unit cube.
func cubeFaces() []*topo.Body {
	p := math.P3
	return []*topo.Body{
		quadBody("bottom", p(0, 0, 0), p(0, 1, 0), p(1, 1, 0), p(1, 0, 0)), // -Z
		quadBody("top", p(0, 0, 1), p(1, 0, 1), p(1, 1, 1), p(0, 1, 1)),    // +Z
		quadBody("front", p(0, 0, 0), p(1, 0, 0), p(1, 0, 1), p(0, 0, 1)),  // -Y
		quadBody("back", p(0, 1, 0), p(0, 1, 1), p(1, 1, 1), p(1, 1, 0)),   // +Y
		quadBody("left", p(0, 0, 0), p(0, 0, 1), p(0, 1, 1), p(0, 1, 0)),   // -X
		quadBody("right", p(1, 0, 0), p(1, 1, 0), p(1, 1, 1), p(1, 0, 1)),  // +X
	}
}

func TestStitchClosedSurfacesYieldsSolid(t *testing.T) {
	body, err := Stitch(cubeFaces(), 0, false, "stitch")
	if err != nil {
		t.Fatalf("Stitch: %v", err)
	}
	if !body.IsSolid() {
		t.Error("stitching the six closed cube faces should yield a solid")
	}
	if got := len(body.Faces()); got != 6 {
		t.Errorf("stitched cube has %d faces, want 6", got)
	}
	if got := len(BoundaryEdges(body)); got != 0 {
		t.Errorf("stitched cube has %d boundary edges, want 0 (watertight)", got)
	}
	if r := Validate(body); !r.Valid || !r.Closed || !r.Manifold || !r.OrientationOK {
		t.Errorf("stitched cube validation = %+v, want fully valid", r)
	}
}

func TestStitchMaintainAsSurfaceKeepsOpen(t *testing.T) {
	// Even when the quilt closes, maintainSurface keeps it a surface body.
	body, err := Stitch(cubeFaces(), 0, true, "stitch")
	if err != nil {
		t.Fatalf("Stitch: %v", err)
	}
	if body.IsSolid() {
		t.Error("maintainSurface should keep the result a surface body")
	}
}

func TestStitchOpenQuiltStaysSurface(t *testing.T) {
	// Drop the top face → the quilt cannot close, so the result is a surface body.
	faces := cubeFaces()[1:] // omit bottom? no — omit one to leave an opening
	body, err := Stitch(faces, 0, false, "stitch")
	if err != nil {
		t.Fatalf("Stitch: %v", err)
	}
	if body.IsSolid() {
		t.Error("an open quilt must not become a solid")
	}
	if len(BoundaryEdges(body)) == 0 {
		t.Error("an open quilt should report boundary edges")
	}
}

func TestStitchNoBodiesErrors(t *testing.T) {
	if _, err := Stitch(nil, 0, false, "stitch"); err == nil {
		t.Error("stitching no bodies should error")
	}
}
