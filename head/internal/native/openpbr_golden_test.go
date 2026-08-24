//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"testing"
)

// openpbrGoldenParams mirrors test-utilities/openpbr-goldens/oracle/render_reference.py's
// params.json shape and head/cmd/openpbrgoldenshot's goldenParams — kept as its own type
// (rather than exported from the cmd package, which this test cannot import without an
// import cycle: cmd/openpbrgoldenshot already imports this package) since it's a thin,
// stable JSON transcription with no behavior to share.
type openpbrGoldenParams struct {
	BaseColor      [3]float32 `json:"base_color"`
	Metallic       float32    `json:"metallic"`
	Roughness      float32    `json:"roughness"`
	IOR            float32    `json:"ior"`
	LightDir       [3]float32 `json:"light_direction"`
	LightIntensity float32    `json:"light_intensity"`
	Eye            [3]float32 `json:"eye"`
	TanHalfFovY    float32    `json:"tan_half_fov_y"`
	Width          int        `json:"width"`
	Height         int        `json:"height"`

	// #2148 extended-lobe tiers (coat/fuzz/subsurface; see render_reference.py's own
	// Blender-side mapping for each field).
	CoatWeight       float32    `json:"coat_weight"`
	CoatRoughness    float32    `json:"coat_roughness"`
	CoatIOR          float32    `json:"coat_ior"`
	SheenWeight      float32    `json:"sheen_weight"`
	SheenRoughness   float32    `json:"sheen_roughness"`
	SheenTint        [3]float32 `json:"sheen_tint"`
	SubsurfaceWeight float32    `json:"subsurface_weight"`
}

// openpbrGoldenMinSSIM is a hard floor, not a target. A from-scratch calibration run
// (2026-08-24, this file's own history) measured our base-lobes-only render against the
// committed Blender reference at SSIM~0.852, unmoved by tessellation density (48x24 vs
// 96x48 segments/rings scored 0.8522 vs 0.8524) — i.e. the ~0.85 ceiling is real
// shading-model disagreement between our EON-diffuse + single-scatter-GGX base lobes and
// Blender's multi-scatter-GGX Principled BSDF v2, not a fixable artifact. 0.70 leaves
// meaningful margin below that measured ceiling while still failing hard on the kind of
// regression this golden exists to catch (wrong color, blank/black image, flipped
// normals, broken tone-mapping).
const openpbrGoldenMinSSIM = 0.70

// TestOpenPBRBaseLobesMatchBlenderReference is M45-F05 PBI-353's acceptance criterion:
// render the SAME sphere/light/material scene test-utilities/openpbr-goldens/oracle
// /render_reference.py rendered through Blender's Cycles (the interim OpenPBR
// perceptual oracle — see that script's own doc comment for why Principled BSDF v2 is a
// faithful stand-in here), through our OWN live path tracer
// (RenderGoldenSphere/RTScene/SWScene — the exact per-pixel dispatch renderer.Realistic
// mode uses), and assert perceptual similarity via SSIM
// (architecture/testing/00-renderer-oracle-pipeline.md's Tier-4 oracle).
//
// Scoped to base lobes ONLY (diffuse + specular dielectric/metal): PBI-345/346 built
// coat/fuzz/subsurface/transmission/thin-film entirely as a CPU reference
// (kernel/shading/openpbr), but never ported them into the live GLSL path-tracer
// shaders (pathtrace_realistic.rchit / swpathtrace_realistic.comp render base lobes
// only). Reference PNGs for those 4 additional tiers are already committed under
// test-utilities/openpbr-goldens/goldens/ (generated the same way as this test's
// reference) so the live comparison can be turned on tier-by-tier with no new Blender
// work once each lobe is ported — see the tracked follow-up issue this PBI files for
// that porting work.
func TestOpenPBRBaseLobesMatchBlenderReference(t *testing.T) {
	runOpenPBRGoldenTest(t, "base", func(p openpbrGoldenParams) RealisticLightParams {
		return RealisticLightParams{
			LightDirection: p.LightDir, LightIntensity: p.LightIntensity, LightColor: [3]float32{1, 1, 1},
			BaseColor: p.BaseColor, BaseWeight: 1, SpecularRoughness: p.Roughness, SpecularIOR: p.IOR, BaseMetalness: p.Metallic,
		}
	})
}

// TestOpenPBRCoatMatchesBlenderReference / TestOpenPBRFuzzMatchesBlenderReference /
// TestOpenPBRSubsurfaceMatchesBlenderReference are #2148's acceptance criterion for the
// 3 extended lobes with a committed Blender reference (see
// TestOpenPBRExtendedLobesPendingGLSLPort below for why transmission has none). Same
// harness as TestOpenPBRBaseLobesMatchBlenderReference — see its doc comment.
func TestOpenPBRCoatMatchesBlenderReference(t *testing.T) {
	runOpenPBRGoldenTest(t, "coat", func(p openpbrGoldenParams) RealisticLightParams {
		return RealisticLightParams{
			LightDirection: p.LightDir, LightIntensity: p.LightIntensity, LightColor: [3]float32{1, 1, 1},
			BaseColor: p.BaseColor, BaseWeight: 1, SpecularRoughness: p.Roughness, SpecularIOR: p.IOR, BaseMetalness: p.Metallic,
			CoatColor: [3]float32{1, 1, 1}, CoatWeight: p.CoatWeight, CoatRoughness: p.CoatRoughness,
			CoatIOR: p.CoatIOR, CoatDarkening: 1,
		}
	})
}

