//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Two-cap-exit cut live render (EPIC Oblikovati/Oblikovati#1724, ADR-0046). The kernel certifies the
// two-cap-exit cut against OCC moments + a membership audit (kernel/ops); this drives the SAME cut through
// the real feature pipeline and renders it offscreen, proving the whole wall + two ellipse-holed caps +
// tunnel reach the screen as lit geometry. Shares bodyFractionOf/backgroundRendered with the rim-crossing
// render (both robust to the second-window llvmpipe dimming/black-frame harness artifact).
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

// twoCapPart returns a session whose active part holds the two-cap-exit cut: an r=3 h=10 target minus a steep
// oblique r=0.7 tool that enters one cap and exits the other, leaving the wall whole.
func twoCapPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "two-cap.opd", true)
	if err != nil {
		t.Fatalf("add part: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	th := 20.0 * stdmath.Pi / 180
	ux, uz := stdmath.Sin(th), stdmath.Cos(th)
	target, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target cylinder: %v", err)
	}
	tool, err := brep.SolidCylinder(math.P3(-2.416, 0, -2.518), math.V3(math.Scalar(ux), 0, math.Scalar(uz)), 0.7, 16)
	if err != nil {
		t.Fatalf("tool cylinder: %v", err)
	}
	feature.NewBaseFeatures(def.Features()).AddBase(target)
	feature.NewBaseFeatures(def.Features()).AddBase(tool)
	feature.NewModifyFeatures(def.Features()).AddCombine(0, 1, ops.Cut)
	def.Recompute()
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(10, -8.5, 14), math.P3(0, 0, 5.5), math.V3(0, 0, 1)
	s.SetCamera(cam)
	return s
}

// TestTwoCapCrossingCutRendersLive asserts the two-cap cut reaches the viewport as a large solid and writes a
// PNG for visual inspection.
func TestTwoCapCrossingCutRendersLive(t *testing.T) {
	dockLaidOut = false
	icons = nil
	win := newViewportWindow(t)
	defer win.Destroy()

	s := twoCapPart(t)
	for range 8 {
		viewportFrame(win, s)
	}
	px, w, h, ok := win.ReadbackViewport(0)
	if !ok {
		t.Fatal("viewport readback unavailable")
	}
	if !backgroundRendered(px) {
		t.Skip("offscreen frame did not composite (all-black, no background) — second-window llvmpipe harness artifact")
	}
	body := bodyFractionOf(px, w, h)
	t.Logf("two-cap cut body fraction = %.4f", body)
	if body < 0.10 {
		t.Errorf("two-cap cut rendered near-blank: body fraction %.4f — the cut B-rep is not reaching the screen", body)
	}
	dir := t.TempDir()
	if p := os.Getenv("OBK_SHOT_DIR"); p != "" {
		dir = p
	}
	out := filepath.Join(dir, "two-cap-cut.png")
	if err := writeBGRAPNG(px, w, h, out); err != nil {
		t.Fatalf("write PNG: %v", err)
	}
	t.Logf("saved two-cap cut render to %s", out)
}
