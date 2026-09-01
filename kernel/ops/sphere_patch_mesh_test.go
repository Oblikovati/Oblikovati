// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Regression for the multi-arc sphere-patch bug (M2 Phase 1, Oblikovati/Oblikovati#1334): a sphere
// face bounded by several arcs (a box cut) meshed wrong through the lat/long (u,v) path — a hand-built
// octant came out ~38% over its analytic πR³/6. spherePatchMesh meshes it in a gnomonic chart, so the
// patch carries true curvature and the body's volume matches the analytic octant.

// octantBody builds the +,+,+ sphere octant: one spherical face bounded by three quarter ARCS (Arc3d,
// not full circles — TessellateEdge walks the whole curve domain, so a full Circle edge would mesh the
// whole circle) plus three planar quarter-disks meeting at the centre. Exactly 1/8 of the sphere.
func octantBody(t *testing.T, radius float64) *topo.Body {
	t.Helper()
	O := math.P3(0, 0, 0)
	A, B, C := math.P3(radius, 0, 0), math.P3(0, radius, 0), math.P3(0, 0, radius)
	sphere, _ := geom.NewSphere(O, radius)
	q := stdmath.Pi / 2
	circZ, _ := geom.NewArc3d(O, math.V3(0, 0, 1), math.V3(1, 0, 0), radius, 0, q) // A→B (equator)
	circX, _ := geom.NewArc3d(O, math.V3(1, 0, 0), math.V3(0, 1, 0), radius, 0, q) // B→C (x=0 plane)
	circY, _ := geom.NewArc3d(O, math.V3(0, 1, 0), math.V3(0, 0, 1), radius, 0, q) // C→A (y=0 plane)
	planeZ, _ := geom.NewPlane(O, math.V3(0, 0, -1))
	planeX, _ := geom.NewPlane(O, math.V3(-1, 0, 0))
	planeY, _ := geom.NewPlane(O, math.V3(0, -1, 0))
	lin := func(s string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(s, "x", i)) }

	bld := topo.NewBuilder(true, lin("body", 0))
	vO := bld.AddVertex(O, lin("v", 0))
	vA := bld.AddVertex(A, lin("v", 1))
	vB := bld.AddVertex(B, lin("v", 2))
	vC := bld.AddVertex(C, lin("v", 3))
	ab := bld.AddEdge(circZ, vA, vB, lin("arc", 0))
	bc := bld.AddEdge(circX, vB, vC, lin("arc", 1))
	ca := bld.AddEdge(circY, vC, vA, lin("arc", 2))
	oa := bld.AddEdge(geom.NewLineSegment(O, A), vO, vA, lin("r", 0))
	ob := bld.AddEdge(geom.NewLineSegment(O, B), vO, vB, lin("r", 1))
	oc := bld.AddEdge(geom.NewLineSegment(O, C), vO, vC, lin("r", 2))

	bld.AddFace(sphere, lin("sph", 0), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bc), topo.Fwd(ca)))
	bld.AddFace(planeZ, lin("pz", 0), topo.OuterLoop(topo.Fwd(ob), topo.Rev(ab), topo.Rev(oa)))
	bld.AddFace(planeX, lin("px", 0), topo.OuterLoop(topo.Fwd(oc), topo.Rev(bc), topo.Rev(ob)))
	bld.AddFace(planeY, lin("py", 0), topo.OuterLoop(topo.Fwd(oa), topo.Rev(ca), topo.Rev(oc)))
	return bld.Build()
}

// TestSpherePatchOctantVolume is the headline regression: the spherical triangle (3 arcs, a corner at
// the +z pole) must mesh to its true curvature so the octant volume is πR³/6, not the ~38%-over the old
// (u,v) path produced.
func TestSpherePatchOctantVolume(t *testing.T) {
	const R = 5.0
	body := octantBody(t, R)
	if r := Validate(body); !r.Valid || !r.Closed || !r.Manifold || !body.IsSolid() {
		t.Fatalf("octant not a valid closed manifold solid: %+v", r)
	}
	got := BodyGeometryProperties(body, DefaultQuality()).Volume
	want := stdmath.Pi * R * R * R / 6
	if rel := stdmath.Abs(got-want) / want; rel > 0.02 {
		t.Errorf("octant volume %.4f, want %.4f (rel %.4f > 2%%) — multi-arc sphere patch mis-meshed", got, want, rel)
	}
}

// TestSpherePatchDirectMeshIsCurved exercises spherePatchMesh directly on a sphere patch near the +z cap
// (corners off the pole): it must claim the patch, add interior Steiner points (else the patch is flat),
// and keep EVERY vertex exactly on the sphere (the gnomonic lift is exact).
func TestSpherePatchDirectMeshIsCurved(t *testing.T) {
	const R = 4.0
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), R)
	// A spherical patch sampled ON the sphere along its four bounding iso-curves (so the boundary
	// points are exact sphere points, as a real arc tessellation would supply).
	outer := sphereQuadBoundary(sphere, 0.2, 1.0, 0.6, 1.1, 6)
	m, ok := spherePatchMesh(nil, sphere, outer, nil, DefaultQuality())
	if !ok {
		t.Fatal("spherePatchMesh declined a hemisphere patch it should handle")
	}
	if len(m.Positions) <= len(outer) {
		t.Errorf("no interior Steiner points added (%d verts for %d boundary) — patch would be flat", len(m.Positions), len(outer))
	}
	for _, p := range m.Positions {
		if d := float64(p.DistanceTo(sphere.Center)); stdmath.Abs(d-R) > 1e-6 {
			t.Errorf("mesh vertex %v is off the sphere (radius %.6f, want %.1f)", p, d, R)
		}
	}
}

// sphereQuadBoundary samples the closed boundary of the (u,v)-rectangle [u0,u1]×[v0,v1] on the sphere,
// n points per side, every point exact on the surface.
func sphereQuadBoundary(s geom.Sphere, u0, u1, v0, v1 float64, n int) []math.Point3 {
	var out []math.Point3
	add := func(uf func(t float64) (u, v float64)) {
		for k := range n {
			u, v := uf(float64(k) / float64(n))
			out = append(out, s.PointAt(u, v))
		}
	}
	add(func(t float64) (float64, float64) { return u0 + (u1-u0)*t, v0 })
	add(func(t float64) (float64, float64) { return u1, v0 + (v1-v0)*t })
	add(func(t float64) (float64, float64) { return u1 - (u1-u0)*t, v1 })
	add(func(t float64) (float64, float64) { return u0, v1 - (v1-v0)*t })
	return out
}
