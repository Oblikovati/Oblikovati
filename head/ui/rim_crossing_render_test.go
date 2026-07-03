//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Rim-crossing cut live render (EPIC Oblikovati/Oblikovati#1724, ADR-0046, slice 2). The kernel certifies the
// rim-crossing cap-crossing cut against OCC moments + a membership audit (kernel/ops); this drives the SAME
// cut through the real feature pipeline (two base cylinders + a Combine-Cut) and renders it offscreen through
// the actual viewport, proving the watertight two-rim-band wall + mixed-arc cap reach the screen as lit
// geometry. Before the notched-band wall mesher (twoRimHoledBandMesh) this cut rendered as a torn, inside-out
// band (~140 free edges); a regression there would drop the lit fraction or leave a visible hole in the wall.
package ui

import (
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/scene"
)

// rimCrossPart returns a session whose active part holds the slice-2 rim-crossing cut, built through the real
// feature recompute: an r=3 h=10 target cylinder minus an oblique r=0.9 tool (base -5.6) whose exit ellipse
// crosses the top rim, notching the wall.
func rimCrossPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "rim-cross.opd", true)
	if err != nil {
		t.Fatalf("add part: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sc := 1 / stdmath.Sqrt2
	target, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target cylinder: %v", err)
	}
	tool, err := brep.SolidCylinder(math.P3(-5.6, 0, 2), math.V3(math.Scalar(sc), 0, math.Scalar(sc)), 0.9, 16)
	if err != nil {
		t.Fatalf("tool cylinder: %v", err)
	}
	feature.NewBaseFeatures(def.Features()).AddBase(target)
	feature.NewBaseFeatures(def.Features()).AddBase(tool)
	feature.NewModifyFeatures(def.Features()).AddCombine(0, 1, ops.Cut)
	def.Recompute()
	// The slice-1 3/4 view from above-front-right (shared with capCrossPart): proven to survive the shared
	// lighting/exposure dimming a prior test leaves, and it shows both the +x top-rim notch and the wall.
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(10, -8.5, 14), math.P3(0, 0, 5.5), math.V3(0, 0, 1)
	s.SetCamera(cam)
	return s
}

// bodyFractionOf returns the fraction of viewport pixels covered by the light-gray solid, detected as grayish
// (r≈g≈b, so NOT blue-dominant like the gradient background) rather than by an absolute luma cutoff. A gray
// body stays gray when the shared renderer exposure dims between tests, so this is robust to test order where
// litFraction's fixed 175-luma threshold is not (Oblikovati#1724).
func bodyFractionOf(px []byte, w, h int) float64 {
	if w == 0 || h == 0 {
		return 0
	}
	body := 0
	for i := 0; i+3 < len(px); i += 4 {
		b, g, r := float64(px[i]), float64(px[i+1]), float64(px[i+2])
		if r > 30 && r > 0.75*b && g > 0.75*b { // not near-black, and not the blue-dominant background
			body++
		}
	}
	return float64(body) / float64(w*h)
}

// backgroundRendered reports whether the frame actually composited: a real frame clears to the blue gradient
// background, so it has many blue-dominant pixels; an all-black un-presented frame (the second-window llvmpipe
// artifact) has none.
func backgroundRendered(px []byte) bool {
	blue := 0
	for i := 0; i+3 < len(px); i += 4 {
		b, r := float64(px[i]), float64(px[i+2])
		if b > 40 && b > r+20 { // blue-dominant and above near-black
			blue++
		}
	}
	return blue > 100
}

// TestRimCrossingCutRendersLive asserts the rim-crossing cut reaches the viewport as a large solid (a torn
// wall or inverted tunnel would drop it off the screen) and writes a PNG for visual inspection.
func TestRimCrossingCutRendersLive(t *testing.T) {
	// Rebind the window-bound GPU caches to THIS fresh window/context (icons/dock globals otherwise still
	// point at a prior test's destroyed window) — correct hygiene per TestInWindowDockedViewportIsInteractive.
	dockLaidOut = false
	icons = nil
	win := newViewportWindow(t)
	defer win.Destroy()

	s := rimCrossPart(t)
	for i := 0; i < 8; i++ {
		viewportFrame(win, s)
	}
	px, w, h, ok := win.ReadbackViewport(0)
	if !ok {
		t.Fatal("viewport readback unavailable")
	}
	// A prior render test in the package can leave an offscreen Vulkan window that composites an all-black
	// frame under llvmpipe — the SECOND window never presents (reproduces with slice-1's own geometry, so it
	// is a harness artifact, not this cut). That frame has NO blue background either, unlike any real geometry
	// defect (which still clears to the gradient), so it is unambiguous: skip it like a missing-driver env,
	// while a composited frame with a torn/absent solid still hard-fails below. The kernel certification
	// (watertight + OCC moments + CSG membership, kernel/ops) is the geometry proof; this is the visual smoke.
	if !backgroundRendered(px) {
		t.Skip("offscreen frame did not composite (all-black, no background) — second-window llvmpipe harness artifact")
	}
	body := bodyFractionOf(px, w, h)
	t.Logf("rim-crossing cut body fraction = %.4f", body)
	if body < 0.10 {
		t.Errorf("rim-crossing cut rendered near-blank: body fraction %.4f — the cut B-rep is not reaching the screen", body)
	}
	dir := t.TempDir()
	if p := os.Getenv("OBK_SHOT_DIR"); p != "" {
		dir = p
	}
	out := filepath.Join(dir, "rim-crossing-cut.png")
	if err := writeBGRAPNG(px, w, h, out); err != nil {
		t.Fatalf("write PNG: %v", err)
	}
	t.Logf("saved rim-crossing cut render to %s", out)
}
