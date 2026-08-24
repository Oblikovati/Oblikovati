//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"testing"

	"oblikovati.org/renderer"
)

// pathtraceTestScene is the shared quad/light/material fixture both
// TestRTScenePipelineMatchesCPUOracle and this file's tests use — kept in one place so
// the hardware and software backends are provably tested against the IDENTICAL scene.
func pathtraceTestScene() (verts []float32, indices []uint32, origin, direction [3]float32, params PipelineParams) {
	verts = []float32{-5, -5, 0, 5, -5, 0, 5, 5, 0, -5, 5, 0}
	indices = []uint32{0, 1, 2, 0, 2, 3}
	origin = [3]float32{0, 0, 2}
	direction = [3]float32{0, 0, -1}
	params = PipelineParams{
		LightPos:          [3]float32{2, 2, 3},
		LightIntensity:    50,
		LightColor:        [3]float32{1, 1, 1},
		BaseColor:         [3]float32{0.8, 0.2, 0.2},
		BaseWeight:        1,
		SpecularRoughness: 0.3,
		SpecularIOR:       1.5,
	}
	return verts, indices, origin, direction, params
}

// TestSWScenePathtraceMatchesCPUOracle is PBI-346's explicit acceptance criterion: the
// same single-bounce CPU-oracle cross-check as PBI-345's
// TestRTScenePipelineMatchesCPUOracle, run against the software (compute-BVH) backend.
func TestSWScenePathtraceMatchesCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (SW pathtrace test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewSWScene()
	if err != nil {
		t.Skipf("no compute-capable device available: %v", err)
	}
	defer scene.Destroy()

	verts, indices, origin, direction, params := pathtraceTestScene()
	triangles := make([]renderer.Triangle, len(indices)/3)
	for i := range triangles {
		v := func(j int) [3]float32 {
			k := indices[i*3+j] * 3
			return [3]float32{verts[k], verts[k+1], verts[k+2]}
		}
		triangles[i] = renderer.Triangle{V0: v(0), V1: v(1), V2: v(2), InstanceID: 1, PrimitiveID: uint32(i)}
	}
	bvh := renderer.BuildBVH(triangles)
	if err := scene.Build(swBuildInputFrom(bvh, triangles)); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := scene.BuildPathtracePipeline(swpathtraceCompSPV); err != nil {
		t.Fatalf("BuildPathtracePipeline: %v", err)
	}

	gotR, gotG, gotB := scene.TracePathtrace(origin, direction, 0, 1e6, params)
	wantR, wantG, wantB := cpuOracleRadiance(origin, direction, params)

	const tol = 0.02
	if absf(gotR-wantR) > tol || absf(gotG-wantG) > tol || absf(gotB-wantB) > tol {
		t.Errorf("software backend radiance = (%v,%v,%v), want CPU oracle (%v,%v,%v) within %v",
			gotR, gotG, gotB, wantR, wantG, wantB, tol)
	}
}

// TestPathtraceBackendParity is PBI-346's other explicit acceptance criterion: an
// identical scene rendered with the hardware-RT checkbox on (PBI-345's RT pipeline) vs
// off (this PBI's compute-BVH pipeline) must match within tolerance — the same
// "backend swap changes only speed, not fidelity" property PBI-334 already proved at the
// raw-Intersector level, now proved at the shading-integrator level.
func TestPathtraceBackendParity(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (pathtrace backend parity test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	verts, indices, origin, direction, params := pathtraceTestScene()

	rtScene, err := w.NewRTScene()
	if err != nil {
		t.Skipf("no hardware ray tracing available: %v", err)
	}
	defer rtScene.Destroy()
	if err := rtScene.AddMesh(verts, indices, 1); err != nil {
		t.Fatalf("RTScene.AddMesh: %v", err)
	}
	if err := rtScene.Build(); err != nil {
		t.Fatalf("RTScene.Build: %v", err)
	}
	if err := rtScene.BuildPipeline(pathtraceRgenSPV, pathtraceRmissSPV, shadowRmissSPV, pathtraceRchitSPV); err != nil {
		t.Skipf("no hardware RT pipeline available: %v", err)
	}

	swScene, err := w.NewSWScene()
	if err != nil {
		t.Skipf("no compute-capable device available: %v", err)
	}
	defer swScene.Destroy()
	triangles := []renderer.Triangle{
		{V0: [3]float32{-5, -5, 0}, V1: [3]float32{5, -5, 0}, V2: [3]float32{5, 5, 0}, InstanceID: 1, PrimitiveID: 0},
		{V0: [3]float32{-5, -5, 0}, V1: [3]float32{5, 5, 0}, V2: [3]float32{-5, 5, 0}, InstanceID: 1, PrimitiveID: 1},
	}
	bvh := renderer.BuildBVH(triangles)
	if err := swScene.Build(swBuildInputFrom(bvh, triangles)); err != nil {
		t.Fatalf("SWScene.Build: %v", err)
	}
	if err := swScene.BuildPathtracePipeline(swpathtraceCompSPV); err != nil {
		t.Fatalf("SWScene.BuildPathtracePipeline: %v", err)
	}

	rtR, rtG, rtB := rtScene.TracePipeline(origin, direction, 0, 1e6, params)
	swR, swG, swB := swScene.TracePathtrace(origin, direction, 0, 1e6, params)

	const tol = 0.02
	if absf(rtR-swR) > tol || absf(rtG-swG) > tol || absf(rtB-swB) > tol {
		t.Errorf("hardware=(%v,%v,%v) software=(%v,%v,%v), want equal within %v", rtR, rtG, rtB, swR, swG, swB, tol)
	}
}
