// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"
	"testing"

	"oblikovati.org/renderer"
)

// TestMeshDrawItemsRendersPlacedMesh pins #1773: a placed mesh reference produces a shaded-triangle
// draw item so the viewport can draw it (before this it yielded no body and nothing rendered). The
// tetra STL is 4 facets → 4 flat-shaded triangles, each with its own three vertices and a unit
// face normal.
func TestMeshDrawItemsRendersPlacedMesh(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6)
	pf, err := s.ImportMeshFile(tempSTL(t))
	if err != nil {
		t.Fatalf("ImportMeshFile: %v", err)
	}

	items := s.MeshDrawItems()
	if len(items) != 1 {
		t.Fatalf("MeshDrawItems = %d items, want 1", len(items))
	}
	it := items[0]
	if it.Primitive != renderer.Triangles {
		t.Errorf("primitive = %v, want Triangles", it.Primitive)
	}
	if it.Shading != renderer.ShadeFlat {
		t.Errorf("shading = %v, want ShadeFlat (a reference mesh is always lit)", it.Shading)
	}
	// 4 facets, flat-shaded with unshared vertices: 12 positions / normals / indices.
	if len(it.Positions) != 12 || len(it.Normals) != 12 || len(it.Indices) != 12 {
		t.Fatalf("pos/norm/idx = %d/%d/%d, want 12/12/12", len(it.Positions), len(it.Normals), len(it.Indices))
	}
	for i, n := range it.Normals {
		if l := n.Length(); math.Abs(float64(l)-1) > 1e-6 {
			t.Errorf("normal %d length = %v, want unit", i, l)
		}
	}
	if it.ObjectID != uint64(pf.ID()) {
		t.Errorf("ObjectID = %d, want the feature id %d (for picking)", it.ObjectID, uint64(pf.ID()))
	}
}

// TestMeshDisplaySignatureTracksSet pins the cheap cache key (#1773): the signature is present once a
// mesh is placed and drops to absent when the only mesh is suppressed — so the head's flatten cache
// rebuilds exactly when the visible mesh set changes, not on an orbit.
func TestMeshDisplaySignatureTracksSet(t *testing.T) {
	t.Parallel()
	s, _ := newPartWithBlock(t, 6)
	if _, ok := s.MeshDisplaySignature(); ok {
		t.Fatal("signature should be absent before any mesh is placed")
	}

	pf, err := s.ImportMeshFile(tempSTL(t))
	if err != nil {
		t.Fatalf("ImportMeshFile: %v", err)
	}
	sig, ok := s.MeshDisplaySignature()
	if !ok || sig == "" {
		t.Fatal("signature should be present after placing a mesh")
	}

	pf.SetSuppressed(true)
	if _, ok := s.MeshDisplaySignature(); ok {
		t.Error("signature should be absent when the only mesh is suppressed")
	}
}
