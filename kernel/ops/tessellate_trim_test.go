// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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
	t.Parallel()
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

// fullCylinderFace builds the complete 2π periodic side of a cylinder (radius r, height h):
// two closed-circle edges and a seam, with the classic periodic loop (up the seam, around
// the top circle, down the seam, around the bottom). The shape produced by brep.SolidCylinder.
func fullCylinderFace(t *testing.T, r, h float64) *topo.Face {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "fcyl", 0))
	bld := topo.NewBuilder(false, lin)
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatal(err)
	}
	bot, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatal(err)
	}
	top := geom.Circle{Center: math.P3(0, 0, math.Scalar(h)), Normal: bot.Normal, RefDir: bot.RefDir, Radius: bot.Radius}
	pb, pt := bot.PointAt(0), top.PointAt(0)
	vb, vt := bld.AddVertex(pb, lin), bld.AddVertex(pt, lin)
	eb := bld.AddEdge(bot, vb, vb, lin)
	et := bld.AddEdge(top, vt, vt, lin)
	es := bld.AddEdge(geom.NewLineSegment(pb, pt), vb, vt, lin)
	bld.AddFace(cyl, lin, topo.OuterLoop(topo.Fwd(es), topo.Rev(et), topo.Rev(es), topo.Fwd(eb)))
	return bld.Build().Faces()[0]
}

// TestFullPeriodicCylinderFaceArea tessellates the complete 2π cylinder side and checks the
// mesh area equals the analytic lateral area 2π·r·h (a hair under, inscribed). Regression for
// periodicBandGrid: before it, a full-seam-wrap loop fell back to the surface's whole UV
// domain (unbounded height) and got the area/volume badly wrong.
func TestFullPeriodicCylinderFaceArea(t *testing.T) {
	t.Parallel()
	const r, h = 2.0, 5.0
	mesh := TessellateFace(fullCylinderFace(t, r, h), DefaultQuality())
	want := 2 * stdmath.Pi * r * h // ≈ 62.83
	if got := meshArea(mesh); got > want+1e-9 || (want-got)/want > 0.03 {
		t.Errorf("full cylinder side area = %g, want a hair under %g (2π·r·h, inscribed)", got, want)
	}
}

// fullConeFrustumFace builds the complete 2π band of a cone between v∈[v1,v2] (radii r=v·tan):
// two closed-circle edges and a straight ruling seam, with the same periodic loop as a cylinder.
func fullConeFrustumFace(t *testing.T, halfAngle, v1, v2 float64) *topo.Face {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "cone", 0))
	bld := topo.NewBuilder(false, lin)
	cone, err := geom.NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), halfAngle)
	if err != nil {
		t.Fatal(err)
	}
	circle := func(v float64) geom.Circle {
		c, err := geom.NewCircle(math.P3(0, 0, math.Scalar(v)), math.V3(0, 0, 1), v*stdmath.Tan(halfAngle))
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	lo, hi := circle(v1), circle(v2)
	hi = geom.Circle{Center: hi.Center, Normal: lo.Normal, RefDir: lo.RefDir, Radius: hi.Radius}
	pLo, pHi := lo.PointAt(0), hi.PointAt(0)
	vLo, vHi := bld.AddVertex(pLo, lin), bld.AddVertex(pHi, lin)
	eLo := bld.AddEdge(lo, vLo, vLo, lin)
	eHi := bld.AddEdge(hi, vHi, vHi, lin)
	es := bld.AddEdge(geom.NewLineSegment(pLo, pHi), vLo, vHi, lin)
	bld.AddFace(cone, lin, topo.OuterLoop(topo.Fwd(es), topo.Rev(eHi), topo.Rev(es), topo.Fwd(eLo)))
	return bld.Build().Faces()[0]
}

// TestConeFrustumFaceArea tessellates a full cone frustum band and checks the mesh area equals
// the analytic lateral area π(r1+r2)·slant (a hair under, inscribed). Proves the periodic-band
// tessellator handles a cone (varying radius), the foundation countersink holes build on.
func TestConeFrustumFaceArea(t *testing.T) {
	t.Parallel()
	const ha = stdmath.Pi / 4 // 45°, tan = 1
	const v1, v2 = 2.0, 4.0   // r1=2, r2=4
	mesh := TessellateFace(fullConeFrustumFace(t, ha, v1, v2), DefaultQuality())
	r1, r2 := v1*stdmath.Tan(ha), v2*stdmath.Tan(ha)
	slant := (v2 - v1) / stdmath.Cos(ha)
	want := stdmath.Pi * (r1 + r2) * slant // ≈ 53.3
	if got := meshArea(mesh); got > want+1e-9 || (want-got)/want > 0.03 {
		t.Errorf("cone frustum area = %g, want a hair under %g (π(r1+r2)·slant, inscribed)", got, want)
	}
}

// coneApexFace builds a full cone closing to its apex: a single rim-circle boundary (no seam),
// the shape of a drill point. apex at origin, axis +Z, rim at v=vRim.
func coneApexFace(t *testing.T, halfAngle, vRim float64) *topo.Face {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "tip", 0))
	bld := topo.NewBuilder(false, lin)
	cone, err := geom.NewCone(math.P3(0, 0, 0), math.V3(0, 0, 1), halfAngle)
	if err != nil {
		t.Fatal(err)
	}
	rim, err := geom.NewCircle(math.P3(0, 0, math.Scalar(vRim)), math.V3(0, 0, 1), vRim*stdmath.Tan(halfAngle))
	if err != nil {
		t.Fatal(err)
	}
	v := bld.AddVertex(rim.PointAt(0), lin)
	e := bld.AddEdge(rim, v, v, lin)
	bld.AddFace(cone, lin, topo.OuterLoop(topo.Fwd(e)))
	return bld.Build().Faces()[0]
}

// TestConeApexFaceArea tessellates a cone closing to its apex and checks the mesh area equals
// the analytic lateral area π·r²/sin(halfAngle) (a hair under, inscribed). Proves the apex-fan
// path, the foundation a conical drill point builds on.
func TestConeApexFaceArea(t *testing.T) {
	t.Parallel()
	const ha, vRim = stdmath.Pi / 4, 2.0 // 45°, rim radius 2
	mesh := TessellateFace(coneApexFace(t, ha, vRim), DefaultQuality())
	r := vRim * stdmath.Tan(ha)
	want := stdmath.Pi * r * r / stdmath.Sin(ha) // π·r²/sin(halfAngle) ≈ 17.77
	if got := meshArea(mesh); got > want+1e-9 || (want-got)/want > 0.03 {
		t.Errorf("cone apex area = %g, want a hair under %g (π·r²/sin, inscribed)", got, want)
	}
}

// TestTrimmedCurvedFaceOutwardWinding checks every emitted triangle winds outward (its
// geometric normal agrees with the cylinder's outward radial), needed for correct
// divergence-theorem volume of curved-faced solids.
func TestTrimmedCurvedFaceOutwardWinding(t *testing.T) {
	t.Parallel()
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
