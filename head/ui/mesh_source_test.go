//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/head/viewport"
	"oblikovati.org/math"
	"oblikovati.org/renderer"
)

// fakeMeshSource is a stand-in placed-mesh provider that counts how often the (expensive) draw-item
// build runs, so a test can prove the head flattens a mesh once and retains it across frames.
type fakeMeshSource struct {
	sig   string
	items []renderer.DrawItem
	calls int
}

func (f *fakeMeshSource) MeshDisplaySignature() (string, bool) {
	return f.sig, f.sig != ""
}

func (f *fakeMeshSource) MeshDrawItems() []renderer.DrawItem {
	f.calls++
	return f.items
}

func oneTriangleItem() renderer.DrawItem {
	return renderer.DrawItem{
		Primitive: renderer.Triangles,
		Positions: []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		Normals:   []math.Vector3{math.V3(0, 0, 1), math.V3(0, 0, 1), math.V3(0, 0, 1)},
		Indices:   []int{0, 1, 2},
		Shading:   renderer.ShadeFlat,
	}
}

// TestCachedPlacedMeshRetainsAcrossFrames pins #1773: the placed mesh is flattened ONCE and held
// while its set signature is unchanged (an orbit re-calls this every frame), and re-flattened only
// when the mesh set changes — the retention that keeps a dense scan off the per-frame O(n) path.
func TestCachedPlacedMeshRetainsAcrossFrames(t *testing.T) {
	placedMeshCache.sig, placedMeshCache.has, placedMeshCache.mesh = "", false, viewport.Mesh{}
	src := &fakeMeshSource{sig: "m1;", items: []renderer.DrawItem{oneTriangleItem()}}

	m1, k1, has1 := cachedPlacedMesh(src)
	if !has1 || k1 != "m1;" || m1.TriVCount == 0 {
		t.Fatalf("first call: has=%v key=%q triVerts=%d, want true/\"m1;\"/>0", has1, k1, m1.TriVCount)
	}
	if m2, k2, _ := cachedPlacedMesh(src); k2 != k1 || m2.TriVCount != m1.TriVCount {
		t.Errorf("second call diverged from cache: key %q vs %q, triVerts %d vs %d", k2, k1, m2.TriVCount, m1.TriVCount)
	}
	if src.calls != 1 {
		t.Errorf("MeshDrawItems built %d times over two frames, want 1 (retained)", src.calls)
	}

	src.sig = "m1;m2;" // a mesh placed/removed changes the set signature → rebuild
	if _, k3, _ := cachedPlacedMesh(src); k3 != "m1;m2;" || src.calls != 2 {
		t.Errorf("signature change should rebuild: key=%q calls=%d, want \"m1;m2;\"/2", k3, src.calls)
	}
}

// TestCachedPlacedMeshEmptyWhenNoMeshes: with no placed meshes the source reports no signature, so
// there is nothing to draw and no flatten happens.
func TestCachedPlacedMeshEmptyWhenNoMeshes(t *testing.T) {
	placedMeshCache.sig, placedMeshCache.has, placedMeshCache.mesh = "", false, viewport.Mesh{}
	src := &fakeMeshSource{sig: ""}
	if _, _, has := cachedPlacedMesh(src); has {
		t.Error("cachedPlacedMesh should report has=false when there are no placed meshes")
	}
	if src.calls != 0 {
		t.Errorf("MeshDrawItems built %d times with no meshes, want 0", src.calls)
	}
}
