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
