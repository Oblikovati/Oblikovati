//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/renderer"
)

// TestLinearBaseColorDecodesSRGB proves linearBaseColor undoes mesh.frag's own
// toLinear(c) = pow(c, 2.2) gamma encode — the fix for #2150 (Realistic mode was
// feeding renderer.Surface.Albedo's sRGB-encoded values straight to a BRDF that
// expects linear input, unlike the raster pipeline which decodes in-shader).
func TestLinearBaseColorDecodesSRGB(t *testing.T) {
	got := linearBaseColor([4]float32{0.5, 1, 0, 1})
	want := [3]float32{
		float32(stdmath.Pow(0.5, 2.2)),
		1,
		0,
	}
	for i := range got {
		if diff := got[i] - want[i]; diff > 1e-5 || diff < -1e-5 {
			t.Errorf("linearBaseColor(0.5,1,0)[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// solidEnvDistribution builds a small, uniformly-bright renderer.EnvironmentDistribution
// for pickLightParams/pickEnvironmentLight tests — luminance is the same at every texel,
// so Sample's direction is unconstrained by content (any RNG draw is equally valid) while
// its pdf is still a real, non-degenerate value these tests can check against.
func solidEnvDistribution() *renderer.EnvironmentDistribution {
	const w, h = 8, 4
	pixels := make([]float32, w*h*4)
	for i := range pixels {
		pixels[i] = 1
	}
	return renderer.NewEnvironmentDistribution(w, h, pixels)
}

func onLight(intensity float32) renderer.SceneLighting {
	return renderer.SceneLighting{Lights: []renderer.SceneLight{
		{Kind: renderer.DirectionalLight, Direction: [3]float32{0, 0, 1}, Color: [3]float32{1, 1, 1}, Intensity: intensity, On: true},
	}}
}

// TestPickLightParamsNoLightsNoEnvironmentIsUnlit covers pickLightParams' degenerate
// branch directly: no active lights and no active environment must leave every Light*
// field zero (shading to black downstream, not crashing or picking a phantom source).
func TestPickLightParamsNoLightsNoEnvironmentIsUnlit(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	got := pickLightParams(renderer.SceneLighting{}, rng, renderer.DrawItem{}, app.EnvironmentState{}, nil)
	if got.LightIntensity != 0 || got.LightIsEnvironment != 0 || got.LightDirection != ([3]float32{}) {
		t.Errorf("unlit scene: params = %+v, want every Light* field zero", got)
	}
}

// TestPickLightParamsDiscreteLightOnlyAlwaysPicksLight covers the (still-reachable)
// pre-#2135 code path directly: with no active environment, pickLightParams must never
// route to pickEnvironmentLight regardless of the RNG draw (pEnv is exactly 0 when
// envWeight is 0) — this is the coverage gap #2135's environment-selection branch left
// behind (pickDiscreteLight had 0% coverage from the existing GPU-backed Realistic-mode
// tests alone, since those fixtures all have an active environment).
func TestPickLightParamsDiscreteLightOnlyAlwaysPicksLight(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		got := pickLightParams(onLight(2), rng, renderer.DrawItem{}, app.EnvironmentState{}, nil)
		if got.LightIsEnvironment != 0 {
			t.Fatalf("seed %d: LightIsEnvironment = %v, want 0 (no active environment to compete for selection)", seed, got.LightIsEnvironment)
		}
		if got.LightIntensity <= 0 {
			t.Fatalf("seed %d: LightIntensity = %v, want > 0", seed, got.LightIntensity)
		}
	}
}

// TestPickLightParamsEnvironmentOnlyAlwaysPicksEnvironment is
// TestPickLightParamsDiscreteLightOnlyAlwaysPicksLight's mirror: with no active lights,
// pEnv is exactly 1, so pickLightParams must always route to pickEnvironmentLight
// regardless of the RNG draw.
func TestPickLightParamsEnvironmentOnlyAlwaysPicksEnvironment(t *testing.T) {
	env := app.EnvironmentState{Preset: "Sky", Intensity: 2}
	dist := solidEnvDistribution()
	for seed := int64(0); seed < 20; seed++ {
		rng := rand.New(rand.NewSource(seed))
		got := pickLightParams(renderer.SceneLighting{}, rng, renderer.DrawItem{}, env, dist)
		if got.LightIsEnvironment != 1 {
			t.Fatalf("seed %d: LightIsEnvironment = %v, want 1 (no active lights to compete for selection)", seed, got.LightIsEnvironment)
		}
		if got.LightIntensity <= 0 {
			t.Fatalf("seed %d: LightIntensity = %v, want > 0", seed, got.LightIntensity)
		}
		if length := stdmath.Sqrt(float64(got.LightDirection[0]*got.LightDirection[0] +
			got.LightDirection[1]*got.LightDirection[1] + got.LightDirection[2]*got.LightDirection[2])); stdmath.Abs(length-1) > 1e-4 {
			t.Fatalf("seed %d: LightDirection = %v, length %v, want a unit vector", seed, got.LightDirection, length)
		}
	}
}

// TestPickLightParamsWeightsSelectionBetweenStrategies checks pEnv actually discriminates
// on relative weight, not just presence/absence: a light overwhelmingly brighter than the
// environment should be picked far more often than the environment is, and vice versa —
// the property pEnv = envWeight/(envWeight+lightsWeight) exists to guarantee.
func TestPickLightParamsWeightsSelectionBetweenStrategies(t *testing.T) {
	dist := solidEnvDistribution()
	count := func(lightIntensity, envIntensity float32) (envPicks int) {
		rng := rand.New(rand.NewSource(7))
		env := app.EnvironmentState{Preset: "Sky", Intensity: envIntensity}
		const n = 2000
		for i := 0; i < n; i++ {
			if pickLightParams(onLight(lightIntensity), rng, renderer.DrawItem{}, env, dist).LightIsEnvironment != 0 {
				envPicks++
			}
		}
		return envPicks
	}

	if got := count(1000, 0.001); got > 50 {
		t.Errorf("light >> environment: environment picked %d/2000 times, want it rare (dominant light should win almost always)", got)
	}
	if got := count(0.001, 1000); got < 1950 {
		t.Errorf("environment >> light: environment picked %d/2000 times, want it dominant", got)
	}
}

// TestPickDiscreteLightSingleLightScalesByIntensity is a precise (not statistical) check
// of pickDiscreteLight's formula: with exactly one light, LightDistribution.Sample's pdf
// is always 1 regardless of the RNG draw, so LightIntensity should reduce to
// light.Intensity/pLight exactly — the same relationship pickEnvironmentLight's own
// LightIntensity = 1/(pEnv*pdf) mirrors.
func TestPickDiscreteLightSingleLightScalesByIntensity(t *testing.T) {
	dist := renderer.NewLightDistribution(onLight(3).Lights)
	rng := rand.New(rand.NewSource(1))
	var params native.RealisticLightParams
	const pLight = 0.7
	pickDiscreteLight(&params, rng, dist, pLight)

	wantIntensity := float32(3 / pLight)
	if diff := params.LightIntensity - wantIntensity; diff > 1e-4 || diff < -1e-4 {
		t.Errorf("LightIntensity = %v, want %v (light.Intensity=3 / pLight=%v, pdf=1 for a single light)", params.LightIntensity, wantIntensity, pLight)
	}
	if params.LightIsEnvironment != 0 {
		t.Errorf("LightIsEnvironment = %v, want 0", params.LightIsEnvironment)
	}
	if params.LightColor != ([3]float32{1, 1, 1}) {
		t.Errorf("LightColor = %v, want the light's own color (1,1,1)", params.LightColor)
	}
}

// TestPickDiscreteLightNilDistributionLeavesParamsZero guards pickDiscreteLight's own
// defensive nil check (reached when pLight<=0 too, e.g. an active environment claiming
// the entire selection weight) — must not panic or fabricate a light.
func TestPickDiscreteLightNilDistributionLeavesParamsZero(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	var params native.RealisticLightParams
	pickDiscreteLight(&params, rng, nil, 1)
	if params.LightIntensity != 0 {
		t.Errorf("LightIntensity = %v, want 0 (nil distribution)", params.LightIntensity)
	}
}

// TestPickEnvironmentLightScalesByPEnvAndPDF is pickDiscreteLight's env-branch
// counterpart: LightIsEnvironment/LightIntensity are set and LightDirection stays a unit
// vector after renderer.RotateAroundZ (a broken rotation matrix would change its length,
// not just its bearing) — checked structurally across several (pEnv, rotation) pairs
// rather than hand-computing Sample's exact pdf/direction, which
// TestEnvironmentDistributionPDFIntegratesToOne/TestEnvironmentDistributionDirectionIsUnitLength
// (renderer package) already cover directly.
func TestPickEnvironmentLightScalesByPEnvAndPDF(t *testing.T) {
	dist := solidEnvDistribution()
	cases := []struct {
		pEnv     float64
		rotation float32
	}{
		{0.4, 1.1}, {0.05, 0}, {0.95, -2.3}, {1.0, stdmath.Pi},
	}
	for _, c := range cases {
		rng := rand.New(rand.NewSource(5))
		var params native.RealisticLightParams
		pickEnvironmentLight(&params, rng, dist, c.rotation, c.pEnv)

		if params.LightIsEnvironment != 1 {
			t.Errorf("pEnv=%v rotation=%v: LightIsEnvironment = %v, want 1", c.pEnv, c.rotation, params.LightIsEnvironment)
		}
		if params.LightIntensity <= 0 {
			t.Errorf("pEnv=%v rotation=%v: LightIntensity = %v, want > 0", c.pEnv, c.rotation, params.LightIntensity)
		}
		length := stdmath.Sqrt(float64(params.LightDirection[0]*params.LightDirection[0] +
			params.LightDirection[1]*params.LightDirection[1] + params.LightDirection[2]*params.LightDirection[2]))
		if stdmath.Abs(length-1) > 1e-4 {
			t.Errorf("pEnv=%v rotation=%v: LightDirection = %v, length %v, want a unit vector", c.pEnv, c.rotation, params.LightDirection, length)
		}
	}
}

// TestRealisticTraceResolutionClampsToOnePixel covers realisticTraceResolution's own
// floor: a panel small enough that pw/realisticInteractiveDownscale truncates to 0 while
// the camera is moving must still trace at least a 1x1 preview, not a degenerate 0x0
// dispatch.
func TestRealisticTraceResolutionClampsToOnePixel(t *testing.T) {
	tracePW, tracePH := realisticTraceResolution(1, 1, true)
	if tracePW != 1 || tracePH != 1 {
		t.Errorf("realisticTraceResolution(1,1,true) = (%d,%d), want (1,1)", tracePW, tracePH)
	}
}

// TestPickEnvironmentLightAllBlackDistributionLeavesParamsZero covers
// pickEnvironmentLight's own pdf<=0 guard directly — unreachable through pickLightParams
// itself (an all-black environment has TotalWeight()==0, so pEnv is 0 and
// pickEnvironmentLight is never selected), but a real defensive check worth its own test:
// division by a zero pdf must not happen even if called directly.
func TestPickEnvironmentLightAllBlackDistributionLeavesParamsZero(t *testing.T) {
	dist := renderer.NewEnvironmentDistribution(8, 4, make([]float32, 8*4*4)) // all-black
	rng := rand.New(rand.NewSource(1))
	var params native.RealisticLightParams
	pickEnvironmentLight(&params, rng, dist, 0, 0.5)
	if params.LightIsEnvironment != 0 || params.LightIntensity != 0 {
		t.Errorf("params = %+v, want every field left zero (pdf<=0 guard)", params)
	}
}

// TestApplyEnvironmentParams covers applyEnvironmentParams' three branches directly:
// inactive (nothing set), active+ShowImage (background visibility AND rotation/intensity
// for light sampling), active+!ShowImage (rotation/intensity only — an environment can
// light the scene while hidden as a backdrop, applyEnvironmentParams' own doc comment).
func TestApplyEnvironmentParams(t *testing.T) {
	cases := []struct {
		name            string
		env             app.EnvironmentState
		wantEnvEnabled  float32
		wantEnvRotation float32
	}{
		{"inactive", app.EnvironmentState{}, 0, 0},
		{"active and shown", app.EnvironmentState{Preset: "Sky", Rotation: 1.2, Intensity: 3, ShowImage: true}, 1, 1.2},
		{"active but hidden", app.EnvironmentState{Preset: "Sky", Rotation: 1.2, Intensity: 3, ShowImage: false}, 0, 1.2},
	}
	for _, c := range cases {
		var params native.RealisticLightParams
		applyEnvironmentParams(&params, c.env)
		if params.EnvEnabled != c.wantEnvEnabled {
			t.Errorf("%s: EnvEnabled = %v, want %v", c.name, params.EnvEnabled, c.wantEnvEnabled)
		}
		if params.EnvRotation != c.wantEnvRotation {
			t.Errorf("%s: EnvRotation = %v, want %v", c.name, params.EnvRotation, c.wantEnvRotation)
		}
	}
}

// windowBrightFraction renders several frames (accumulating samples for Realistic mode)
// and returns the fraction of the composited window's pixels that are bright — the
// same ITU-R 601 luma threshold litFraction uses, but over ReadbackWindow (the whole
// composited swapchain, which is where the Realistic path's presentation texture
// actually lands via native.Image — unlike ReadbackViewport, which only sees the
// raster win.RenderViewport render target the Realistic path bypasses entirely).
func windowBrightFraction(win *native.Window, s *app.Session, frames int) float64 {
	for i := 0; i < frames; i++ {
		viewportFrame(win, s)
	}
	px, w, h, ok := win.ReadbackWindow()
	if !ok || w == 0 || h == 0 {
		return 0
	}
	bright, total := 0, w*h
	for i := 0; i+3 < len(px); i += 4 {
		b, g, r := float64(px[i]), float64(px[i+1]), float64(px[i+2])
		if 0.299*r+0.587*g+0.114*b > 40 { // Realistic's tone-mapped output is dimmer than the raster themed fill; a lower bar than litFraction's 175 still separates lit geometry from a near-black background
			bright++
		}
	}
	return float64(bright) / float64(total)
}

// TestRealisticModeSelectedViaAPIRendersNonBlank is PBI-350's first acceptance
// criterion: selecting kRealisticRendering (via the same app.Session.SetDisplayMode
// call api/client's MethodViewSetDisplayMode wire handler makes — addin/router/view.go's
// setDisplayMode) renders the new path tracer, asserted on the offscreen test backend.
// Compares against the SAME box rendered in ShadedWithEdges (the raster pipeline) rather
// than an empty-scene baseline: with no geometry, renderRealisticViewportImage returns
// ok=false and chrome_viewport.go's dispatcher falls back to raster (a deliberate
// "never leave the panel blank" safety net) — so an empty-scene comparison would compare
// raster background against raster background, proving nothing. Comparing against the
// SAME geometry in a known-raster style instead proves Realistic mode's pixels come from
// a genuinely different pipeline, not a silent fallback.
func TestRealisticModeSelectedViaAPIRendersNonBlank(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()

	s := rvBoxPart(t)
	defer DestroyRealisticState(win, s) // registered after win's defer, so it (LIFO) runs first
	if err := s.SetLightingStyle("Three Point"); err != nil {
		t.Fatalf("SetLightingStyle: %v", err) // exercises multi-light consumption end to end (#565 fold-in)
	}

	if err := s.SetDisplayMode(types.ShadedWithEdgesRendering); err != nil {
		t.Fatalf("SetDisplayMode(raster): %v", err)
	}
	viewportFrame(win, s)
	rasterPixels, w1, h1, ok1 := win.ReadbackWindow()
	if !ok1 {
		t.Fatal("ReadbackWindow (raster) failed")
	}

	if err := s.SetDisplayMode(types.RealisticRendering); err != nil {
		t.Fatalf("SetDisplayMode(realistic): %v", err)
	}
	lit := windowBrightFraction(win, s, 6)
	realisticPixels, w2, h2, ok2 := win.ReadbackWindow()
	if !ok2 {
		t.Fatal("ReadbackWindow (realistic) failed")
	}
	if w1 != w2 || h1 != h2 {
		t.Fatalf("window size changed between renders: %dx%d vs %dx%d", w1, h1, w2, h2)
	}

	if lit < 0.02 {
		t.Errorf("Realistic mode's own render is near-blank: bright fraction %.4f, want >= 0.02", lit)
	}

	diffPixels := 0
	for i := 0; i+3 < len(rasterPixels); i += 4 {
		if rasterPixels[i] != realisticPixels[i] || rasterPixels[i+1] != realisticPixels[i+1] || rasterPixels[i+2] != realisticPixels[i+2] {
			diffPixels++
		}
	}
	diffFraction := float64(diffPixels) / float64(w1*h1)
	t.Logf("Realistic bright fraction=%.4f, pixels differing from raster=%.4f", lit, diffFraction)
	if diffFraction < 0.05 {
		t.Errorf("Realistic mode's render is nearly identical to the raster render (%.4f of pixels differ, want >= 0.05) — suggests it silently fell back to the raster pipeline instead of path tracing", diffFraction)
	}
}

// TestRealisticSceneNotRebuiltOnCameraOnlyChange is the regression test for a live-testing
// finding during #2155: orbiting the camera used to retrigger a full RT/SW scene rebuild
// (destroy + recreate BLAS/TLAS + pipeline) every frame, because renderRealisticViewportImage
// conflated the accumulator's reset signal (fires on ANY camera/scene/material change) with
// the RT scene's own rebuild trigger (mesh content only) — turning a trivial 2-mesh part
// into ~370ms/frame instead of single-digit ms, and stressing the exact GPU-resource churn
// realisticState's own doc comment already flags as a hang-inducing failure class.
func TestRealisticSceneNotRebuiltOnCameraOnlyChange(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	s := rvBoxPart(t)
	defer DestroyRealisticState(win, s)
	if err := s.SetDisplayMode(types.RealisticRendering); err != nil {
		t.Fatalf("SetDisplayMode: %v", err)
	}

	const slot, pw, ph = 0, 64, 64
	cam := s.Camera()
	if _, _, ok := renderRealisticViewportImage(win, s, slot, cam, pw, ph); !ok {
		t.Fatal("first render did not build the RT scene")
	}
	st := realisticStateFor(s, slot)
	firstRT, firstSW := st.rt, st.sw
	if firstRT == nil && firstSW == nil {
		t.Fatal("neither backend built on first render")
	}

	for i := 0; i < 5; i++ {
		cam = cam.Orbit(0.2, 0)
		if _, _, ok := renderRealisticViewportImage(win, s, slot, cam, pw, ph); !ok {
			t.Fatalf("render %d (camera-only change) failed", i)
		}
	}

	if st.rt != firstRT {
		t.Errorf("hardware RT scene was rebuilt on camera-only change: got %p, want unchanged %p", st.rt, firstRT)
	}
	if st.sw != firstSW {
		t.Errorf("software scene was rebuilt on camera-only change: got %p, want unchanged %p", st.sw, firstSW)
	}
}

// TestRealisticModeCaptureViewportStaysRasterCaptureWindowIsLive is the regression test
// for #2149: drawRealisticResult (chrome_viewport.go) composites Realistic mode's
// path-traced texture directly onto the window's swapchain via native.Image, bypassing
// the offscreen viewport render target dispatchRasterDraw/win.RenderViewport writes to
// (and that ReadbackViewport/SaveViewportPNG — the wire.MethodViewportCapture /
// capture_viewport path — read back from). So switching to Realistic mode leaves that
// offscreen target holding the LAST raster frame, documented on
// [wire.CaptureViewportArgs] (#2149): callers needing a live Realistic-mode capture must
// use wire.MethodViewportCaptureWindow / capture_window instead, which reads back the
// composited swapchain ReadbackWindow also uses here.
func TestRealisticModeCaptureViewportStaysRasterCaptureWindowIsLive(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()

	s := rvBoxPart(t)
	defer DestroyRealisticState(win, s)
	if err := s.SetDisplayMode(types.ShadedWithEdgesRendering); err != nil {
		t.Fatalf("SetDisplayMode(raster): %v", err)
	}
	for i := 0; i < 8; i++ {
		viewportFrame(win, s)
	}
	rasterOffscreen, _, _, ok := win.ReadbackViewport(0)
	if !ok {
		t.Fatal("ReadbackViewport (raster) failed")
	}

	if err := s.SetDisplayMode(types.RealisticRendering); err != nil {
		t.Fatalf("SetDisplayMode(realistic): %v", err)
	}
	if lit := windowBrightFraction(win, s, 6); lit < 0.02 {
		t.Fatalf("Realistic mode's window composite is near-blank: bright fraction %.4f", lit)
	}

	staleOffscreen, w1, h1, ok := win.ReadbackViewport(0)
	if !ok {
		t.Fatal("ReadbackViewport (after switching to realistic) failed")
	}
	if len(staleOffscreen) != len(rasterOffscreen) {
		t.Fatalf("offscreen target size changed: %d vs %d bytes", len(staleOffscreen), len(rasterOffscreen))
	}
	for i := range rasterOffscreen {
		if staleOffscreen[i] != rasterOffscreen[i] {
			t.Fatalf("offscreen viewport target changed after switching to Realistic mode — "+
				"the #2149 documented limitation no longer holds, so capture_viewport's own doc "+
				"comment on wire.CaptureViewportArgs needs updating (byte %d: raster=%d now=%d)",
				i, rasterOffscreen[i], staleOffscreen[i])
		}
	}

	windowPixels, w2, h2, ok := win.ReadbackWindow()
	if !ok {
		t.Fatal("ReadbackWindow (realistic) failed")
	}
	if w1 == 0 || h1 == 0 || w2 == 0 || h2 == 0 {
		t.Fatalf("empty readback: offscreen %dx%d, window %dx%d", w1, h1, w2, h2)
	}
	bright, total := 0, w2*h2
	for i := 0; i+3 < len(windowPixels); i += 4 {
		b, g, r := float64(windowPixels[i]), float64(windowPixels[i+1]), float64(windowPixels[i+2])
		if 0.299*r+0.587*g+0.114*b > 40 {
			bright++
		}
	}
	if frac := float64(bright) / float64(total); frac < 0.02 {
		t.Errorf("ReadbackWindow (the capture_window path) bright fraction = %.4f, want >= 0.02 — "+
			"it should show the LIVE Realistic render even though capture_viewport cannot", frac)
	}
}

// TestRealisticModeBackendParityThroughRealUIPath is PBI-350's second acceptance
// criterion, extending PBI-346's single-ray and F04's per-pixel-image parity tests
// through the actual UI rendering path (renderViewportImage → renderRealisticViewportImage):
// the same scene, camera, and multi-light rig, accumulated for the same number of
// samples, must converge to (nearly) the same displayed image whether the hardware-RT
// checkbox forces the hardware or the software backend.
func TestRealisticModeBackendParityThroughRealUIPath(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()

	// 12, not more: both runs draw the SAME seeded light-pick sequence (see the tolerance
	// comment below), so parity is already exact well under 48 samples — kept modest to
	// limit this test's Vulkan-command churn, the heaviest in this suite and the one most
	// likely to trip this sandboxed environment's rare GPU-driver faults (already tracked
	// as a class, e.g. the #2143 root cause found and fixed earlier in this milestone).
	const samples = 12

	hwOn := true
	s := rvBoxPart(t)
	defer DestroyRealisticState(win, s) // registered after win's defer, so it (LIFO) runs first
	if err := s.SetDisplayMode(types.RealisticRendering); err != nil {
		t.Fatalf("SetDisplayMode: %v", err)
	}
	if err := s.SetLightingStyle("Three Point"); err != nil {
		t.Fatalf("SetLightingStyle: %v", err)
	}
	prefs := s.ViewCubePrefs()
	prefs.HardwareRayTracing = &hwOn
	s.SetViewCubePrefs(prefs)
	if !realisticHardwareEnabled(win, s) {
		t.Skip("no hardware ray tracing available on this device — parity check needs both backends")
	}
	for i := 0; i < samples; i++ {
		viewportFrame(win, s)
	}
	hwPixels, w1, h1, ok1 := win.ReadbackWindow()
	if !ok1 {
		t.Fatal("ReadbackWindow (hardware) failed")
	}

	hwOff := false
	prefs.HardwareRayTracing = &hwOff
	s.SetViewCubePrefs(prefs)
	DestroyRealisticState(win, s) // free the hardware run's GPU state and force a fresh accumulator
	for i := 0; i < samples; i++ {
		viewportFrame(win, s)
	}
	swPixels, w2, h2, ok2 := win.ReadbackWindow()
	if !ok2 {
		t.Fatal("ReadbackWindow (software) failed")
	}
	if w1 != w2 || h1 != h2 {
		t.Fatalf("window size changed between runs: %dx%d vs %dx%d", w1, h1, w2, h2)
	}

	var sumDiff, maxDiff float64
	for i := range hwPixels {
		d := float64(hwPixels[i]) - float64(swPixels[i])
		if d < 0 {
			d = -d
		}
		sumDiff += d
		if d > maxDiff {
			maxDiff = d
		}
	}
	meanDiff := sumDiff / float64(len(hwPixels))
	t.Logf("hardware vs software after %d samples: mean byte diff=%.3f, max byte diff=%.0f", samples, meanDiff, maxDiff)
	// Each fresh realisticState seeds its light-picking RNG identically (rand.NewSource(1)),
	// so both runs draw the SAME per-dispatch light-pick sequence — a deliberate, tight,
	// non-flaky comparison ("same inputs, does each backend compute the same shading"),
	// not two independently-noisy Monte Carlo accumulations converging toward each other.
	// A small tolerance still allows for backend floating-point/tone-map rounding and the
	// window's anti-aliased UI/HUD text pixels.
	if meanDiff > 6 {
		t.Errorf("mean per-channel byte difference between hardware and software backends = %.3f, want <= 6 (converged output should match, only convergence SPEED should differ)", meanDiff)
	}
}
