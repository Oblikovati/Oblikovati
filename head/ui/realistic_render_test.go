//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

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
