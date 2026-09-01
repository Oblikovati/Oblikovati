// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Holed cylinder-wall tessellation (M2 Phase 2, Oblikovati/Oblikovati#1335). A drilled cylinder leaves its
// SIDE wall a full-period face with holes; the unroll-and-CDT mesher must cover the wall MINUS the holes
// with the correct curved area. The hole here is a (u,v)-axis-aligned "window" whose area on the cylinder
// is exactly R·Δu·Δv (the metric is flat), so the holed-wall area has a closed-form oracle.

const wallR, wallH = 3.0, 12.0

// windowedCylinderWall builds a cylinder side (radius wallR, axis z, z∈[0,wallH]) with its seam at angle 0
// and one rectangular window hole over u∈[uLo,uHi], v∈[vLo,vHi] — clear of the seam. The window's two
// horizontal edges are exact circle arcs (v=const on a cylinder) and its two vertical edges are line
// rulings (u=const), so the hole tessellates faithfully (not a decimated polyline).
func windowedCylinderWall(t *testing.T, uLo, uHi, vLo, vHi float64) *topo.Face {
	t.Helper()
	bottom, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	top := geom.Circle{Center: math.P3(0, 0, wallH), Normal: bottom.Normal, RefDir: bottom.RefDir, Radius: wallR}
	side, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)

	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("wall", "body", 0)))
	vb := bld.AddVertex(bottom.PointAt(0), topo.NewLineage(topo.Tok("wall", "vb", 0)))
	vt := bld.AddVertex(top.PointAt(0), topo.NewLineage(topo.Tok("wall", "vt", 0)))
	eb := bld.AddEdge(bottom, vb, vb, topo.NewLineage(topo.Tok("wall", "eb", 0)))
	et := bld.AddEdge(top, vt, vt, topo.NewLineage(topo.Tok("wall", "et", 0)))
	es := bld.AddEdge(geom.NewLineSegment(bottom.PointAt(0), top.PointAt(0)), vb, vt, topo.NewLineage(topo.Tok("wall", "es", 0)))
	hole := windowHoleLoop(bld, uLo, uHi, vLo, vHi)
	bld.AddFace(side, topo.NewLineage(topo.Tok("wall", "face", 0)),
		topo.OuterLoop(topo.Fwd(es), topo.Rev(et), topo.Rev(es), topo.Fwd(eb)), hole)
	return bld.Build().Faces()[0]
}

// windowHoleLoop adds the four corner vertices and four edges (two v-const arcs, two u-const line rulings)
// of a rectangular window on the cylinder wall, returning it as an inner (hole) loop.
func windowHoleLoop(bld *topo.Builder, uLo, uHi, vLo, vHi float64) topo.LoopSpec {
	z := math.V3(0, 0, 1)
	x := math.V3(1, 0, 0)
	c00, c10 := wallPoint(uLo, vLo), wallPoint(uHi, vLo)
	c11, c01 := wallPoint(uHi, vHi), wallPoint(uLo, vHi)
	v00 := bld.AddVertex(c00, topo.NewLineage(topo.Tok("win", "v00", 0)))
	v10 := bld.AddVertex(c10, topo.NewLineage(topo.Tok("win", "v10", 0)))
	v11 := bld.AddVertex(c11, topo.NewLineage(topo.Tok("win", "v11", 0)))
	v01 := bld.AddVertex(c01, topo.NewLineage(topo.Tok("win", "v01", 0)))
	arcB, _ := geom.NewArc3d(math.P3(0, 0, math.Scalar(vLo)), z, x, wallR, uLo, uHi-uLo)
	arcT, _ := geom.NewArc3d(math.P3(0, 0, math.Scalar(vHi)), z, x, wallR, uHi, uLo-uHi)
	eB := bld.AddEdge(arcB, v00, v10, topo.NewLineage(topo.Tok("win", "eB", 0)))
	eR := bld.AddEdge(geom.NewLineSegment(c10, c11), v10, v11, topo.NewLineage(topo.Tok("win", "eR", 0)))
	eT := bld.AddEdge(arcT, v11, v01, topo.NewLineage(topo.Tok("win", "eT", 0)))
	eL := bld.AddEdge(geom.NewLineSegment(c01, c00), v01, v00, topo.NewLineage(topo.Tok("win", "eL", 0)))
	return topo.InnerLoop(topo.Fwd(eB), topo.Fwd(eR), topo.Fwd(eT), topo.Fwd(eL))
}

