//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"testing"

	"oblikovati.org/renderer"
	"oblikovati.org/renderer/rttest"
)

// rtSceneIntersector adapts *RTScene to renderer.Intersector, so the exact same shared
// contract-test suite (renderer/rttest) that exercises renderer.FakeIntersector can run
// against the real hardware backend — PBI-333's own acceptance criterion, and the one
// PBI-334 explicitly says to repeat identically for the software backend below.
type rtSceneIntersector struct{ scene *RTScene }

func (a rtSceneIntersector) TraceRay(ray renderer.Ray) renderer.Hit {
	h := a.scene.Trace(ray.Origin, ray.Direction, ray.TMin, ray.TMax)
	return renderer.Hit{
		Hit: h.Hit, T: h.T, Point: h.Point, Normal: h.Normal,
		InstanceID: h.InstanceID, PrimitiveID: h.PrimitiveID,
	}
}

// TestRTSceneSatisfiesIntersectorContract runs renderer/rttest's shared suite (straight
// hit, miss, TMin/TMax clipping, nearest-of-two) against the real hardware backend.
func TestRTSceneSatisfiesIntersectorContract(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (RT contract test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	// The factory below is called from INSIDE each rttest subtest, so it only has this
	// outer t, not the subtest's own — t.Cleanup here would register on the outer test
	// and run only once, at the SAME point a defer would, with no guaranteed order
	// against a plain defer registered earlier. Track scenes explicitly and destroy
	// them before the window: defers unwind LIFO, so the window's defer must be
	// registered FIRST (runs last) and the scene-cleanup defer registered SECOND (runs
	// first, while the device is still alive).
	defer w.Destroy()
	var scenes []*RTScene
	defer func() {
		for _, s := range scenes {
			s.Destroy()
		}
	}()

	rttest.RunIntersectorContractTests(t, func(triangles []renderer.Triangle) renderer.Intersector {
		scene, err := w.NewRTScene()
		if err != nil {
			t.Skipf("no hardware ray tracing available: %v", err)
		}
		scenes = append(scenes, scene)
		byInstance := map[uint32][]renderer.Triangle{}
		for _, tri := range triangles {
			byInstance[tri.InstanceID] = append(byInstance[tri.InstanceID], tri)
		}
		for instanceID, tris := range byInstance {
			verts, indices := trianglesToMesh(tris)
			if err := scene.AddMesh(verts, indices, instanceID); err != nil {
				t.Fatalf("AddMesh: %v", err)
			}
		}
		if err := scene.Build(); err != nil {
			t.Fatalf("Build: %v", err)
		}
		return rtSceneIntersector{scene}
	})
}

// swSceneIntersector adapts *SWScene to renderer.Intersector, mirroring
// rtSceneIntersector for the software BVH-traversal backend.
type swSceneIntersector struct{ scene *SWScene }

func (a swSceneIntersector) TraceRay(ray renderer.Ray) renderer.Hit {
	h := a.scene.Trace(ray.Origin, ray.Direction, ray.TMin, ray.TMax)
	return renderer.Hit{
		Hit: h.Hit, T: h.T, Point: h.Point, Normal: h.Normal,
		InstanceID: h.InstanceID, PrimitiveID: h.PrimitiveID,
	}
}

// TestSWSceneSatisfiesIntersectorContract is PBI-334's explicit acceptance criterion:
// "the identical Intersector contract test suite from PBI-333 passes against this
// backend" — the same rttest.RunIntersectorContractTests call as
// TestRTSceneSatisfiesIntersectorContract, against the software backend instead.
func TestSWSceneSatisfiesIntersectorContract(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (SW contract test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	// See TestRTSceneSatisfiesIntersectorContract's comment: scenes must be destroyed
	// before the window, and a defer registered first runs last.
	defer w.Destroy()
	var scenes []*SWScene
	defer func() {
		for _, s := range scenes {
			s.Destroy()
		}
	}()

	rttest.RunIntersectorContractTests(t, func(triangles []renderer.Triangle) renderer.Intersector {
		scene, err := w.NewSWScene()
		if err != nil {
			t.Skipf("no compute-capable device available: %v", err)
		}
		scenes = append(scenes, scene)
		bvh := renderer.BuildBVH(triangles)
		if err := scene.Build(swBuildInputFrom(bvh, triangles)); err != nil {
			t.Fatalf("Build: %v", err)
		}
		return swSceneIntersector{scene}
	})
}

// trianglesToMesh flattens a same-instance triangle group into RTScene.AddMesh's flat
// vertex/index form (each triangle's 3 vertices are appended, unindexed — simple and
// correct for the small test scenes this contract suite uses).
func trianglesToMesh(triangles []renderer.Triangle) (vertices []float32, indices []uint32) {
	for i, tri := range triangles {
		for _, v := range [3][3]float32{tri.V0, tri.V1, tri.V2} {
			vertices = append(vertices, v[0], v[1], v[2])
		}
		base := uint32(i * 3)
		indices = append(indices, base, base+1, base+2)
	}
	return vertices, indices
}
