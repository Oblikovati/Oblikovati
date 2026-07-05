// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// meshBoundsCenterAbove returns a ray origin directly above the mesh's bounding-box centre and a
// straight-down direction — robust to whatever scale the STL imported at.
func meshBoundsCenterAbove(g *feature.MeshGeometry) (math.Point3, math.Vector3) {
	box := math.EmptyBox()
	for _, v := range g.Vertices {
		box = box.ExtendPoint(v)
	}
	c := box.Center()
	return math.P3(c.X, c.Y, box.Max.Z+10), math.V3(0, 0, -1)
}

// TestRayPickerHitsPlacedMeshFacet pins #1776: a ray onto a placed mesh returns a MeshFaceHandle for
// the owning mesh, a ray beside it misses, and a filter that excludes mesh faces returns nothing.
func TestRayPickerHitsPlacedMeshFacet(t *testing.T) {
	s, _ := newPartWithBlock(t, 6)
	if _, err := s.ImportMeshFile(tempSTL(t)); err != nil {
		t.Fatalf("ImportMeshFile: %v", err)
	}
	meshes := s.PickableMeshes()
	if len(meshes) != 1 {
		t.Fatalf("PickableMeshes = %d, want 1", len(meshes))
	}
	p := NewRayPicker(s.Camera(), func() []*topo.Body { return nil }).
		WithMeshes(func() []*feature.MeshFeature { return meshes })

	origin, dir := meshBoundsCenterAbove(meshes[0].Geometry())
	sel, _, ok := p.nearestMeshFacet(origin, dir, NewSelectionFilter())
	if !ok {
		t.Fatal("a ray straight down onto the mesh should hit a facet")
	}
	h, isMesh := sel.(MeshFaceHandle)
	if !isMesh {
		t.Fatalf("pick = %T, want MeshFaceHandle", sel)
	}
	if h.Mesh != meshes[0] {
		t.Error("handle points at the wrong mesh")
	}
	if len(h.Face().VertexIndices()) < 3 {
		t.Error("picked facet should have at least 3 vertices")
	}

	if _, _, miss := p.nearestMeshFacet(math.P3(origin.X+1e6, origin.Y, origin.Z), dir, NewSelectionFilter()); miss {
		t.Error("a ray far beside the mesh should miss")
	}
	if _, _, blocked := p.nearestMeshFacet(origin, dir, NewSelectionFilter(SelectFace)); blocked {
		t.Error("a face-only filter must not return a mesh facet")
	}
}

// TestRayPickerMeshIndexIsCached: the per-mesh BVH is built once and reused across picks (the
// hover-safe retention), so a second pick returns the same cached entry.
func TestRayPickerMeshIndexIsCached(t *testing.T) {
	s, _ := newPartWithBlock(t, 6)
	if _, err := s.ImportMeshFile(tempSTL(t)); err != nil {
		t.Fatalf("ImportMeshFile: %v", err)
	}
	g := s.PickableMeshes()[0].Geometry()
	p := NewRayPicker(s.Camera(), func() []*topo.Body { return nil })
	first := p.meshRayEntry(g)
	if first.index == nil {
		t.Fatal("expected a built ray index")
	}
	if second := p.meshRayEntry(g); second != first {
		t.Error("meshRayEntry rebuilt the index instead of returning the cached one")
	}
}
