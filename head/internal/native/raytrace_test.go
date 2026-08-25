//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"testing"

	"oblikovati.org/renderer"
)

// unitQuadVerticesIndices is the same unit-square scene renderer/rttest uses (a
// two-triangle quad in the z=0 plane spanning [-0.5,0.5]^2), as flat vertex/index
// buffers for RTScene.AddMesh.
func unitQuadVerticesIndices() ([]float32, []uint32) {
	verts := []float32{
		-0.5, -0.5, 0, // 0
		0.5, -0.5, 0, // 1
		0.5, 0.5, 0, // 2
		-0.5, 0.5, 0, // 3
	}
	indices := []uint32{0, 1, 2, 0, 2, 3}
	return verts, indices
}

// TestRTSceneMatchesCPUOracle is PBI-333's live hardware cross-check: a straight-down
// ray traced through the real BLAS/TLAS/ray-query pipeline must agree with
// renderer.FakeIntersector (the CPU-reference oracle both PBI-332 and this backend are
// held to) on hit/t/point/normal/instance id, for the same scene.
func TestRTSceneMatchesCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (RT scene test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewRTScene()
	if err != nil {
		t.Skipf("no hardware ray tracing available: %v", err)
	}
	defer scene.Destroy()

	verts, indices := unitQuadVerticesIndices()
	if err := scene.AddMesh(verts, indices, 7); err != nil {
		t.Fatalf("AddMesh: %v", err)
	}
	if err := scene.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}

	oracle := renderer.NewFakeIntersector([]renderer.Triangle{
		{V0: [3]float32{-0.5, -0.5, 0}, V1: [3]float32{0.5, -0.5, 0}, V2: [3]float32{0.5, 0.5, 0}, InstanceID: 7, PrimitiveID: 0},
		{V0: [3]float32{-0.5, -0.5, 0}, V1: [3]float32{0.5, 0.5, 0}, V2: [3]float32{-0.5, 0.5, 0}, InstanceID: 7, PrimitiveID: 1},
	})

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
					// Winding of the two GLSL triangles may face either way relative to
					// the CPU oracle's Möller-Trumbore convention; compare magnitude per
					// axis (both must lie along the same ±z axis for this planar quad).
					t.Errorf("Normal[%d] magnitude = %v, want %v", i, absf(got.Normal[i]), absf(want.Normal[i]))
				}
			}
			if got.InstanceID != want.InstanceID {
				t.Errorf("InstanceID = %d, want %d", got.InstanceID, want.InstanceID)
			}
		})
	}
}

func absf(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// TestRTSceneAcrossSeparateWindowsInOneProcess is a regression test for a bug found
// live while adding M45-F05 PBI-350: the extension function pointers
// (vkGetAccelerationStructureBuildSizesKHR etc.) were resolved once into a
// process-global cache keyed off the FIRST device ever seen, then reused unchanged by
// every later RTScene — including one created on a SECOND, different Vulkan device (a
// second window). A device-level function pointer from vkGetDeviceProcAddr is valid
// only for the device it was retrieved from (Vulkan spec); reusing one against a
// different device is undefined behavior, and manifested here as an intermittent
// SIGSEGV inside obk_rt_scene_add_mesh (~50-65% of runs). Fixed by resolving the
// function table fresh per RTScene (RTScene::rtFn) instead of caching it globally —
// this test creates and fully exercises two RTScenes on two separate windows in strict
// sequence, which reproduced the crash before the fix.
func TestRTSceneAcrossSeparateWindowsInOneProcess(t *testing.T) {
	verts, indices := unitQuadVerticesIndices()

	for i := 0; i < 2; i++ {
		w, err := CreateWindow(64, 64, "Oblikovati (RT cross-window regression test)")
		if err != nil {
			t.Skipf("no window/GPU available: %v", err)
		}
		scene, err := w.NewRTScene()
		if err != nil {
			w.Destroy()
			t.Skipf("no hardware ray tracing available: %v", err)
		}
		if err := scene.AddMesh(verts, indices, uint32(i+1)); err != nil {
			t.Fatalf("window %d: AddMesh: %v", i, err)
		}
		if err := scene.Build(); err != nil {
			t.Fatalf("window %d: Build: %v", i, err)
		}
		hit := scene.Trace([3]float32{0, 0, 2}, [3]float32{0, 0, -1}, 0, 1e6)
		if !hit.Hit {
			t.Errorf("window %d: expected a hit through the quad's center, got a miss", i)
		}
		scene.Destroy()
		w.Destroy()
	}
}
