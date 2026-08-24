//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"math"
	"testing"

	"oblikovati.org/kernel/shading/openpbr"
	"oblikovati.org/renderer"
)

// specularSingleScatter reproduces extended_lobes.glsl's openpbrSpecularCoat exactly:
// GGX + exact Fresnel, single-scatter only (no Kulla-Conty compensation) — the same
// simplification realistic_image_test.go's own cpuOracleRealisticPixel already makes for
// the base specular term, applied here at the coat's own roughness/ior.
func specularSingleScatter(wi, wo openpbr.Vec3, roughness, ior float64) float64 {
	cosI, cosO := wi.Z, wo.Z
	if cosI <= 0 || cosO <= 0 {
		return 0
	}
	alpha := openpbr.AlphaFromRoughness(roughness)
	h := wi.Add(wo).Normalize()
	d := openpbr.DistributionGGX(h, alpha)
	g := openpbr.SmithG2(wi, wo, alpha)
	fr := openpbr.DielectricFresnel(ior, math.Abs(wi.Dot(h)))
	return fr * d * g / (4 * cosI * cosO)
}

// lerpColor3 mirrors kernel/shading/openpbr's own unexported lerpColor.
func lerpColor3(a, b openpbr.Color3, t float64) openpbr.Color3 {
	return openpbr.Color3{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
	}
}

// cpuOracleExtendedPixel reproduces extended_lobes.glsl's openpbrShadeSurface exactly,
// including its two documented simplifications relative to the CPU reference package:
// Fuzz's LayerFuzz is a plain weighted mix (no fuzzScalarAlbedo energy compensation, see
// extended_lobes.glsl's header), and Subsurface is fed through the same single-scatter
// diffuse shape as the base lobe (roughness=0, so algebraically exactly rho/pi) rather
// than a real random walk. Coat and Thin-film ARE exact ports, so this oracle calls
// kernel/shading/openpbr's own CoatDarkeningFactor/LayerCoat/FresnelWithThinFilm
// directly. Intersects the same flat z=0 quad as cpuOracleRealisticPixel.
func cpuOracleExtendedPixel(origin, direction [3]float32, p RealisticLightParams) (r, g, b float32) {
	if direction[2] >= 0 {
		return 0, 0, 0
	}
	t := -origin[2] / direction[2]
	if t <= 0 {
		return 0, 0, 0
	}
	hx := origin[0] + direction[0]*t
	hy := origin[1] + direction[1]*t
	if hx < -5 || hx > 5 || hy < -5 || hy > 5 {
		return 0, 0, 0
	}

	wi := openpbr.Vec3{X: float64(p.LightDirection[0]), Y: float64(p.LightDirection[1]), Z: float64(p.LightDirection[2])}.Normalize()
	wo := openpbr.Vec3{X: -float64(direction[0]), Y: -float64(direction[1]), Z: -float64(direction[2])}
	if wi.Z <= 0 || wo.Z <= 0 {
		return 0, 0, 0
	}

	baseColor := openpbr.Color3{R: float64(p.BaseColor[0]), G: float64(p.BaseColor[1]), B: float64(p.BaseColor[2])}
	diffuse := openpbr.DiffuseEON(baseColor.Scale(float64(p.BaseWeight)), 0, wi, wo)

	h := wi.Add(wo).Normalize()
	cosH := math.Max(wi.Dot(h), 0)
	fr := openpbr.FresnelWithThinFilm(cosH, float64(p.SpecularIOR), float64(p.ThinFilmIOR),
		float64(p.ThinFilmThicknessMicrons), float64(p.ThinFilmWeight))
	alpha := openpbr.AlphaFromRoughness(float64(p.SpecularRoughness))
	d := openpbr.DistributionGGX(h, alpha)
	g2 := openpbr.SmithG2(wi, wo, alpha)
	specTerm := d * g2 / (4 * wi.Z * wo.Z)
	specular := fr.Scale(specTerm)

	diffuseSlab := diffuse
	if p.SubsurfaceWeight > 0 {
		ssColor := openpbr.Color3{R: float64(p.SubsurfaceColor[0]), G: float64(p.SubsurfaceColor[1]), B: float64(p.SubsurfaceColor[2])}
		albedo := openpbr.SubsurfaceSingleScatterAlbedo(ssColor, float64(p.SubsurfaceAnisotropy))
		subsurface := albedo.Scale(1 / math.Pi) // openpbrDiffuseSingleScatter at roughness=0
		diffuseSlab = openpbr.MixSubsurface(diffuse, subsurface, float64(p.SubsurfaceWeight))
	}
	base := diffuseSlab.Add(specular)

	coatColor := openpbr.Color3{R: float64(p.CoatColor[0]), G: float64(p.CoatColor[1]), B: float64(p.CoatColor[2])}
	fCoat := specularSingleScatter(wi, wo, float64(p.CoatRoughness), float64(p.CoatIOR))
	darkening := openpbr.CoatDarkeningFactor(1, float64(p.CoatWeight), float64(p.CoatDarkening), float64(p.CoatIOR))
	coated := openpbr.LayerCoat(fCoat, base, coatColor, float64(p.CoatWeight), darkening, wo, float64(p.CoatIOR))

	fuzzColor := openpbr.Color3{R: float64(p.FuzzColor[0]), G: float64(p.FuzzColor[1]), B: float64(p.FuzzColor[2])}
	fuzzed := coated
	if p.FuzzWeight > 0 {
		fFuzz := openpbr.SpecularFuzz(wi, wo, float64(p.FuzzRoughness), fuzzColor)
		fuzzed = lerpColor3(coated, fFuzz, float64(p.FuzzWeight))
	}

	cosTheta := wi.Z
	scale := float64(p.LightIntensity) * cosTheta
	return float32(fuzzed.R * float64(p.LightColor[0]) * scale),
		float32(fuzzed.G * float64(p.LightColor[1]) * scale),
		float32(fuzzed.B * float64(p.LightColor[2]) * scale)
}

