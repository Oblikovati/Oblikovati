// SPDX-License-Identifier: GPL-2.0-only

package transform_test

// Fixture builders restated from kernel/ops' test package. Go cannot share a _test.go
// helper across packages, and a shared fixture package would have to import kernel/ops,
// which kernel/ops' own tests could then not use (import cycle). This is the test
// scaffolding sonar.cpd.exclusions already accounts for.

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// csgBox builds a validated solid box of the given size with its near corner at p.
func csgBox(p math.Point3, sx, sy, sz float64) *topo.Body {
	m := subd.Box(sx, sy, sz)
	for i := range m.Verts {
		m.Verts[i] = m.Verts[i].TranslateBy(p.AsVector())
	}
	return subd.ToBody(m, "box")
}

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

// shellBox builds an axis-aligned box [0,sx]×[0,sy]×[0,sz].
func shellBox(sx, sy, sz float64) *topo.Body {
	return subd.ToBody(subd.Box(sx, sy, sz), "box")
}

// topFaceKey returns the reference key of the +Z (top) face.
func topFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).Z > 0.99 {
			return f.ReferenceKey()
		}
	}
	t.Fatal("no +Z face found")
	return nil
}