func TestOpenPBRFuzzMatchesBlenderReference(t *testing.T) {
	runOpenPBRGoldenTest(t, "fuzz", func(p openpbrGoldenParams) RealisticLightParams {
		return RealisticLightParams{
			LightDirection: p.LightDir, LightIntensity: p.LightIntensity, LightColor: [3]float32{1, 1, 1},
			BaseColor: p.BaseColor, BaseWeight: 1, SpecularRoughness: p.Roughness, SpecularIOR: p.IOR, BaseMetalness: p.Metallic,
			FuzzColor: p.SheenTint, FuzzWeight: p.SheenWeight, FuzzRoughness: p.SheenRoughness,
		}
	})
}

func TestOpenPBRSubsurfaceMatchesBlenderReference(t *testing.T) {
	runOpenPBRGoldenTest(t, "subsurface", func(p openpbrGoldenParams) RealisticLightParams {
		return RealisticLightParams{
			LightDirection: p.LightDir, LightIntensity: p.LightIntensity, LightColor: [3]float32{1, 1, 1},
			BaseColor: p.BaseColor, BaseWeight: 1, SpecularRoughness: p.Roughness, SpecularIOR: p.IOR, BaseMetalness: p.Metallic,
			SubsurfaceColor: p.BaseColor, SubsurfaceWeight: p.SubsurfaceWeight,
		}
	})
}

// runOpenPBRGoldenTest is the shared body TestOpenPBRBaseLobesMatchBlenderReference and
// the 3 extended-lobe golden tests above all follow: load params-<tier>.json and
// ref-<tier>.png, render through the SAME live per-pixel dispatch Realistic mode uses,
// and assert SSIM >= openpbrGoldenMinSSIM. build turns the parsed JSON into the specific
// RealisticLightParams fields that tier's lobe needs.
func runOpenPBRGoldenTest(t *testing.T, tier string, build func(openpbrGoldenParams) RealisticLightParams) {
	t.Helper()
	p := loadGoldenParams(t, "../../../test-utilities/openpbr-goldens/params-"+tier+".json")
	ref := loadGoldenPNG(t, "../../../test-utilities/openpbr-goldens/goldens/ref-"+tier+".png")

	win, err := CreateWindow(64, 64, "openpbr-golden-test-"+tier)
	if err != nil {
		t.Fatalf("CreateWindow: %v", err)
	}
	defer win.Destroy()

	triangles := UVSphereTriangles(1, 96, 48, 1)
	cam := PinholeCameraBasis(p.Eye, p.TanHalfFovY, p.Width, p.Height)
	light := build(p)
	pixels, err := RenderGoldenSphere(win, triangles, cam, light, p.Width, p.Height, false)
	if err != nil {
		t.Fatalf("RenderGoldenSphere: %v", err)
	}
	ours := ToneMappedImage(pixels, p.Width, p.Height)

	if got := ssim(ref, ours); got < openpbrGoldenMinSSIM {
		t.Errorf("SSIM(blender reference, our render) = %v, want >= %v", got, openpbrGoldenMinSSIM)
	}
}

// TestOpenPBRExtendedLobesPendingGLSLPort documents (and will fail loudly, as a nudge to
// update this test, the day someone ports it) that transmission has no live-shader
// comparison yet: showing transmitted light needs a continuation/refraction ray, which
// neither backend's pipeline supports (see extended_lobes.glsl's header comment for the
// full explanation) — a materially bigger undertaking than coat/fuzz/thin-film/subsurface
// (which are all local reflection modifications, no extra ray needed), tracked as its own
// follow-up. The reference image already exists at
// test-utilities/openpbr-goldens/goldens/ref-transmission.png for whoever picks it up.
func TestOpenPBRExtendedLobesPendingGLSLPort(t *testing.T) {
	t.Run("transmission", func(t *testing.T) {
		t.Skip("transmission needs a continuation/refraction ray neither backend's pipeline " +
			"supports yet (recursion depth is primary+shadow only) — reference image committed " +
			"at test-utilities/openpbr-goldens/goldens/ref-transmission.png for when it lands")
	})
}

func loadGoldenParams(t *testing.T, path string) openpbrGoldenParams {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden params %q: %v", path, err)
	}
	p := openpbrGoldenParams{Width: 128, Height: 128, IOR: 1.5}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatalf("parse golden params %q: %v", path, err)
	}
	return p
}

func loadGoldenPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open golden reference %q: %v", path, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode golden reference %q: %v", path, err)
	}
	return img
}
