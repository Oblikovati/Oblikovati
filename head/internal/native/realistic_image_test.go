//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"math"
	"testing"

	"oblikovati.org/kernel/shading/openpbr"
	"oblikovati.org/renderer"
)

// realisticTestScene is the fixture both the hardware and software per-pixel image
// tests use: the same 10x10 quad at z=0 used throughout M45's tests, and a camera
// pulled back far enough (tanHalfFovY covers a wider frustum than the quad) that a
// rendered image has BOTH hit pixels (near center) and miss pixels (near the
// border) — unlike the single-ray harness tests, which only ever probe one point.
// The quad's two triangles share a diagonal seam at y=x (see realisticTestScene's
// triangle list); the eye is nudged off dead-center (rather than (0,0,10)) so no
// per-pixel ray in a square image lands EXACTLY on that seam, which would trip #2146
// (a float32 precision crack in the software backend's non-watertight ray-triangle
// test — a real, separately-tracked bug, not something this fixture should mask by
// coincidence rather than by design).
func realisticTestScene() (verts []float32, indices []uint32, cam CameraBasis, params RealisticLightParams) {
	verts = []float32{-5, -5, 0, 5, -5, 0, 5, 5, 0, -5, 5, 0}
	indices = []uint32{0, 1, 2, 0, 2, 3}
	cam = CameraBasis{
		Eye: [3]float32{0.37, 0, 10}, TMin: 0,
		Forward: [3]float32{0, 0, -1}, TMax: 1e6,
		Right: [3]float32{1, 0, 0}, TanHalfFovY: 0.6,
		Up: [3]float32{0, 1, 0}, Aspect: 1,
	}
	params = RealisticLightParams{
		LightDirection:    [3]float32{0.3, 0.5, 0.8},
		LightIntensity:    2,
		LightColor:        [3]float32{1, 1, 1},
		BaseColor:         [3]float32{0.8, 0.2, 0.2},
		BaseWeight:        1,
		SpecularRoughness: 0.3,
		SpecularIOR:       1.5,
	}
	return verts, indices, cam, params
}

// realisticTestSceneEnvLight mirrors realisticTestScene but with LightIsEnvironment set
// (#2135/#2155's illumination-contribution follow-up): the shader must source its light
// color from the bound environment texture (dummyEnvGrey in these harnesses, which never
// call InitViewport) at LightDirection, instead of LightColor — LightColor is deliberately
// zeroed so a shader that wrongly ignores LightIsEnvironment renders black (an obvious
// large diff) rather than coincidentally matching by falling through to an unused field.
func realisticTestSceneEnvLight() (verts []float32, indices []uint32, cam CameraBasis, params RealisticLightParams) {
	verts, indices, cam, params = realisticTestScene()
	params.LightIsEnvironment = 1
	params.EnvIntensity = 3
	params.LightColor = [3]float32{0, 0, 0}
	return verts, indices, cam, params
}

// pixelRay reproduces pathtrace_realistic.rgen/swpathtrace_realistic.comp's per-pixel
// camera-ray generation exactly, in Go, so the CPU oracle probes the SAME ray a GPU
// invocation for (px,py) would trace.
func pixelRay(px, py, width, height int, cam CameraBasis) (origin, dir [3]float32) {
	ndcX := (float32(px)+0.5)/float32(width)*2 - 1
	ndcY := -((float32(py)+0.5)/float32(height)*2 - 1)
	fx, fy, fz := cam.Forward[0], cam.Forward[1], cam.Forward[2]
	fx += ndcX * cam.Aspect * cam.TanHalfFovY * cam.Right[0]
	fy += ndcX * cam.Aspect * cam.TanHalfFovY * cam.Right[1]
	fz += ndcX * cam.Aspect * cam.TanHalfFovY * cam.Right[2]
	fx += ndcY * cam.TanHalfFovY * cam.Up[0]
	fy += ndcY * cam.TanHalfFovY * cam.Up[1]
	fz += ndcY * cam.TanHalfFovY * cam.Up[2]
	n := float32(math.Sqrt(float64(fx*fx + fy*fy + fz*fz)))
	return cam.Eye, [3]float32{fx / n, fy / n, fz / n}
}

// dummyEnvGrey mirrors raytrace.cpp/swtrace.cpp's ensure_dummy_env's own hardcoded 1x1
// fallback color — env_binding_for falls back to it whenever a scene's HeadContext has
// no live Viewport, which every test harness in this file is (CreateWindow, never
// InitViewport). cpuOracleRealisticPixel's LightIsEnvironment branch substitutes it as
// the "sampled" color, since these harnesses never upload a real equirect texture for
// the oracle to independently decode pixel-for-pixel.
const dummyEnvGrey = 0.05

