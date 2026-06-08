// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// unitPatch is a flat bilinear B-spline surface over the unit square in z=0 (PointAt(u,v)=(u,v,0)),
// so an L-shaped trim on it has a known 3D area — a controlled stand-in for an imported freeform
// face that exercises trimmedPatchMesh end to end (ParamAt → CDT → 3D mesh).
func unitPatch(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0)},
		{math.P3(1, 0, 0), math.P3(1, 1, 0)},
	}
	w := [][]float64{{1, 1}, {1, 1}}
	s, err := geom.NewBSplineSurface(1, 1, ctrl, w, []float64{0, 0, 1, 1}, []float64{0, 0, 1, 1})
	if err != nil {
		t.Fatalf("NewBSplineSurface: %v", err)
	}
	return s
}

func TestRefinedTrimmedMeshConcavePatch(t *testing.T) {
	s := unitPatch(t)
	// An L-shaped (concave) trim on the patch: the mesh must cover exactly the L, not bridge the
	// notch (the over-count) and not tear (the under-count) — verified by total 3D triangle area.
	outer := []math.Point3{
		math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 0.5, 0),
		math.P3(0.5, 0.5, 0), math.P3(0.5, 1, 0), math.P3(0, 1, 0),
	}
	m := trimmedPatchMesh(s, outer, nil)
	if m.TriangleCount() == 0 {
		t.Fatal("patch produced no triangles")
	}
	var area float64
	bad := 0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		a, b, c := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		gn := a.VectorTo(b).Cross(a.VectorTo(c))
		area += stdmath.Sqrt(float64(gn.Dot(gn))) / 2
		sn := m.Normals[m.Indices[i]].Add(m.Normals[m.Indices[i+1]]).Add(m.Normals[m.Indices[i+2]])
		if gn.Dot(sn) < 0 {
			bad++
		}
	}
	if stdmath.Abs(area-0.75) > 1e-6 {
		t.Errorf("L-trim mesh area = %g, want 0.75 (notch bridged or torn)", area)
	}
	if bad > 0 {
		t.Errorf("%d triangles wind against their vertex normals", bad)
	}
}

// TestGridPatchMeshAddsInteriorNodes guards the sphere-cap smoothness fix: a curved sphere patch
// must mesh with INTERIOR (u,v) nodes (a refined surface), not a boundary-only fan of long flat
// triangles (the inner-bell-mouth slivers), and every triangle must agree with its vertex normals.
func TestGridPatchMeshAddsInteriorNodes(t *testing.T) {
	s, err := geom.NewSphere(math.P3(0, 0, 0), 10)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	// A 16-gon patch in (u,v) away from the pole/seam (a closed non-iso-rectangular trim).
	const n = 16
	cu, cv, r := 1.0, 0.4, 0.4
	var uv []math.Point2
	var p3 []math.Point3
	for k := 0; k < n; k++ {
		a := 2 * stdmath.Pi * float64(k) / n
		u, v := cu+r*stdmath.Cos(a), cv+r*stdmath.Sin(a)
		uv = append(uv, math.P2(math.Scalar(u), math.Scalar(v)))
		p3 = append(p3, s.PointAt(u, v))
	}
	m := gridPatchMesh(s, p3, nil, uv, nil)
	if m.TriangleCount() <= n {
		t.Errorf("expected interior refinement, got %d triangles (no interior nodes added)", m.TriangleCount())
	}
	// The patch must be CONSISTENTLY oriented — no shared edge traversed the same direction by both its
	// triangles (that is a fold / back-face) — and wound outward in aggregate. A per-triangle normal test
	// is too strict: a thin sliver's flat geometric normal can oppose its averaged vertex normals while
	// the triangle is correctly wound with its neighbours (see patchMeshFrom's global winding).
	dir := map[[2]int]int{}
	var agree float64
	for i := 0; i+2 < len(m.Indices); i += 3 {
		ia, ib, ic := m.Indices[i], m.Indices[i+1], m.Indices[i+2]
		dir[[2]int{ia, ib}]++
		dir[[2]int{ib, ic}]++
		dir[[2]int{ic, ia}]++
		a, b, c := m.Positions[ia], m.Positions[ib], m.Positions[ic]
		gn := a.VectorTo(b).Cross(a.VectorTo(c))
		agree += float64(gn.Dot(m.Normals[ia].Add(m.Normals[ib]).Add(m.Normals[ic])))
	}
	folds := 0
	for _, c := range dir {
		if c > 1 {
			folds++
		}
	}
	if folds > 0 {
		t.Errorf("%d shared edges traversed the same direction by both triangles (fold/back-face)", folds)
	}
	if agree <= 0 {
		t.Errorf("patch winds inward overall (aggregate gn·normal = %g)", agree)
	}
}

// TestWeldedFreeEdgeCount pins the watertightness metric the sphere-cap fallback decision uses: a
// closed tetrahedron welds to 0 free edges; dropping a face exposes 3 boundary edges.
func TestWeldedFreeEdgeCount(t *testing.T) {
	m := &Mesh{
		Positions: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 1)},
		Indices:   []int{0, 2, 1, 0, 1, 3, 1, 2, 3, 0, 3, 2}, // all 4 faces: closed
	}
	if free := weldedFreeEdgeCount(m); free != 0 {
		t.Errorf("closed tetrahedron has %d free edges; want 0", free)
	}
	m.Indices = m.Indices[:9] // drop one face → an open shell with a 3-edge boundary
	if free := weldedFreeEdgeCount(m); free != 3 {
		t.Errorf("tetrahedron missing one face has %d free edges; want 3", free)
	}
}

// TestPatchIsManifoldTolerance pins the fallback threshold: a patch whose welded free edges do not
// exceed its loop boundary is kept (manifold), one that exceeds it (a torn cap) is rejected.
func TestPatchIsManifoldTolerance(t *testing.T) {
	// one triangle, loop of its 3 vertices: 3 boundary edges == want 3 → kept.
	tri := &Mesh{Positions: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)}, Indices: []int{0, 1, 2}}
	if !patchIsManifold(tri, [][]int{{0, 1, 2}}) {
		t.Error("a single triangle with a 3-vertex loop should be manifold (3 free == want 3)")
	}
	// same triangle but claim a smaller loop (want 2) → 3 free > 2 → rejected (a torn patch).
	if patchIsManifold(tri, [][]int{{0, 1}}) {
		t.Error("free edges (3) exceeding the loop (2) should be rejected as torn")
	}
}

// TestMetricScaleCylinder pins the metric generalisation to analytic surfaces: a cylinder's (u,v) is
// anisotropic — a unit step in u (angle) spans R in 3D, in v (axial) spans 1 — and its v-domain is
// INFINITE, which metricScale must clamp instead of sampling ±Inf. So su≈R, sv≈1.
func TestMetricScaleCylinder(t *testing.T) {
	const r = 7.0
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	su, sv := metricScale(cyl)
	if stdmath.Abs(su-r) > 1e-9 {
		t.Errorf("su = %g; want the radius %g (3D length of a unit angular step)", su, r)
	}
	if stdmath.Abs(sv-1) > 1e-9 {
		t.Errorf("sv = %g; want 1 (3D length of a unit axial step)", sv)
	}
}