// checkImageAgainstExtendedOracle mirrors checkImageAgainstOracle for the extended lobes.
func checkImageAgainstExtendedOracle(t *testing.T, got []float32, width, height int, cam CameraBasis, params RealisticLightParams, tol float32) {
	t.Helper()
	hits := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			origin, dir := pixelRay(x, y, width, height, cam)
			wantR, wantG, wantB := cpuOracleExtendedPixel(origin, dir, params)
			i := (y*width + x) * 3
			gotR, gotG, gotB := got[i], got[i+1], got[i+2]
			if abs32(gotR-wantR) > tol || abs32(gotG-wantG) > tol || abs32(gotB-wantB) > tol {
				t.Fatalf("pixel (%d,%d) = (%v,%v,%v), want (%v,%v,%v) within %v", x, y, gotR, gotG, gotB, wantR, wantG, wantB, tol)
			}
			if wantR > 0 || wantG > 0 || wantB > 0 {
				hits++
			}
		}
	}
	if hits == 0 {
		t.Fatal("degenerate test image: no hit pixels")
	}
}

// extendedLobeCase names one #2148 lobe test and how to populate its RealisticLightParams
// fields on top of realisticTestScene's base fixture.
type extendedLobeCase struct {
	name string
	set  func(p *RealisticLightParams)
}

var extendedLobeCases = []extendedLobeCase{
	{"coat", func(p *RealisticLightParams) {
		p.CoatColor = [3]float32{1, 1, 1}
		p.CoatWeight = 1
		p.CoatRoughness = 0.05
		p.CoatIOR = 1.5
		p.CoatDarkening = 1
	}},
	{"fuzz", func(p *RealisticLightParams) {
		p.FuzzColor = [3]float32{0.9, 0.9, 1}
		p.FuzzWeight = 0.6
		p.FuzzRoughness = 0.4
	}},
	{"thinfilm", func(p *RealisticLightParams) {
		p.ThinFilmWeight = 1
		p.ThinFilmThicknessMicrons = 0.5
		p.ThinFilmIOR = 1.3
	}},
	{"subsurface", func(p *RealisticLightParams) {
		p.SubsurfaceColor = [3]float32{0.9, 0.3, 0.2}
		p.SubsurfaceWeight = 0.7
		p.SubsurfaceAnisotropy = 0
	}},
}