// wallPoint is the cylinder-wall point at angle u and axial v.
func wallPoint(u, v float64) math.Point3 {
	return math.P3(math.Scalar(wallR*stdmath.Cos(u)), math.Scalar(wallR*stdmath.Sin(u)), math.Scalar(v))
}

// TestHoledCylinderWallAreaMatchesAnalytic meshes a windowed cylinder wall and checks the area equals the
// full lateral area minus the window's R·Δu·Δv — i.e. the hole really is cut, with correct curved area.
func TestHoledCylinderWallAreaMatchesAnalytic(t *testing.T) {
	t.Parallel()
	const uLo, uHi, vLo, vHi = stdmath.Pi / 4, stdmath.Pi / 2, 4.0, 8.0
	face := windowedCylinderWall(t, uLo, uHi, vLo, vHi)

	mesh := tessellateCurvedFace(face, DefaultQuality())
	if mesh == nil || len(mesh.Indices) == 0 {
		t.Fatal("holed cylinder wall produced no mesh")
	}
	got := meshArea(mesh)
	want := 2*stdmath.Pi*wallR*wallH - wallR*(uHi-uLo)*(vHi-vLo)
	// 3% tolerance: the drilled wall is triangulated boundary-only (interior Steiner points make the
	// hole's constrained-Delaunay recovery leak — like trimmedPatchMesh's documented pole/seam case), so
	// the curved area carries a slightly higher chord deficit than an interior-refined patch. The B-rep is
	// exact; the mesh is watertight (verified separately) — this only bounds the display/property error.
	if rel := stdmath.Abs(got-want) / want; rel > 0.03 {
		t.Errorf("holed wall area %.4f, want %.4f (full − window) — rel %.4f > 3%%", got, want, rel)
	}
}

// TestHoledCylinderWallMeshIsRouted confirms the dispatch reaches holedConicWallMesh for a holed
// periodic cylinder side rather than falling through to the full-domain grid (which ignores holes).
func TestHoledCylinderWallMeshIsRouted(t *testing.T) {
	t.Parallel()
	face := windowedCylinderWall(t, stdmath.Pi/4, stdmath.Pi/2, 4, 8)
	outer := FaceOuterBoundary(face, DefaultQuality())
	holes := faceHoleBoundaries(face, DefaultQuality())
	if _, ok := HoledConicWallMesh(face.Geometry(), outer, holes, DefaultQuality()); !ok {
		t.Error("holedConicWallMesh declined a holed periodic cylinder wall; the dispatch would fall back")
	}
}

// TestHoledCylinderWallDeclinesNoHoles: a plain (hole-free) cylinder side is the periodicBandGrid case, so
// the holed mesher must decline it.
func TestHoledCylinderWallDeclinesNoHoles(t *testing.T) {
	t.Parallel()
	side, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	outer := []math.Point3{wallPoint(0, 0), wallPoint(2, 0), wallPoint(4, 0), wallPoint(0, wallH)}
	if _, ok := HoledConicWallMesh(side, outer, nil, DefaultQuality()); ok {
		t.Error("a hole-free cylinder side should defer from the holed-wall mesher")
	}
}

