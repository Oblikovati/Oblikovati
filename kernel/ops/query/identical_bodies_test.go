// SPDX-License-Identifier: GPL-2.0-only

package query_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"

	"oblikovati.org/kernel/ops/heal"

	"oblikovati.org/kernel/ops/query"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// tetraEdges lists a tetrahedron's six edges as ordered corner pairs (low→high).
var tetraEdges = [6][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}

// tetraFaceCorners lists each face's corners wound CCW seen from OUTSIDE, for a
// positively oriented tetra (triple product of v1-v0, v2-v0, v3-v0 > 0).
var tetraFaceCorners = [4][3]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}}

// scaleneVerts is the corner tetra {x,y,z≥0, x/2+y/3+z/5≤1}: it lacks every coordinate-plane
// mirror, so its centroid-relative third moment ∮xyz dA is non-zero (a chiral reference body).
var scaleneVerts = [4]math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(0, 3, 0), math.P3(0, 0, 5)}

// tetraSolid builds a solid tetrahedron directly (shared vertices/edges, clean loops) the way the
// brep primitives do — so the analytic path integrates it. The four vertices must be positively
// oriented so every face winds outward.
func tetraSolid(feat string, verts [4]math.Point3) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	v := [4]*topo.Vertex{}
	for i, p := range verts {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	e := map[[2]int]*topo.Edge{}
	for i, pr := range tetraEdges {
		e[pr] = bld.AddEdge(geom.NewLineSegment(verts[pr[0]], verts[pr[1]]), v[pr[0]], v[pr[1]],
			topo.NewLineage(topo.Tok(feat, "edge", i)))
	}
	addTetraFaces(bld, verts, e, feat)
	return bld.Build()
}

// addTetraFaces adds the four outward triangular faces, reusing each shared edge with the winding
// that keeps its two uses oppositely oriented.
func addTetraFaces(bld *topo.Builder, verts [4]math.Point3, e map[[2]int]*topo.Edge, feat string) {
	for fi, c := range tetraFaceCorners {
		uses := make([]topo.Use, 3)
		for k := range 3 {
			uses[k] = tetraEdgeUse(e, c[k], c[(k+1)%3])
		}
		n := verts[c[0]].VectorTo(verts[c[1]]).Cross(verts[c[1]].VectorTo(verts[c[2]]))
		surf, _ := geom.NewPlane(verts[c[0]], n)
		bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", fi)), topo.OuterLoop(uses...))
	}
}

// tetraEdgeUse returns the forward use of edge a→b, or the reversed use of the stored b→a edge.
func tetraEdgeUse(e map[[2]int]*topo.Edge, a, b int) topo.Use {
	if edge, ok := e[[2]int{a, b}]; ok {
		return topo.Fwd(edge)
	}
	return topo.Rev(e[[2]int{b, a}])
}