// cpuOracleRealisticPixel reproduces pathtrace_realistic.rchit/swpathtrace_realistic.comp's
// shading math exactly in Go (directional light, no inverse-square), intersecting the
// flat z=0 quad analytically (bounded to x,y in [-5,5]) rather than re-implementing
// ray-triangle intersection — TestRTSceneMatchesCPUOracle/TestSWSceneMatchesCPUOracle
// (PBI-333/334) already cover that. p.LightIsEnvironment != 0 (#2135/#2155's
// illumination-contribution follow-up) substitutes dummyEnvGrey for p.LightColor,
// mirroring directLightAt's own env-vs-discrete-light branch.
func cpuOracleRealisticPixel(origin, direction [3]float32, p RealisticLightParams) (r, g, b float32) {
	if direction[2] >= 0 {
		return 0, 0, 0 // ray parallel to or moving away from the z=0 plane: no hit
	}
	t := -origin[2] / direction[2]
	if t <= 0 {
		return 0, 0, 0
	}
	hx := origin[0] + direction[0]*t
	hy := origin[1] + direction[1]*t
	if hx < -5 || hx > 5 || hy < -5 || hy > 5 {
		return 0, 0, 0 // outside the quad: miss
	}

	wi := openpbr.Vec3{X: float64(p.LightDirection[0]), Y: float64(p.LightDirection[1]), Z: float64(p.LightDirection[2])}.Normalize()
	wo := openpbr.Vec3{X: -float64(direction[0]), Y: -float64(direction[1]), Z: -float64(direction[2])}
	// buildBasis(n=(0,0,1)) picks tangent=(1,0,0), bitangent=(0,1,0) — the local frame
	// equals the world frame here, same as cpuOracleRadiance's fixture.
	if wi.Z <= 0 || wo.Z <= 0 {
		return 0, 0, 0
	}

	rho := openpbr.Color3{R: float64(p.BaseColor[0]) * float64(p.BaseWeight),
		G: float64(p.BaseColor[1]) * float64(p.BaseWeight), B: float64(p.BaseColor[2]) * float64(p.BaseWeight)}
	diffuse := openpbr.DiffuseEON(rho, 0, wi, wo)

	alpha := openpbr.AlphaFromRoughness(float64(p.SpecularRoughness))
	h := wi.Add(wo).Normalize()
	d := openpbr.DistributionGGX(h, alpha)
	g2 := openpbr.SmithG2(wi, wo, alpha)
	fr := openpbr.DielectricFresnel(float64(p.SpecularIOR), math.Max(wi.Dot(h), 0))
	specular := fr * d * g2 / (4 * wi.Z * wo.Z)

	lightColor := [3]float32{p.LightColor[0], p.LightColor[1], p.LightColor[2]}
	if p.LightIsEnvironment != 0 {
		lightColor = [3]float32{dummyEnvGrey * p.EnvIntensity, dummyEnvGrey * p.EnvIntensity, dummyEnvGrey * p.EnvIntensity}
	}

	cosTheta := wi.Z
	rF := (diffuse.R + specular) * float64(lightColor[0]) * float64(p.LightIntensity) * cosTheta
	gF := (diffuse.G + specular) * float64(lightColor[1]) * float64(p.LightIntensity) * cosTheta
	bF := (diffuse.B + specular) * float64(lightColor[2]) * float64(p.LightIntensity) * cosTheta
	return float32(rF), float32(gF), float32(bF)
}

// checkImageAgainstOracle asserts every pixel of a width*height*3 image matches the
// CPU oracle within tol, failing fast with the first mismatching pixel's coordinates.
func checkImageAgainstOracle(t *testing.T, got []float32, width, height int, cam CameraBasis, params RealisticLightParams, tol float32) {
	t.Helper()
	hits, misses := 0, 0
	for y := range height {
		for x := range width {
			origin, dir := pixelRay(x, y, width, height, cam)
			wantR, wantG, wantB := cpuOracleRealisticPixel(origin, dir, params)
			i := (y*width + x) * 3
			gotR, gotG, gotB := got[i], got[i+1], got[i+2]
			if abs32(gotR-wantR) > tol || abs32(gotG-wantG) > tol || abs32(gotB-wantB) > tol {
				t.Fatalf("pixel (%d,%d) = (%v,%v,%v), want (%v,%v,%v) within %v", x, y, gotR, gotG, gotB, wantR, wantG, wantB, tol)
			}
			if wantR > 0 || wantG > 0 || wantB > 0 {
				hits++
			} else {
				misses++
			}
		}
	}
	if hits == 0 || misses == 0 {
		t.Fatalf("degenerate test image: %d hit pixels, %d miss pixels — want both present", hits, misses)
	}
}

func TestRTSceneRealisticImageMatchesCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (realistic HW image test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewRTScene()
	if err != nil {
		t.Skipf("no hardware ray tracing available: %v", err)
	}
	defer scene.Destroy()

	verts, indices, cam, params := realisticTestScene()
	if err := scene.AddMesh(verts, indices, 1); err != nil {
		t.Fatalf("AddMesh: %v", err)
	}
	if err := scene.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := scene.BuildRealisticPipeline(pathtraceRealisticRgenSPV, pathtraceRealisticRmissSPV, shadowRmissSPV, pathtraceRealisticRchitSPV); err != nil {
		t.Skipf("no hardware RT pipeline available: %v", err)
	}

	const width, height = 16, 16
	pixels, err := scene.TraceRealisticImage(width, height, cam, params)
	if err != nil {
		t.Fatalf("TraceRealisticImage: %v", err)
	}
	checkImageAgainstOracle(t, pixels, width, height, cam, params, 0.02)
}

func TestSWSceneRealisticImageMatchesCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (realistic SW image test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewSWScene()
	if err != nil {
		t.Skipf("no compute-capable device available: %v", err)
	}
	defer scene.Destroy()

	verts, indices, cam, params := realisticTestScene()
	triangles := make([]renderer.Triangle, len(indices)/3)
	for i := range triangles {
		v := func(j int) [3]float32 {
			k := indices[i*3+j] * 3
			return [3]float32{verts[k], verts[k+1], verts[k+2]}
		}
		triangles[i] = renderer.Triangle{V0: v(0), V1: v(1), V2: v(2), InstanceID: 1, PrimitiveID: uint32(i)}
	}
	bvh := renderer.BuildBVH(triangles)
	if err := scene.Build(SWBuildInputFrom(bvh, triangles)); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := scene.BuildRealisticPathtracePipeline(swpathtraceRealisticCompSPV); err != nil {
		t.Fatalf("BuildRealisticPathtracePipeline: %v", err)
	}

	const width, height = 16, 16
	pixels, err := scene.TraceRealisticPathtraceImage(width, height, cam, params)
	if err != nil {
		t.Fatalf("TraceRealisticPathtraceImage: %v", err)
	}
	checkImageAgainstOracle(t, pixels, width, height, cam, params, 0.02)
}

