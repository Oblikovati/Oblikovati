//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Cone-tool cap-crossing cut live render (EPIC Oblikovati/Oblikovati#1724, ADR-0046). The kernel certifies
// the cone-tool cap-crossing cut against OCC moments + a membership audit (kernel/ops); this drives the SAME
// cut through the real feature pipeline and renders it offscreen, proving the holed wall + ellipse-holed top
// cap + bottom cap + cone tunnel reach the screen as lit geometry. Shares bodyFractionOf/backgroundRendered
// with the rim-crossing render (robust to the second-window llvmpipe dimming/black-frame harness artifact).
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

// coneCapPart returns a session whose active part holds the cone-tool cap-crossing cut: an r=3 h=10 target
// minus an oblique frustum (rBase 0.9 → rTop 0.6, 45°) that enters the wall once and exits the top cap.
func coneCapPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "cone-cap.opd", true)
	if err != nil {
		t.Fatalf("add part: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sc := 1 / stdmath.Sqrt2
	target, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target cylinder: %v", err)
	}
	top := math.P3(math.Scalar(-6.5+16*sc), 0, math.Scalar(2+16*sc))
	tool, err := brep.SolidCylinderCone(math.P3(-6.5, 0, 2), top, 0.9, 0.6, "cone")
	if err != nil {
		t.Fatalf("tool frustum: %v", err)
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

// TestConeCapCrossingCutRendersLive asserts the cone-tool cut reaches the viewport as a large solid and
// writes a PNG for visual inspection.
func TestConeCapCrossingCutRendersLive(t *testing.T) {
	dockLaidOut = false
	icons = nil
	win := newViewportWindow(t)
	defer win.Destroy()

	s := coneCapPart(t)
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
	t.Logf("cone-cap cut body fraction = %.4f", body)
	if body < 0.10 {
		t.Errorf("cone-cap cut rendered near-blank: body fraction %.4f — the cut B-rep is not reaching the screen", body)
	}
	dir := t.TempDir()
	if p := os.Getenv("OBK_SHOT_DIR"); p != "" {
		dir = p
	}
	out := filepath.Join(dir, "cone-cap-cut.png")
	if err := writeBGRAPNG(px, w, h, out); err != nil {
		t.Fatalf("write PNG: %v", err)
	}
	t.Logf("saved cone-cap cut render to %s", out)
}
