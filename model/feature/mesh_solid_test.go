// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// tetraMeshGeometry is a unit tetrahedron with outward-wound triangular facets.
func tetraMeshGeometry() *MeshGeometry {
	return &MeshGeometry{
		Vertices: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 1)},
		Facets:   [][]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}},
	}
}

// TestMeshSolidConvertsTetra is the #492 acceptance: a tetra mesh converts to a validated
// 4-face solid body.
func TestMeshSolidConvertsTetra(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	ms := NewMeshFeatures(fs).AddSolid(tetraMeshGeometry())
	fs.Recompute()
	if !ms.Health().OK() {
		t.Fatalf("mesh-solid sick: %+v", ms.Health())
	}
	bodies := fs.Result()
	if len(bodies) != 1 {
		t.Fatalf("mesh-solid result = %d bodies, want 1", len(bodies))
	}
	solid := bodies[0]
	if r := ops.Validate(solid); !r.Valid || !solid.IsSolid() {
		t.Fatalf("converted body not a valid solid: %+v", r)
	}
	if n := len(solid.Faces()); n != 4 {
		t.Errorf("face count = %d, want 4", n)
	}
	if v := query.BodyGeometryProperties(solid, ops.DefaultQuality()).Volume; stdmath.Abs(v-1.0/6) > 1e-9 {
		t.Errorf("volume = %v, want %v", v, 1.0/6)
	}
}

// TestPresentationMeshPassesBodyThrough: the presentation MeshFeature carries the mesh without
// altering the running body (#492 acceptance bullet 2).
func TestPresentationMeshPassesBodyThrough(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(math.P2(0, 0), 5)
	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 10 })
	fs.Recompute()
	before := fs.Result()[0]

	mf := NewMeshFeatures(fs).Add(tetraMeshGeometry())
	fs.Recompute()
	if !mf.Health().OK() {
		t.Fatalf("presentation mesh sick: %+v", mf.Health())
	}
	after := fs.Result()
	if len(after) != 1 || after[0] != before {
		t.Errorf("presentation mesh altered the running body: %d bodies (want 1, same pointer)", len(after))
	}
	if mf.Definition().(*MeshFeature).Geometry().Facets == nil {
		t.Error("presentation mesh dropped its geometry")
	}
}

// TestMeshSolidRoundTrip checks the mesh-solid feature (with its inline mesh) survives an
// .obk round trip and rebuilds the same solid (#492 acceptance: round-trip).
func TestMeshSolidRoundTrip(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewMeshFeatures(fs).AddSolid(tetraMeshGeometry())

	data, err := fs.MarshalRecipe(oneSketch{})
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	fresh := NewPartFeatures(nil)
	if err := fresh.ApplyRecipe(data, oneSketch{}, nil); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	g := fresh.Item(0).Definition().(*MeshSolidFeature).Geometry()
	if len(g.Vertices) != 4 || len(g.Facets) != 4 {
		t.Fatalf("restored mesh = %d verts, %d facets; want 4, 4", len(g.Vertices), len(g.Facets))
	}
	fresh.Recompute()
	solid := fresh.Result()[0]
	if r := ops.Validate(solid); !r.Valid || !solid.IsSolid() {
		t.Fatalf("restored body not a valid solid: %+v", r)
	}
	if v := query.BodyGeometryProperties(solid, ops.DefaultQuality()).Volume; stdmath.Abs(v-1.0/6) > 1e-9 {
		t.Errorf("restored volume = %v, want %v", v, 1.0/6)
	}
}
