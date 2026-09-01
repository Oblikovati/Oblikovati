// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestConformableSurfaces pins which surfaces have a boundary-faithful conforming re-mesher: planes,
// cylinders and cones, but not torus/sphere/B-spline (no conformingMesh for those yet, #1073).
func TestConformableSurfaces(t *testing.T) {
	t.Parallel()
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), 1)
	cases := []struct {
		name string
		s    geom.Surface
		want bool
	}{
		{"plane", pl, true},
		{"cylinder", geom.Cylinder{Radius: 1}, true},
		{"cone", geom.Cone{}, true},
		{"sphere", sph, false},
		{"torus", geom.Torus{}, false},
	}
	for _, c := range cases {
		if got := conformable(c.s); got != c.want {
			t.Errorf("conformable(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestConformingPlaneMeshKeepsNearCollinearPoint guards the plane-absorber fix (#1073): a planar
// face whose boundary carries a near-collinear point (as a near-straight shared arc edge produces)
// must reproduce BOTH segments through that point, so it conforms to a curved neighbour that keeps
// it — instead of the area-gated planarTris dropping it and cracking the shared edge.
func TestConformingPlaneMeshKeepsNearCollinearPoint(t *testing.T) {
	t.Parallel()
	// A pentagon on z=0 whose top edge carries a near-collinear apex (1, 2.0001) between (2,2) and (0,2).
	mid := math.P3(1, 2.0001, 0)
	loop := []math.Point3{
		math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(2, 2, 0), mid, math.P3(0, 2, 0),
	}
	f := planarFaceFromLoop(t, loop)

	m := conformingPlaneMesh(f, DefaultQuality())
	if m == nil {
		t.Fatal("conformingPlaneMesh returned nil for a simple planar face")
	}
	if !meshHasSegment(m, math.P3(2, 2, 0), mid) || !meshHasSegment(m, mid, math.P3(0, 2, 0)) {
		t.Error("conforming plane mesh dropped the near-collinear boundary point — it would crack a curved neighbour")
	}
}

// TestIsBareTriangleFace pins the #1766 predicate: a straight-edged planar triangle is bare, while a
// 5-point boundary (an absorber's shape, whose near-collinear point must be re-meshed) is NOT — so the
// conformance-repair skip can never fire on a face that could crack a neighbour.
func TestIsBareTriangleFace(t *testing.T) {
	t.Parallel()
	tri := planarFaceFromLoop(t, []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)})
	if !isBareTriangleFace(tri) {
		t.Error("a straight-edged planar triangle should be a bare triangle")
	}
	pent := planarFaceFromLoop(t, []math.Point3{
		math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(2, 2, 0), math.P3(1, 2.0001, 0), math.P3(0, 2, 0),
	})
	if isBareTriangleFace(pent) {
		t.Error("a 5-point (absorber) boundary must not be classified as a bare triangle")
	}
	for _, f := range tetra(1, math.V3(0, 0, 0)).Faces() {
		if !isBareTriangleFace(f) {
			t.Error("every tetra face is a straight-edged planar triangle")
		}
	}
	if !allBareTriangleFaces(tetra(1, math.V3(0, 0, 0)).Faces()) {
		t.Error("a tetra is a pure bare-triangle soup")
	}
}

// TestConformingMeshIsNoOpOnBareTriangle: re-meshing a triangle reproduces the same triangle, so
// conformingPlaneMesh returns nil (keep existing) — the skip that drops the per-face CDT (#1766).
func TestConformingMeshIsNoOpOnBareTriangle(t *testing.T) {
	t.Parallel()
	tri := planarFaceFromLoop(t, []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)})
	if m := conformingPlaneMesh(tri, DefaultQuality()); m != nil {
		t.Errorf("conformingPlaneMesh should skip a bare triangle (return nil), got %d tris", m.TriangleCount())
	}
}

// TestBareTriangleSoupTessellatesWatertight proves the body-level conformance skip is a genuine no-op
// on a closed triangle soup: the tetra still tessellates watertight (zero free edges), one triangle per
// face — i.e. skipping the repair pass did not reintroduce a crack (#1766).
func TestBareTriangleSoupTessellatesWatertight(t *testing.T) {
	t.Parallel()
	body := tetra(1, math.V3(0, 0, 0))
	if !allBareTriangleFaces(body.Faces()) {
		t.Fatal("tetra should be a bare-triangle soup (the skip path)")
	}
	for _, gq := range gateQualities() {
		mesh, _ := TessellateBody(body, gq.q)
		if got := weldedFreeEdgeCount(mesh); got != 0 {
			t.Errorf("%s quality: tessellated tetra has %d free edges, want 0 (watertight)", gq.name, got)
		}
		if mesh.TriangleCount() != 4 {
			t.Errorf("%s quality: tetra tessellated to %d triangles, want 4 (one per face)", gq.name, mesh.TriangleCount())
		}
	}
}

// planarFaceFromLoop builds a single planar (z=0) face from a CCW 3D point loop, each side a line
// edge — the fixture for the plane-conformance tests.
func planarFaceFromLoop(t *testing.T, loop []math.Point3) *topo.Face {
	t.Helper()
	return planarLoopBody(t, math.P3(0, 0, 0), math.V3(0, 0, 1), loop).Faces()[0]
}

// meshHasSegment reports whether some triangle of m has an edge between a and b (within weld tol).
func meshHasSegment(m *Mesh, a, b math.Point3) bool {
	near := func(p, qp math.Point3) bool { return p.DistanceTo(qp) < 1e-6 }
	for tr := 0; 3*tr+2 < len(m.Indices); tr++ {
		v := [3]math.Point3{m.Positions[m.Indices[3*tr]], m.Positions[m.Indices[3*tr+1]], m.Positions[m.Indices[3*tr+2]]}
		for k := range 3 {
			p, qp := v[k], v[(k+1)%3]
			if (near(p, a) && near(qp, b)) || (near(p, b) && near(qp, a)) {
				return true
			}
		}
	}
	return false
}
