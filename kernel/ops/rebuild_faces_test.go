// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// multiSpanPatch is a genuinely single-span bicubic Bézier carried on extra spans by inserting
// knots (F01): same geometry, denser net — the over-defined surface a rebuild cleans up.
func multiSpanPatch(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := make([][]math.Point3, 4)
	w := make([][]float64, 4)
	for i := range 4 {
		ctrl[i] = make([]math.Point3, 4)
		w[i] = []float64{1, 1, 1, 1}
		for j := range 4 {
			ctrl[i][j] = math.P3(float64(i), float64(j), float64((i-1)*(j-1))*0.4)
		}
	}
	bez := []float64{0, 0, 0, 0, 1, 1, 1, 1}
	s, err := geom.NewBSplineSurface(3, 3, ctrl, w, bez, bez)
	if err != nil {
		t.Fatalf("bicubic patch: %v", err)
	}
	for _, u := range []float64{0.5} {
		if s, err = s.InsertKnotU(u, 1); err != nil {
			t.Fatalf("InsertKnotU: %v", err)
		}
	}
	for _, v := range []float64{0.5} {
		if s, err = s.InsertKnotV(v, 1); err != nil {
			t.Fatalf("InsertKnotV: %v", err)
		}
	}
	return s
}

// surfaceFaceBody wraps a B-spline surface in a single-face surface body with straight boundary
// edges at the domain corners. The loop geometry is incidental — RebuildFaceSurfaces reads only
// the face surface and preserves the loops.
func surfaceFaceBody(t *testing.T, s geom.BSplineSurface) *topo.Body {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("rebuild", "body", 0)))
	corners := [4]math.Point3{s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 1), s.PointAt(0, 1)}
	v := make([]*topo.Vertex, 4)
	for i, p := range corners {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok("rebuild", "v", i)))
	}
	uses := make([]topo.Use, 4)
	for i := range 4 {
		j := (i + 1) % 4
		e := bld.AddEdge(geom.NewLineSegment(corners[i], corners[j]), v[i], v[j], topo.NewLineage(topo.Tok("rebuild", "e", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(s, topo.NewLineage(topo.Tok("rebuild", "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

func TestRebuildFaceSurfacesCollapsesMultiSpan(t *testing.T) {
	src := multiSpanPatch(t)
	body := surfaceFaceBody(t, src)
	out, dev, err := ops.RebuildFaceSurfaces(body, 3, 3, 4, 4, 0)
	if err != nil {
		t.Fatalf("RebuildFaceSurfaces: %v", err)
	}
	if dev > 1e-6 {
		t.Errorf("collapsing a multi-span bicubic face to a single span should be near-exact, dev=%g", dev)
	}
	face := out.Faces()[0]
	bs, ok := face.Geometry().(geom.BSplineSurface)
	if !ok {
		t.Fatalf("rebuilt face geometry is %T, want geom.BSplineSurface", face.Geometry())
	}
	if bs.UDegree != 3 || bs.VDegree != 3 {
		t.Errorf("rebuilt degrees = %dx%d, want 3x3", bs.UDegree, bs.VDegree)
	}
	if len(bs.Ctrl) != 4 || len(bs.Ctrl[0]) != 4 {
		t.Errorf("rebuilt net = %dx%d, want a 4x4 single span", len(bs.Ctrl), len(bs.Ctrl[0]))
	}
}

func TestRebuildFaceSurfacesErrorsWhenNoFreeformFace(t *testing.T) {
	// A planar box has only unbounded-domain analytic faces → nothing to rebuild.
	box := csgBox(math.P3(0, 0, 0), 1, 1, 1)
	if _, _, err := ops.RebuildFaceSurfaces(box, 3, 3, 4, 4, 0); err == nil {
		t.Error("a body with only analytic (unbounded-domain) faces should report nothing to rebuild")
	}
}