// TestHoledCylinderWallDeclinesNonCylinder: only a cylinder unrolls with a flat metric, so a holed sphere
// (or any other surface) must defer — the unroll-and-CDT path is cylinder-specific.
func TestHoledCylinderWallDeclinesNonCylinder(t *testing.T) {
	t.Parallel()
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), wallR)
	outer := []math.Point3{wallPoint(0, 0), wallPoint(2, 0), wallPoint(4, 0)}
	hole := []math.Point3{wallPoint(1, 1), wallPoint(1.2, 1), wallPoint(1.1, 1.2)}
	if _, ok := HoledConicWallMesh(sph, outer, [][]math.Point3{hole}, DefaultQuality()); ok {
		t.Error("a non-cylinder surface should defer from the holed-wall mesher")
	}
}

// TestHoledCylinderWallDeclinesNonWrappingOuter: an outer loop that does NOT wrap the full period is an
// ordinary contractible patch (ToUVLoops handles it), so the wrapping-wall mesher must decline.
func TestHoledCylinderWallDeclinesNonWrappingOuter(t *testing.T) {
	t.Parallel()
	side, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), wallR)
	outer := []math.Point3{wallPoint(0, 0), wallPoint(0.5, 0), wallPoint(0.5, wallH), wallPoint(0, wallH)}
	hole := []math.Point3{wallPoint(0.2, 4), wallPoint(0.3, 4), wallPoint(0.25, 6)}
	if _, ok := HoledConicWallMesh(side, outer, [][]math.Point3{hole}, DefaultQuality()); ok {
		t.Error("a non-wrapping outer loop should defer from the holed-wall mesher")
	}
}

// squareHoleUV is a closed unit square hole loop in (u,v) offset by (du,0) — a synthetic near-pinch hole.
func squareHoleUV(du float64) []math.Point2 {
	return []math.Point2{
		math.P2(math.Scalar(du), 0), math.P2(math.Scalar(du+1), 0),
		math.P2(math.Scalar(du+1), 1), math.P2(math.Scalar(du), 1), math.P2(math.Scalar(du), 0),
	}
}

func TestMeanLoopChord2D(t *testing.T) {
	t.Parallel()
	if got := meanLoopChord2D(squareHoleUV(0)); stdmath.Abs(got-1) > 1e-9 { // every edge is a unit segment
		t.Errorf("meanLoopChord2D = %g, want 1", got)
	}
	if got := meanLoopChord2D([]math.Point2{math.P2(0, 0)}); got != 0 {
		t.Errorf("meanLoopChord2D(single) = %g, want 0", got)
	}
}

func TestMinCrossVertexDistance(t *testing.T) {
	t.Parallel()
	// squares at u∈[0,1] and u∈[5,6]: nearest vertices are u=1 and u=5, gap 4.
	if got := minCrossVertexDistance(squareHoleUV(0), squareHoleUV(5)); stdmath.Abs(got-4) > 1e-9 {
		t.Errorf("minCrossVertexDistance = %g, want 4", got)
	}
}

func TestNeckCorridorNodes(t *testing.T) {
	t.Parallel()
	// Two holes 0.2 apart (gap/chord = 0.2 < nearNeckChords): the corridor between them is seeded.
	if got := neckCorridorNodes([][]math.Point2{squareHoleUV(0), squareHoleUV(1.2)}); len(got) == 0 {
		t.Error("neckCorridorNodes seeded nothing for near-touching holes (gap 0.2)")
	}
	// Well-separated holes (gap 6 ≫ nearNeckChords·chord): no corridor seeding.
	if got := neckCorridorNodes([][]math.Point2{squareHoleUV(0), squareHoleUV(7)}); len(got) != 0 {
		t.Errorf("neckCorridorNodes seeded %d nodes for well-separated holes; want 0", len(got))
	}
	// Not exactly two holes: never seeds (an ordinary single-hole drilling is untouched).
	if got := neckCorridorNodes([][]math.Point2{squareHoleUV(0)}); got != nil {
		t.Errorf("neckCorridorNodes seeded for a single hole; want nil")
	}
}
