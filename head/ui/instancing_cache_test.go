//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// These pin the render-instancing caching/indexing so the orbit-perf and dedup wins can't regress:
// the per-source mesh cache (no per-frame re-tessellation), its key (camera-independent but
// style/selection sensitive), the instance bounds, and the end-to-end one-mesh-many-instances dedup.

func instCamera() scene.Camera {
	return scene.Camera{Eye: math.P3(40, 40, 40), Target: math.P3(0, 0, 0), Up: math.V3(0, 0, 1), FOV: 0.8, Width: 800, Height: 600}
}

func nullSurface(*topo.Body) renderer.Surface { return renderer.Surface{} }

func clearSourceMeshCache() {
	for k := range sourceMeshCache {
		delete(sourceMeshCache, k)
	}
}

// TestCachedSourceMeshReusesOnKey is the orbit fix's guard: with the key unchanged, cachedSourceMesh
// returns the SAME backing arrays (no re-tessellation/re-flatten) — the difference between ~0.18 ms
// and ~57 ms per frame on a 1024-copy assembly. A key change (geometry/style/selection) rebuilds.
func TestCachedSourceMeshReusesOnKey(t *testing.T) {
	clearSourceMeshCache()
	s := assemblyWithPlacedBox(t)
	groups := s.VisibleInstances()
	if len(groups) == 0 {
		t.Fatal("no instance groups for the placed box")
	}
	src := groups[0].Source

	m1 := cachedSourceMesh(src, instCamera(), nullSurface, renderer.ShadedWithEdges, nil, "k1")
	m2 := cachedSourceMesh(src, instCamera(), nullSurface, renderer.ShadedWithEdges, nil, "k1")
	if len(m1.TriVerts) == 0 {
		t.Fatal("flattened source has no triangle vertices")
	}
	if &m1.TriVerts[0] != &m2.TriVerts[0] {
		t.Error("cachedSourceMesh re-flattened on an unchanged key — orbit would re-tessellate every frame")
	}
	m3 := cachedSourceMesh(src, instCamera(), nullSurface, renderer.ShadedWithEdges, nil, "k2")
	if len(m3.TriVerts) > 0 && &m3.TriVerts[0] == &m1.TriVerts[0] {
		t.Error("cachedSourceMesh did not rebuild on a changed key (stale geometry/style/selection)")
	}
}

// TestInstancedSourceKeyCameraIndependentStyleSensitive: the key must be stable across frames (so
// the cache holds while orbiting — it takes no camera) but change when the visual style changes (a
// different style flattens differently), or the source cache would serve a stale mesh.
func TestInstancedSourceKeyCameraIndependentStyleSensitive(t *testing.T) {
	s := assemblyWithPlacedBox(t)
	k := instancedSourceKey(s)
	if k == "" {
		t.Fatal("instancedSourceKey is empty — the source cache would be disabled")
	}
	if again := instancedSourceKey(s); again != k {
		t.Errorf("instancedSourceKey unstable between identical frames: %q then %q", k, again)
	}
	s.SetVisualStyle(renderer.Wireframe)
	if instancedSourceKey(s) == k {
		t.Error("instancedSourceKey did not change when the visual style changed")
	}
}

// TestInstancedBoundsTransformsSources: bounds come from the source range box transformed by the
// occurrence matrices (no tessellation). The fixture's box is placed at +10 on X, so the world
// bounds sit around there — proving framing scales O(instances) without a world mesh.
func TestInstancedBoundsTransformsSources(t *testing.T) {
	s := assemblyWithPlacedBox(t)
	mn, mx, ok := instancedBounds(s.VisibleInstances())
	if !ok {
		t.Fatal("instancedBounds returned ok=false for a placed component")
	}
	if mn[0] < 6 || mn[0] > 10 || mx[0] < 10 || mx[0] > 14 { // box (±2) placed at +10 → ~[8,12]
		t.Errorf("instanced bounds X = [%g,%g], want ~[8,12] (box placed at +10)", mn[0], mx[0])
	}
}

// TestBuildInstancedFrameDedupes is the end-to-end head-layer guard: several copies of one component
// produce ONE source mesh (not N) with the tri record's instanceCount = N and N instance matrices —
// the whole point of instancing (upload one mesh, draw it many times).
func TestBuildInstancedFrameDedupes(t *testing.T) {
	clearSourceMeshCache()
	s := assemblyWithPlacedBox(t) // one box placed
	asm := s.ActiveDocument().Content().(*compdef.AssemblyComponentDefinition)
	boxDoc := s.Workspace().Documents()[0]
	for i := 2; i <= 4; i++ { // three more copies of the SAME component
		if _, err := asm.PlaceComponentFromFile(s.ActiveDocument(), boxDoc,
			fmt.Sprintf("box:%d", i), math.Translation4(math.V3(math.Scalar(10*i), 0, 0))); err != nil {
			t.Fatalf("place copy %d: %v", i, err)
		}
	}

	groups := s.VisibleInstances()
	if len(groups) != 1 {
		t.Fatalf("VisibleInstances = %d groups, want 1 (copies share one mesh)", len(groups))
	}
	n := len(groups[0].Transforms)
	m, mats, recs, _, ok := buildInstancedFrame(groups, groups, renderer.DrawList{}, instCamera(), nullSurface, renderer.Shaded, nil, "k")
	if !ok {
		t.Fatal("buildInstancedFrame returned ok=false")
	}
	if got := len(mats) / 16; got != n {
		t.Errorf("instance matrices = %d, want %d (one per occurrence)", got, n)
	}
	// One box is 12 triangles = 36 indexed verts at most; N copies must NOT inflate the mesh.
	if m.TriVCount > 100 {
		t.Errorf("merged mesh has %d tri verts — the source was duplicated instead of instanced", m.TriVCount)
	}
	var maxInst int32
	for i := 0; i < len(recs); i += 7 {
		if recs[i+5] > maxInst {
			maxInst = recs[i+5]
		}
	}
	if int(maxInst) != n {
		t.Errorf("max draw-record instanceCount = %d, want %d (all copies in one instanced draw)", maxInst, n)
	}
}
