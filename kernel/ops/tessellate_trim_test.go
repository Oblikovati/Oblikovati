// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// quarterCylinderFace builds a trimmed cylinder face (radius r, axis +Z) spanning the
// quarter u∈[0,π/2] and height v∈[0,h] — the shape of a 90° rolling-ball fillet face. Its
// boundary is two arc edges (bottom/top, on the cylinder) and two straight axial edges.
func quarterCylinderFace(t *testing.T, r, h float64) *topo.Face {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "qcyl", 0))
	bld := topo.NewBuilder(false, lin)
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatal(err)
	}
	// Corners (cyl.PointAt: (−r sin u, r cos u, v)).
	p00, p10 := cyl.PointAt(0, 0), cyl.PointAt(stdmath.Pi/2, 0)
	p0h, p1h := cyl.PointAt(0, h), cyl.PointAt(stdmath.Pi/2, h)
	v00, v10 := bld.AddVertex(p00, lin), bld.AddVertex(p10, lin)
	v0h, v1h := bld.AddVertex(p0h, lin), bld.AddVertex(p1h, lin)
	arc := func(z float64) geom.Arc3d {
		a, err := geom.NewArc3d(math.P3(0, 0, z), math.V3(0, 0, 1), math.V3(0, 1, 0), r, 0, stdmath.Pi/2)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	eBot := bld.AddEdge(arc(0), v00, v10, lin) // bottom arc, u:0→π/2
	eRight := bld.AddEdge(geom.NewLineSegment(p10, p1h), v10, v1h, lin)
	eTop := bld.AddEdge(arc(h), v0h, v1h, lin) // top arc, u:0→π/2
	eLeft := bld.AddEdge(geom.NewLineSegment(p0h, p00), v0h, v00, lin)
	bld.AddFace(cyl, lin, topo.OuterLoop(
		topo.Use{Edge: eBot},
		topo.Use{Edge: eRight},
		topo.Use{Edge: eTop, Reversed: true}, // (−r,0,h)→(0,r,h)
		topo.Use{Edge: eLeft},
	))
	return bld.Build().Faces()[0]
}

// TestTrimmedCurvedFaceArea tessellates a quarter-cylinder face and checks the mesh area
// equals the analytic patch area (arc length r·π/2 times height h) — proving the
// tessellator meshes the TRIM region, not the surface's full UV domain. Regression for
// tessellateCurvedFace, which gridded the whole cylinder ignoring the face's loops.
func TestTrimmedCurvedFaceArea(t *testing.T) {
	const r, h = 2.0, 3.0
	f := quarterCylinderFace(t, r, h)
	mesh := TessellateFace(f, Quality{ChordTolerance: 1e-3})
	want := r * (stdmath.Pi / 2) * h // ≈ 9.4248
	if got := meshArea(mesh); stdmath.Abs(got-want) > 0.02 {
		t.Errorf("quarter-cylinder mesh area = %g, want ≈ %g", got, want)
	}
	if mesh.VertexCount() <= 4 {
		t.Fatalf("trimmed curved face not subdivided along the arc: %d vertices", mesh.VertexCount())
	}
}

// TestTrimmedCurvedFaceOutwardWinding checks every emitted triangle winds outward (its
// geometric normal agrees with the cylinder's outward radial), needed for correct
// divergence-theorem volume of curved-faced solids.
func TestTrimmedCurvedFaceOutwardWinding(t *testing.T) {
	f := quarterCylinderFace(t, 2, 3)
	cyl := f.Geometry()
	mesh := TessellateFace(f, Quality{ChordTolerance: 1e-2})
	for i := 0; i+2 < len(mesh.Indices); i += 3 {
		a, b, c := mesh.Positions[mesh.Indices[i]], mesh.Positions[mesh.Indices[i+1]], mesh.Positions[mesh.Indices[i+2]]
		n := a.VectorTo(b).Cross(a.VectorTo(c))
		cen := math.P3((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3, (a.Z+b.Z+c.Z)/3)
		u, v := cyl.ParamAt(cen)
		if n.Dot(cyl.NormalAt(u, v)) < 0 {
			t.Fatalf("triangle %d winds inward", i/3)
		}
	}
}