// triFaceBody builds a one-triangle surface body wound CCW from outside (the 3-point analog of
// quadBody), so a set of them can be heal.Stitch-ed into a solid tetra that DECLINES the analytic path.
func triFaceBody(feat string, p0, p1, p2 math.Point3) *topo.Body {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok(feat, "body", 0)))
	pts := []math.Point3{p0, p1, p2}
	surf, _ := geom.NewPlane(p0, p0.VectorTo(p1).Cross(p1.VectorTo(p2)))
	v := make([]*topo.Vertex, 3)
	for i, p := range pts {
		v[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	uses := make([]topo.Use, 3)
	for i := range 3 {
		e := bld.AddEdge(geom.NewLineSegment(pts[i], pts[(i+1)%3]), v[i], v[(i+1)%3],
			topo.NewLineage(topo.Tok(feat, "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	bld.AddFace(surf, topo.NewLineage(topo.Tok(feat, "face", 0)), topo.OuterLoop(uses...))
	return bld.Build()
}

// stitchedTetra builds the same geometry as tetraSolid but by STITCHING four independent triangle
// faces. heal.Stitch's loop reordering makes query.AnalyticBodyTerms decline, so this body forces the MESH
// signature path — the twin used to prove the two paths are interchangeable.
func stitchedTetra(t *testing.T, feat string, verts [4]math.Point3) *topo.Body {
	t.Helper()
	faces := make([]*topo.Body, 4)
	for fi, c := range tetraFaceCorners {
		faces[fi] = triFaceBody(feat, verts[c[0]], verts[c[1]], verts[c[2]])
	}
	body, err := heal.Stitch(faces, 0, false, feat)
	if err != nil {
		t.Fatalf("heal.Stitch: %v", err)
	}
	return body
}

// mapVerts applies a rigid transform to every corner point.
func mapVerts(m math.Matrix4, verts [4]math.Point3) [4]math.Point3 {
	var out [4]math.Point3
	for i, p := range verts {
		out[i] = m.TransformPoint(p)
	}
	return out
}

// The analytic surface moments must reproduce the mesh surface moments (query.CentroidalMoments/
// triangleSkew) — the SAME quantities by two methods — so the analytic and mesh signature paths are
// interchangeable (#3449). A flat tetra tessellates to exact triangles, so the two agree to
// round-off; a cylinder's curved wall agrees as the tessellation refines.
func TestAnalyticSurfaceMomentsMatchMesh(t *testing.T) {
	t.Parallel()
	tetra := tetraSolid("moments", scaleneVerts)
	assertSurfaceMomentsAgree(t, tetra, 1e-7)
	cyl, err := brep.SolidCylinder(math.P3(1, -2, 0.5), math.V3(0, 0, 1), 2, 5)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	assertSurfaceMomentsAgree(t, cyl, 2e-3) // mesh converges to the analytic value with refinement
}

// assertSurfaceMomentsAgree compares the analytic surface principal moments and skew to the mesh
// (fine-tessellation) values for the same body.
func assertSurfaceMomentsAgree(t *testing.T, b *topo.Body, tol float64) {
	t.Helper()
	at, ok := query.AnalyticAreaTerms(b)
	if !ok {
		t.Fatal("body not analytically integrable")
	}
	mt, _ := query.AnalyticBodyTerms(b)
	am, askew := query.SurfaceCentroidalMoments(at, query.GeometryFromTerms(mt).Centroid)
	mesh, _ := tessellate.TessellateBody(b, query.Quality{ChordTolerance: 1e-4, AngleTolerance: 0.25 * stdmath.Pi / 180})
	props := query.MeshGeometryProperties(mesh)
	mm, mskew := query.CentroidalMoments(mesh, props)
	for i := range am {
		if !query.RelClose(am[i], mm[i], tol) {
			t.Errorf("principal moment %d: analytic %v vs mesh %v (tol %g)", i, am[i], mm[i], tol)
		}
	}
	if d := stdmath.Abs(askew - mskew); d > tol*stdmath.Max(stdmath.Abs(mskew), am[2]) {
		t.Errorf("surface skew: analytic %v vs mesh %v (Δ %g)", askew, mskew, d)
	}
}

// Two GEOMETRICALLY CONGRUENT bodies built differently — one via the clean builder (analytic path),
// one via heal.Stitch (mesh fallback) — must get equal signatures and group together. Before the surface-
// moment fix the analytic path used solid inertia and this failed (the regression the fix closes).
func TestSignatureInterchangeableAcrossPaths(t *testing.T) {
	t.Parallel()
	analytic := tetraSolid("clean", scaleneVerts)
	meshed := stitchedTetra(t, "stitch", scaleneVerts)
	if _, ok := query.AnalyticSignature(analytic); !ok {
		t.Fatal("clean tetra must take the analytic path")
	}
	if _, ok := query.AnalyticSignature(meshed); ok {
		t.Fatal("stitched tetra must DECLINE analytic and take the mesh path")
	}
	assertRigidInvariant(t, query.SignatureOf(analytic, ops.DefaultQuality()), query.SignatureOf(meshed, ops.DefaultQuality()))
	if g := query.GroupIdenticalBodies([]*topo.Body{analytic, meshed},
		query.IdenticalBodiesOptions{MatchReflection: true}, ops.DefaultQuality()); len(g) != 1 {
		t.Errorf("cross-path congruent bodies groups = %v, want 1", g)
	}
}

// A rigid-motion copy (rotate + translate) built on the OTHER path must still group with the
// original — the invariants are shared across paths and across the motion.
func TestCongruentAcrossPathsRotated(t *testing.T) {
	t.Parallel()
	analytic := tetraSolid("clean", scaleneVerts)
	axis, _ := math.UnitVector3FromVector(math.V3(1, 2, 3))
	m := math.Rotation4(0.9, axis, math.P3(1, 1, 1)).Mul(math.Translation4(math.V3(7, -4, 2)))
	meshed := stitchedTetra(t, "stitch", mapVerts(m, scaleneVerts))
	assertRigidInvariant(t, query.SignatureOf(analytic, ops.DefaultQuality()), query.SignatureOf(meshed, ops.DefaultQuality()))
	if g := query.GroupIdenticalBodies([]*topo.Body{analytic, meshed},
		query.IdenticalBodiesOptions{MatchReflection: true}, ops.DefaultQuality()); len(g) != 1 {
		t.Errorf("rigid-motion cross-path copy groups = %v, want 1", g)
	}
}

// A mirror keeps volume, area and the principal moments but flips the skew sign — the reflection
// discriminator. Grouping must fuse the mirror when MatchReflection is on and split it when off.
func TestMirroredTetraFlipsSkewViaAnalytic(t *testing.T) {
	t.Parallel()
	orig := tetraSolid("orig", scaleneVerts)
	xNormal, _ := math.UnitVector3FromVector(math.V3(1, 0, 0))
	mirror := transformed(t, orig, math.Reflection4(math.P3(0, 0, 0), xNormal))
	so := query.SignatureOf(orig, ops.DefaultQuality())
	sm := query.SignatureOf(mirror, ops.DefaultQuality())
	if stdmath.Abs(so.Skew+sm.Skew) > 1e-6*stdmath.Abs(so.Skew) {
		t.Errorf("mirror skew = %v, want %v (negated original)", sm.Skew, -so.Skew)
	}
	if stdmath.Abs(so.Skew) < 1e-3 {
		t.Fatalf("chiral tetra skew ~0 (%v): mirror test is vacuous", so.Skew)
	}
	bodies := []*topo.Body{orig, mirror}
	if g := query.GroupIdenticalBodies(bodies, query.IdenticalBodiesOptions{MatchReflection: true}, ops.DefaultQuality()); len(g) != 1 {
		t.Errorf("MatchReflection=true groups = %v, want 1", g)
	}
	if g := query.GroupIdenticalBodies(bodies, query.IdenticalBodiesOptions{MatchReflection: false}, ops.DefaultQuality()); len(g) != 2 {
		t.Errorf("MatchReflection=false groups = %v, want 2", g)
	}
}

func transformed(t *testing.T, b *topo.Body, m math.Matrix4) *topo.Body {
	t.Helper()
	out, err := transform.TransformBody(b, m, func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		t.Fatalf("transform.TransformBody: %v", err)
	}
	return out
}

// assertRigidInvariant checks the rotation/translation-invariant signature parts — volume, area and
// the sorted principal second moments — agree to the signature tolerance.
func assertRigidInvariant(t *testing.T, a, b query.BodySignature) {
	t.Helper()
	if !query.RelClose(a.Volume, b.Volume, 1e-6) || !query.RelClose(a.Area, b.Area, 1e-6) {
		t.Errorf("volume/area differ: %+v vs %+v", a, b)
	}
	for i := range a.Moments {
		if stdmath.Abs(a.Moments[i]-b.Moments[i]) > 1e-6*stdmath.Max(a.Moments[2], 1e-12) {
			t.Errorf("principal moment %d differs: %v vs %v", i, a.Moments[i], b.Moments[i])
		}
	}
}
