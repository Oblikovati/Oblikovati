//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"testing"
	"unsafe"

	"oblikovati.org/renderer"
)

// swBuildInputFrom converts a renderer.BVH + triangle list into the raw byte layout
// SWScene.Build expects — Go does not reorder struct fields, so a []renderer.BVHNode /
// []renderer.Triangle slice's in-memory bytes already match swtrace.comp's tightly
// packed scalar structs (see swtrace.h's doc comment); this just reinterprets the slice
// headers, no copying.
func swBuildInputFrom(bvh *renderer.BVH, triangles []renderer.Triangle) SWBuildInput {
	nodesBytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(bvh.Nodes))), len(bvh.Nodes)*32)
	triBytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(triangles))), len(triangles)*44)
	return SWBuildInput{Nodes: nodesBytes, TriOrder: bvh.TriangleOrder, Triangles: triBytes}
}

// TestSWSceneMatchesCPUOracle is PBI-334's live hardware cross-check, mirroring PBI-333's
// TestRTSceneMatchesCPUOracle exactly (same quad scene, same oracle) but through the
// software BVH-traversal backend — proving the two Intersector backends agree with each
// other (and the CPU oracle) on hit/t/point/normal/instance id.
func TestSWSceneMatchesCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (SW scene test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewSWScene()
	if err != nil {
		t.Skipf("no compute-capable device available: %v", err)
	}
	defer scene.Destroy()

	triangles := []renderer.Triangle{
		{V0: [3]float32{-0.5, -0.5, 0}, V1: [3]float32{0.5, -0.5, 0}, V2: [3]float32{0.5, 0.5, 0}, InstanceID: 7, PrimitiveID: 0},
		{V0: [3]float32{-0.5, -0.5, 0}, V1: [3]float32{0.5, 0.5, 0}, V2: [3]float32{-0.5, 0.5, 0}, InstanceID: 7, PrimitiveID: 1},
	}
	bvh := renderer.BuildBVH(triangles)
	if err := scene.Build(swBuildInputFrom(bvh, triangles)); err != nil {
		t.Fatalf("Build: %v", err)
	}

	oracle := renderer.NewFakeIntersector(triangles)

	cases := []struct {
		name              string
		origin, direction [3]float32
	}{
		{"straight down through center", [3]float32{0, 0, 2}, [3]float32{0, 0, -1}},
		{"straight down through second triangle", [3]float32{-0.3, 0.3, 2}, [3]float32{0, 0, -1}},
		{"miss", [3]float32{10, 10, 2}, [3]float32{0, 0, -1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ray := renderer.Ray{Origin: c.origin, Direction: c.direction, TMin: 0, TMax: 1e6}
			want := oracle.TraceRay(ray)
			got := scene.Trace(c.origin, c.direction, 0, 1e6)

			if got.Hit != want.Hit {
				t.Fatalf("Hit = %v, want %v", got.Hit, want.Hit)
			}
			if !want.Hit {
				return
			}
			const tol = 1e-3
			if absf(got.T-want.T) > tol {
				t.Errorf("T = %v, want %v", got.T, want.T)
			}
			for i := 0; i < 3; i++ {
				if absf(got.Point[i]-want.Point[i]) > tol {
					t.Errorf("Point[%d] = %v, want %v", i, got.Point[i], want.Point[i])
				}
				if absf(absf(got.Normal[i])-absf(want.Normal[i])) > tol {
					t.Errorf("Normal[%d] magnitude = %v, want %v", i, absf(got.Normal[i]), absf(want.Normal[i]))
				}
			}
			if got.InstanceID != want.InstanceID {
				t.Errorf("InstanceID = %d, want %d", got.InstanceID, want.InstanceID)
			}
		})
	}
}

// TestSWSceneMatchesRTScene is PBI-334's other explicit acceptance criterion in miniature
// (the full "converges to the same image" check is F04's job, PBI-347, once an
// accumulation buffer exists to compare within tolerance) — here, both backends must
// agree EXACTLY on a plain hit-test scene with no accumulation involved, proving the
// backend swap changes only which code answers TraceRay, never the answer itself.
func TestSWSceneMatchesRTScene(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (SW vs RT scene test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	triangles := []renderer.Triangle{
		{V0: [3]float32{-0.5, -0.5, 0}, V1: [3]float32{0.5, -0.5, 0}, V2: [3]float32{0.5, 0.5, 0}, InstanceID: 7, PrimitiveID: 0},
		{V0: [3]float32{-0.5, -0.5, 0}, V1: [3]float32{0.5, 0.5, 0}, V2: [3]float32{-0.5, 0.5, 0}, InstanceID: 7, PrimitiveID: 1},
	}

	rtScene, err := w.NewRTScene()
	if err != nil {
		t.Skipf("no hardware ray tracing available: %v", err)
	}
	defer rtScene.Destroy()
	verts, indices := unitQuadVerticesIndices()
	if err := rtScene.AddMesh(verts, indices, 7); err != nil {
		t.Fatalf("RTScene.AddMesh: %v", err)
	}
	if err := rtScene.Build(); err != nil {
		t.Fatalf("RTScene.Build: %v", err)
	}

	swScene, err := w.NewSWScene()
	if err != nil {
		t.Skipf("no compute-capable device available: %v", err)
	}
	defer swScene.Destroy()
	bvh := renderer.BuildBVH(triangles)
	if err := swScene.Build(swBuildInputFrom(bvh, triangles)); err != nil {
		t.Fatalf("SWScene.Build: %v", err)
	}

	origin, direction := [3]float32{0, 0, 2}, [3]float32{0, 0, -1}
	rtHit := rtScene.Trace(origin, direction, 0, 1e6)
	swHit := swScene.Trace(origin, direction, 0, 1e6)

	if rtHit.Hit != swHit.Hit {
		t.Fatalf("Hit: hardware=%v software=%v, want equal", rtHit.Hit, swHit.Hit)
	}
	const tol = 1e-3
	if absf(rtHit.T-swHit.T) > tol {
		t.Errorf("T: hardware=%v software=%v, want equal within %v", rtHit.T, swHit.T, tol)
	}
	if rtHit.InstanceID != swHit.InstanceID {
		t.Errorf("InstanceID: hardware=%d software=%d, want equal", rtHit.InstanceID, swHit.InstanceID)
	}
}
