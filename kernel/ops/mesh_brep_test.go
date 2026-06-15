// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// unitTetraMesh returns the four corner vertices of a unit tetrahedron and outward-wound
// triangular facets (acceptance fixture for #492).
func unitTetraMesh() ([]math.Point3, [][]int) {
	verts := []math.Point3{
		math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 1),
	}
	facets := [][]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}}
	return verts, facets
}

// TestMeshToBRepTetraSolid is the #492 acceptance: a tetra mesh converts to a validated
// 4-face solid with the analytic volume 1/6.
func TestMeshToBRepTetraSolid(t *testing.T) {
	verts, facets := unitTetraMesh()
	body := MeshToBRep(verts, facets, "mesh")
	if body == nil {
		t.Fatal("MeshToBRep returned nil for a tetra mesh")
	}
	if r := Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("tetra B-rep not a valid solid: valid=%v solid=%v problems=%v", r.Valid, body.IsSolid(), r.Issues)
	}
	if n := len(body.Faces()); n != 4 {
		t.Errorf("face count = %d, want 4", n)
	}
	if v := BodyGeometryProperties(body, DefaultQuality()).Volume; stdmath.Abs(v-1.0/6) > 1e-9 {
		t.Errorf("volume = %v, want %v", v, 1.0/6)
	}
}

// TestMeshToBRepReorientsInwardMesh: an inward-wound (negative-volume) but consistent mesh
// is re-oriented to a positive-volume outward solid rather than rejected.
func TestMeshToBRepReorientsInwardMesh(t *testing.T) {
	verts, facets := unitTetraMesh()
	for _, f := range facets { // flip every facet ⇒ inward winding
		f[1], f[2] = f[2], f[1]
	}
	body := MeshToBRep(verts, facets, "mesh")
	if body == nil {
		t.Fatal("MeshToBRep returned nil for an inward-wound tetra")
	}
	if v := BodyGeometryProperties(body, DefaultQuality()).Volume; v <= 0 {
		t.Errorf("volume = %v, want positive after re-orientation", v)
	}
}

// TestMeshToBRepEmpty: no facets ⇒ nil (nothing to build).
func TestMeshToBRepEmpty(t *testing.T) {
	if b := MeshToBRep(nil, nil, "mesh"); b != nil {
		t.Errorf("MeshToBRep(empty) = %v, want nil", b)
	}
}

// TestMeshToBRepQuadFacet: a quad facet fan-triangulates; a unit cube (6 quad facets)
// becomes a valid solid of volume 1.
func TestMeshToBRepQuadFacet(t *testing.T) {
	verts := []math.Point3{
		math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0),
		math.P3(0, 0, 1), math.P3(1, 0, 1), math.P3(1, 1, 1), math.P3(0, 1, 1),
	}
	facets := [][]int{
		{0, 3, 2, 1}, {4, 5, 6, 7}, {0, 1, 5, 4}, {2, 3, 7, 6}, {1, 2, 6, 5}, {0, 4, 7, 3},
	}
	body := MeshToBRep(verts, facets, "mesh")
	if body == nil {
		t.Fatal("MeshToBRep returned nil for a cube mesh")
	}
	if r := Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("cube B-rep not a valid solid: %v", r.Issues)
	}
	if v := BodyGeometryProperties(body, DefaultQuality()).Volume; stdmath.Abs(v-1.0) > 1e-9 {
		t.Errorf("volume = %v, want 1", v)
	}
}