// TestRTSceneExtendedLobesMatchCPUOracle is #2148's hardware-backend acceptance
// criterion: each extended lobe, dispatched through the SAME live per-pixel pipeline
// PBI-345/350 already verified for the base lobes, matches its Go CPU reference.
func TestRTSceneExtendedLobesMatchCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (extended lobes HW image test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewRTScene()
	if err != nil {
		t.Skipf("no hardware ray tracing available: %v", err)
	}
	defer scene.Destroy()

	verts, indices, cam, base := realisticTestScene()
	if err := scene.AddMesh(verts, indices, 1); err != nil {
		t.Fatalf("AddMesh: %v", err)
	}
	if err := scene.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := scene.BuildRealisticPipeline(pathtraceRealisticRgenSPV, pathtraceRmissSPV, shadowRmissSPV, pathtraceRealisticRchitSPV); err != nil {
		t.Skipf("no hardware RT pipeline available: %v", err)
	}

	const width, height = 16, 16
	for _, c := range extendedLobeCases {
		t.Run(c.name, func(t *testing.T) {
			params := base
			c.set(&params)
			pixels, err := scene.TraceRealisticImage(width, height, cam, params)
			if err != nil {
				t.Fatalf("TraceRealisticImage: %v", err)
			}
			checkImageAgainstExtendedOracle(t, pixels, width, height, cam, params, 0.02)
		})
	}
}

// TestSWSceneExtendedLobesMatchCPUOracle is #2148's software-backend counterpart.
func TestSWSceneExtendedLobesMatchCPUOracle(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (extended lobes SW image test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()

	scene, err := w.NewSWScene()
	if err != nil {
		t.Skipf("no compute-capable device available: %v", err)
	}
	defer scene.Destroy()

	verts, indices, cam, base := realisticTestScene()
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
	for _, c := range extendedLobeCases {
		t.Run(c.name, func(t *testing.T) {
			params := base
			c.set(&params)
			pixels, err := scene.TraceRealisticPathtraceImage(width, height, cam, params)
			if err != nil {
				t.Fatalf("TraceRealisticPathtraceImage: %v", err)
			}
			checkImageAgainstExtendedOracle(t, pixels, width, height, cam, params, 0.02)
		})
	}
}

// TestExtendedLobesZeroWeightReproducesBaseLobes is the #2148 regression guard mirrored
// from kernel/shading/openpbr's own convention (each Layer*/Mix* helper's own weight<=0
// short-circuit): every extended field left at its zero value must render bit-identical
// (within float tolerance) to the pre-#2148 base-lobes-only image.
func TestExtendedLobesZeroWeightReproducesBaseLobes(t *testing.T) {
	w, err := CreateWindow(64, 64, "Oblikovati (extended lobes zero-weight test)")
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
	if err := scene.BuildRealisticPipeline(pathtraceRealisticRgenSPV, pathtraceRmissSPV, shadowRmissSPV, pathtraceRealisticRchitSPV); err != nil {
		t.Skipf("no hardware RT pipeline available: %v", err)
	}

	const width, height = 16, 16
	pixels, err := scene.TraceRealisticImage(width, height, cam, params)
	if err != nil {
		t.Fatalf("TraceRealisticImage: %v", err)
	}
	// cpuOracleRealisticPixel (realistic_image_test.go, base lobes only) and
	// cpuOracleExtendedPixel (this file) must agree exactly when every extended field is
	// zero — proving openpbrShadeSurface's layering short-circuits are wired correctly.
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			origin, dir := pixelRay(x, y, width, height, cam)
			baseR, baseG, baseB := cpuOracleRealisticPixel(origin, dir, params)
			extR, extG, extB := cpuOracleExtendedPixel(origin, dir, params)
			if abs32(baseR-extR) > 1e-5 || abs32(baseG-extG) > 1e-5 || abs32(baseB-extB) > 1e-5 {
				t.Fatalf("pixel (%d,%d): base oracle (%v,%v,%v) != extended oracle (%v,%v,%v) at zero weight",
					x, y, baseR, baseG, baseB, extR, extG, extB)
			}
			i := (y*width + x) * 3
			if abs32(pixels[i]-baseR) > 0.02 || abs32(pixels[i+1]-baseG) > 0.02 || abs32(pixels[i+2]-baseB) > 0.02 {
				t.Fatalf("pixel (%d,%d) GPU = (%v,%v,%v), want base oracle (%v,%v,%v)",
					x, y, pixels[i], pixels[i+1], pixels[i+2], baseR, baseG, baseB)
			}
		}
	}
}
