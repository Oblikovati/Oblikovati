// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A faceted body's vertices already bound it, so it contributes NO curved support and pays only the
// scan that finds that out — which is what keeps a hundred-thousand-facet import affordable.
func TestAPlanarBodyHasNoCurvedSupport(t *testing.T) {
	t.Parallel()
	if got := supportOf(squareFaceBody(t, true)); len(got) != 0 {
		t.Errorf("a planar body contributes %d curved support points; want none", len(got))
	}
}

// A cylinder is the case the vertices cannot bound: it has exactly TWO, both on its seam, so the
// vertex spread ACROSS its axis is zero. The curved support has to reach its rims.
func TestACylinderSupportReachesItsRims(t *testing.T) {
	t.Parallel()
	body := seamedCylinderBody(t, 3, 7)
	if got := len(body.Vertices()); got != 2 {
		t.Fatalf("the fixture must have the cylinder's two seam vertices; got %d", got)
	}
	pts := supportOf(body)
	if len(pts) == 0 {
		t.Fatal("a cylindrical face must contribute curved support")
	}
	widest := 0.0
	for _, p := range pts {
		widest = stdmath.Max(widest, stdmath.Hypot(float64(p.X), float64(p.Y)))
	}
	if stdmath.Abs(widest-3) > 1e-12 { // tol:numeric — the fixture's own radius
		t.Errorf("the curved support reaches %v from the axis; the cylinder's radius is 3", widest)
	}
}

// seamedCylinderBody is one periodic cylindrical face with the seam topology brep.SolidCylinder
// builds: two seam vertices, two rim circles and a seam line, and no caps.
func seamedCylinderBody(t *testing.T, radius, height float64) *Body {
	t.Helper()
	lin := NewLineage(Tok("test", "cyl", 0))
	bld := NewBuilder(true, lin)
	base := math.P3(0, 0, 0)
	axis := math.V3(0, 0, 1)
	bottom, err := geom.NewCircle(base, axis, radius)
	if err != nil {
		t.Fatalf("bottom: %v", err)
	}
	topCenter := math.P3(0, 0, math.Scalar(height))
	top := geom.Circle{Center: topCenter, Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: radius}
	side, err := geom.NewCylinder(base, axis, radius)
	if err != nil {
		t.Fatalf("side: %v", err)
	}
	vb := bld.AddVertex(bottom.PointAt(0), lin)
	vt := bld.AddVertex(top.PointAt(0), lin)
	eb := bld.AddEdge(bottom, vb, vb, lin)
	et := bld.AddEdge(top, vt, vt, lin)
	es := bld.AddEdge(geom.NewLineSegment(bottom.PointAt(0), top.PointAt(0)), vb, vt, lin)
	bld.AddFace(side, lin, OuterLoop(Fwd(es), Rev(et), Rev(es), Fwd(eb)))
	return bld.Build()
}

// supportOf is the curved support of every face of a body — what the size classification collects
// while it walks the faces for their directions.
func supportOf(b *Body) []math.Point3 {
	var pts []math.Point3
	for _, f := range b.Faces() {
		if _, planar := geom.PlanarNormal(f.Geometry()); planar {
			continue
		}
		pts = f.AppendSupportPoints(pts)
	}
	return pts
}
