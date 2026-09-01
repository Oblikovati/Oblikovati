// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// unitQuadFace builds a unit square on z=0 with surface normal +Z, optionally with reversed
// sense (its material side then faces −Z — the shape of a Difference cut wall).
func unitQuadFace(reversed bool) *topo.Face {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("t", "body", 0)))
	lin := topo.NewLineage(topo.Tok("t", "x", 0))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	c := [4]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0)}
	v := [4]*topo.Vertex{}
	for i, p := range c {
		v[i] = bld.AddVertex(p, lin)
	}
	uses := make([]topo.Use, 4)
	for i := range c {
		j := (i + 1) % 4
		uses[i] = topo.Fwd(bld.AddEdge(geom.NewLineSegment(c[i], c[j]), v[i], v[j], lin))
	}
	if reversed {
		bld.AddReversedFace(pl, lin, topo.OuterLoop(uses...))
	} else {
		bld.AddFace(pl, lin, topo.OuterLoop(uses...))
	}
	return bld.Build().Faces()[0]
}

// A reversed face tessellates to the same surface with NEGATED normals and REVERSED triangle
// winding, so a curved cut wall (whose surface normal points into the removed material)
// presents its true material side to shading and the divergence-theorem volume.
func TestReversedFaceFlipsNormalsAndWinding(t *testing.T) {
	t.Parallel()
	q := ops.DefaultQuality()
	plain := ops.TessellateFace(unitQuadFace(false), q)
	rev := ops.TessellateFace(unitQuadFace(true), q)
	if plain.TriangleCount() != rev.TriangleCount() || plain.VertexCount() != rev.VertexCount() {
		t.Fatalf("reversed face changed mesh size: %d/%d vs %d/%d", plain.TriangleCount(), plain.VertexCount(), rev.TriangleCount(), rev.VertexCount())
	}
	for i := range plain.Normals {
		if plain.Normals[i].Add(rev.Normals[i]).Length() > 1e-9 {
			t.Fatalf("normal %d not negated: %+v vs %+v", i, plain.Normals[i], rev.Normals[i])
		}
	}
	if geomNormalZ(plain) <= 0 || geomNormalZ(rev) >= 0 {
		t.Errorf("winding not reversed: plain z=%g (want +), reversed z=%g (want −)", geomNormalZ(plain), geomNormalZ(rev))
	}
}

// geomNormalZ returns the z-component of the first triangle's geometric (winding) normal.
func geomNormalZ(m *ops.Mesh) float64 {
	a, b, c := m.Positions[m.Indices[0]], m.Positions[m.Indices[1]], m.Positions[m.Indices[2]]
	return float64(a.VectorTo(b).Cross(a.VectorTo(c)).Z)
}
