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

// planarFaceFromLoop builds a single planar (z=0) face from a CCW 3D point loop, each side a line
// edge — the fixture for the plane-conformance tests.
func planarFaceFromLoop(t *testing.T, loop []math.Point3) *topo.Face {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("c", "body", 0)))
	lin := topo.NewLineage(topo.Tok("c", "x", 0))
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	verts := make([]*topo.Vertex, len(loop))
	for i, p := range loop {
		verts[i] = bld.AddVertex(p, lin)
	}
	uses := make([]topo.Use, len(loop))
	for i := range loop {
		j := (i + 1) % len(loop)
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(loop[i], loop[j]), verts[i], verts[j], lin))
	}
	bld.AddFace(pl, lin, topo.OuterLoop(uses...))
	return bld.Build().Faces()[0]
}

// meshHasSegment reports whether some triangle of m has an edge between a and b (within weld tol).
func meshHasSegment(m *Mesh, a, b math.Point3) bool {
	near := func(p, qp math.Point3) bool { return p.DistanceTo(qp) < 1e-6 }
	for tr := 0; 3*tr+2 < len(m.Indices); tr++ {
		v := [3]math.Point3{m.Positions[m.Indices[3*tr]], m.Positions[m.Indices[3*tr+1]], m.Positions[m.Indices[3*tr+2]]}
		for k := 0; k < 3; k++ {
			p, qp := v[k], v[(k+1)%3]
			if (near(p, a) && near(qp, b)) || (near(p, b) && near(qp, a)) {
				return true
			}
		}
	}
	return false
}