// TestRealisticImageBackendParity is PBI-350's second acceptance criterion at full-image
// granularity: an identical scene rendered through the hardware and software per-pixel
// pipelines must agree, extending PBI-346's single-ray parity test through a real image.
func TestRealisticImageBackendParity(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (realistic image backend parity test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	verts, indices, cam, params := realisticTestScene()

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
	if err := rtScene.BuildRealisticPipeline(pathtraceRealisticRgenSPV, pathtraceRealisticRmissSPV, shadowRmissSPV, pathtraceRealisticRchitSPV); err != nil {
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
	if err := swScene.Build(SWBuildInputFrom(bvh, triangles)); err != nil {
		t.Fatalf("SWScene.Build: %v", err)
	}
	if err := swScene.BuildRealisticPathtracePipeline(swpathtraceRealisticCompSPV); err != nil {
		t.Fatalf("SWScene.BuildRealisticPathtracePipeline: %v", err)
	}

	const width, height = 16, 16
	rtPixels, err := rtScene.TraceRealisticImage(width, height, cam, params)
	if err != nil {
		t.Fatalf("TraceRealisticImage: %v", err)
	}
	swPixels, err := swScene.TraceRealisticPathtraceImage(width, height, cam, params)
	if err != nil {
		t.Fatalf("TraceRealisticPathtraceImage: %v", err)
	}

	for i := range rtPixels {
		if abs32(rtPixels[i]-swPixels[i]) > 0.02 {
			px, ch := i/3, i%3
			t.Fatalf("pixel %d channel %d: hardware=%v software=%v, want equal within 0.02", px, ch, rtPixels[i], swPixels[i])
		}
	}
}

// TestRTSceneRealisticImageEnvironmentLightMatchesCPUOracle is
// TestRTSceneRealisticImageMatchesCPUOracle's #2135/#2155 counterpart: LightIsEnvironment
// set instead of a discrete light, exercising pathtrace_realistic.rchit's directLightAt
// env-sampling branch (and its binding-4 envMap descriptor) end to end.
func TestRTSceneRealisticImageEnvironmentLightMatchesCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (realistic HW env-light image test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewRTScene()
	if err != nil {
		t.Skipf("no hardware ray tracing available: %v", err)
	}
	defer scene.Destroy()

	verts, indices, cam, params := realisticTestSceneEnvLight()
	if err := scene.AddMesh(verts, indices, 1); err != nil {
		t.Fatalf("AddMesh: %v", err)
	}
	if err := scene.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := scene.BuildRealisticPipeline(pathtraceRealisticRgenSPV, pathtraceRealisticRmissSPV, shadowRmissSPV, pathtraceRealisticRchitSPV); err != nil {
		t.Skipf("no hardware RT pipeline available: %v", err)
	}

	const width, height = 16, 16
	pixels, err := scene.TraceRealisticImage(width, height, cam, params)
	if err != nil {
		t.Fatalf("TraceRealisticImage: %v", err)
	}
	checkImageAgainstOracle(t, pixels, width, height, cam, params, 0.02)
}

// TestSWSceneRealisticImageEnvironmentLightMatchesCPUOracle is
// TestSWSceneRealisticImageMatchesCPUOracle's #2135/#2155 counterpart, exercising
// swpathtrace_realistic.comp's directLightAt env-sampling branch (binding 6's envMap).
func TestSWSceneRealisticImageEnvironmentLightMatchesCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (realistic SW env-light image test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewSWScene()
	if err != nil {
		t.Skipf("no compute-capable device available: %v", err)
	}
	defer scene.Destroy()

	verts, indices, cam, params := realisticTestSceneEnvLight()
	triangles := make([]renderer.Triangle, len(indices)/3)
	for i := range triangles {
		v := func(j int) [3]float32 {
			k := indices[i*3+j] * 3
			return [3]float32{verts[k], verts[k+1], verts[k+2]}
		}
		triangles[i] = renderer.Triangle{V0: v(0), V1: v(1), V2: v(2), InstanceID: 1, PrimitiveID: uint32(i)}
	}
	bvh := renderer.BuildBVH(triangles)
	if err := scene.Build(SWBuildInputFrom(bvh, triangles)); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := scene.BuildRealisticPathtracePipeline(swpathtraceRealisticCompSPV); err != nil {
		t.Fatalf("BuildRealisticPathtracePipeline: %v", err)
	}

	const width, height = 16, 16
	pixels, err := scene.TraceRealisticPathtraceImage(width, height, cam, params)
	if err != nil {
		t.Fatalf("TraceRealisticPathtraceImage: %v", err)
	}
	checkImageAgainstOracle(t, pixels, width, height, cam, params, 0.02)
}

// TestRealisticImageEnvironmentLightBackendParity is TestRealisticImageBackendParity's
// #2135/#2155 counterpart: hardware and software must agree when the picked light IS the
// environment, not just for a discrete light.
func TestRealisticImageEnvironmentLightBackendParity(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (realistic env-light image backend parity test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	verts, indices, cam, params := realisticTestSceneEnvLight()

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
	if err := rtScene.BuildRealisticPipeline(pathtraceRealisticRgenSPV, pathtraceRealisticRmissSPV, shadowRmissSPV, pathtraceRealisticRchitSPV); err != nil {
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
	if err := swScene.Build(SWBuildInputFrom(bvh, triangles)); err != nil {
		t.Fatalf("SWScene.Build: %v", err)
	}
	if err := swScene.BuildRealisticPathtracePipeline(swpathtraceRealisticCompSPV); err != nil {
		t.Fatalf("SWScene.BuildRealisticPathtracePipeline: %v", err)
	}

	const width, height = 16, 16
	rtPixels, err := rtScene.TraceRealisticImage(width, height, cam, params)
	if err != nil {
		t.Fatalf("TraceRealisticImage: %v", err)
	}
	swPixels, err := swScene.TraceRealisticPathtraceImage(width, height, cam, params)
	if err != nil {
		t.Fatalf("TraceRealisticPathtraceImage: %v", err)
	}

	for i := range rtPixels {
		if abs32(rtPixels[i]-swPixels[i]) > 0.02 {
			px, ch := i/3, i%3
			t.Fatalf("pixel %d channel %d: hardware=%v software=%v, want equal within 0.02", px, ch, rtPixels[i], swPixels[i])
		}
	}
}
