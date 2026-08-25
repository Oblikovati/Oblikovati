//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"math"
	"testing"

	"oblikovati.org/kernel/shading/openpbr"
)

// TestRTScenePipelineMatchesCPUOracle is PBI-345's explicit acceptance criterion: a
// rendered single-bounce scene (one light, one flat OpenPBR surface — a plane, not a
// tessellated sphere, so both the GPU and the CPU oracle share an EXACT analytic hit
// point/normal with no tessellation-approximation error to account for) must match the
// F03 CPU oracle's radiance value at the sampled pixel within tolerance.
func TestRTScenePipelineMatchesCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (RT pipeline test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewRTScene()
	if err != nil {
		t.Skipf("no hardware ray tracing available: %v", err)
	}
	defer scene.Destroy()

	// A large quad in the z=0 plane (well beyond the camera ray's landing point at the
	// origin), same winding/normal convention as raytrace_test.go's unit quad.
	verts := []float32{
		-5, -5, 0,
		5, -5, 0,
		5, 5, 0,
		-5, 5, 0,
	}
	indices := []uint32{0, 1, 2, 0, 2, 3}
	if err := scene.AddMesh(verts, indices, 1); err != nil {
		t.Fatalf("AddMesh: %v", err)
	}
	if err := scene.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := scene.BuildPipeline(pathtraceRgenSPV, pathtraceRmissSPV, shadowRmissSPV, pathtraceRchitSPV); err != nil {
		t.Skipf("no hardware RT pipeline (VK_KHR_ray_tracing_pipeline) available: %v", err)
	}

	origin := [3]float32{0, 0, 2}
	direction := [3]float32{0, 0, -1}
	params := PipelineParams{
		LightPos:          [3]float32{2, 2, 3},
		LightIntensity:    50,
		LightColor:        [3]float32{1, 1, 1},
		BaseColor:         [3]float32{0.8, 0.2, 0.2},
		BaseWeight:        1,
		SpecularRoughness: 0.3,
		SpecularIOR:       1.5,
	}
	gotR, gotG, gotB := scene.TracePipeline(origin, direction, 0, 1e6, params)

	wantR, wantG, wantB := cpuOracleRadiance(origin, direction, params)

	const tol = 0.02 // absolute radiance tolerance — both sides share the exact analytic
	// hit point/normal (a flat plane, not a tessellated sphere), so this is a tight
	// numerical cross-check, not a perceptual/image-similarity one.
	if absf(gotR-wantR) > tol || absf(gotG-wantG) > tol || absf(gotB-wantB) > tol {
		t.Errorf("GPU radiance = (%v,%v,%v), want CPU oracle (%v,%v,%v) within %v",
			gotR, gotG, gotB, wantR, wantG, wantB, tol)
	}
}

// cpuOracleRadiance reproduces pathtrace.rchit's math exactly in Go, using the same
// kernel/shading/openpbr functions the GLSL was ported from (diffuse.go/specular.go/
// ggx.go/fresnel.go) — the F03 CPU oracle PBI-345's acceptance criterion names.
func cpuOracleRadiance(origin, direction [3]float32, p PipelineParams) (r, g, b float32) {
	// The camera ray (0,0,2)→(0,0,-1) hits the z=0 quad at the origin with normal +z —
	// computed directly (not re-deriving ray-plane intersection) since this oracle's job
	// is to cross-check the SHADING math, not re-implement ray-triangle intersection
	// (already covered by TestRTSceneMatchesCPUOracle, PBI-333).
	hitPoint := openpbr.Vec3{X: 0, Y: 0, Z: 0}

	lightPos := openpbr.Vec3{X: float64(p.LightPos[0]), Y: float64(p.LightPos[1]), Z: float64(p.LightPos[2])}
	toLight := lightPos.Add(hitPoint.Scale(-1))
	dist := math.Sqrt(toLight.Dot(toLight))
	wi := toLight.Scale(1 / dist)
	wo := openpbr.Vec3{X: -float64(direction[0]), Y: -float64(direction[1]), Z: -float64(direction[2])}

	// The local frame equals the world frame here: buildBasis(n=(0,0,1)) in the shader
	// picks tangent=(1,0,0), bitangent=(0,1,0) — i.e. wi/wo need no transform.
	if wi.Z <= 0 || wo.Z <= 0 {
		return 0, 0, 0
	}

	rho := openpbr.Color3{R: float64(p.BaseColor[0]) * float64(p.BaseWeight),
		G: float64(p.BaseColor[1]) * float64(p.BaseWeight), B: float64(p.BaseColor[2]) * float64(p.BaseWeight)}
	diffuse := openpbr.DiffuseEON(rho, 0, wi, wo) // base_diffuse_roughness fixed at 0, matching the shader

	alpha := openpbr.AlphaFromRoughness(float64(p.SpecularRoughness))
	h := wi.Add(wo).Normalize()
	d := openpbr.DistributionGGX(h, alpha)
	g2 := openpbr.SmithG2(wi, wo, alpha)
	fr := openpbr.DielectricFresnel(float64(p.SpecularIOR), math.Max(wi.Dot(h), 0))
	specular := fr * d * g2 / (4 * wi.Z * wo.Z)

	attenuation := float64(p.LightIntensity) / (dist * dist)
	cosTheta := wi.Z
	rF := (diffuse.R + specular) * float64(p.LightColor[0]) * attenuation * cosTheta
	gF := (diffuse.G + specular) * float64(p.LightColor[1]) * attenuation * cosTheta
	bF := (diffuse.B + specular) * float64(p.LightColor[2]) * attenuation * cosTheta
	return float32(rF), float32(gF), float32(bF)
}
