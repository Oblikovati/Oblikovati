//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Cap-crossing cut live render (EPIC Oblikovati/Oblikovati#1724, ADR-0046). The kernel certifies the
// interior-exit cap-crossing cut against OCCT moments + a membership audit (kernel/ops); this drives the
// SAME cut through the real feature pipeline (two base cylinders + a Combine-Cut) and renders it offscreen
// through the actual viewport, proving the watertight B-rep actually reaches the screen as lit geometry —
// the visual half of the Live-tests discipline. A T-junction crack (the by-value ellipse bug) or an
// inside-out tunnel would drop the lit fraction or leave a visible hole; the PNG is saved for inspection.
package ui

import (
	"image"
	"image/png"
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

// writeBGRAPNG writes Vulkan BGRA8 surface bytes as an opaque RGBA PNG (mirrors native.encodeBGRAPNG,
// which is package-private) so the offscreen render can be inspected as an image.
func writeBGRAPNG(px []byte, width, height int, path string) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i+4 <= len(px); i += 4 {
		img.Pix[i+0], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = px[i+2], px[i+1], px[i+0], 255
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// capCrossPart returns a session whose active part holds the slice-1 cap-crossing cut, built through the
// real feature recompute: an r=3 h=10 target cylinder minus an oblique r=0.9 tool that exits the top cap.
func capCrossPart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "cap-cross.opd", true)
	if err != nil {
		t.Fatalf("add part: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sc := 1 / stdmath.Sqrt2
	target, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("target cylinder: %v", err)
	}
	tool, err := brep.SolidCylinder(math.P3(-6.5, 0, 2), math.V3(math.Scalar(sc), 0, math.Scalar(sc)), 0.9, 16)
	if err != nil {
		t.Fatalf("tool cylinder: %v", err)
	}
	feature.NewBaseFeatures(def.Features()).AddBase(target)
	feature.NewBaseFeatures(def.Features()).AddBase(tool)
	feature.NewModifyFeatures(def.Features()).AddCombine(0, 1, ops.Cut)
	def.Recompute()
	// A 3/4 view from above-front-right so both the top-cap elliptical hole and the wall entry show.
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(10, -8.5, 14), math.P3(0, 0, 5.5), math.V3(0, 0, 1)
	s.SetCamera(cam)
	return s
}

// TestCapCrossingCutRendersLive asserts the cap-crossing cut reaches the viewport as a large lit fraction
// (a crack or inverted tunnel would darken it) and writes a PNG for visual inspection.
func TestCapCrossingCutRendersLive(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()

	s := capCrossPart(t)
	lit := litFraction(win, s)
	t.Logf("cap-crossing cut lit fraction = %.4f", lit)
	// The empty-viewport baseline reads 0.0000 (TestViewportRendersPartAndAssemblyGeometry); this cut renders
	// 0.05–0.11 depending on the shared lighting/exposure state a prior test leaves (the slim cylinder's lit
	// pixels sit near litFraction's 175-luma cutoff, so a dimmer exposure halves the count). 0.03 discriminates
	// a rendered cut from a blank viewport with margin while tolerating that suite-context dimming.
	if lit < 0.03 {
		t.Errorf("cap-crossing cut rendered near-blank: lit fraction %.4f — the cut B-rep is not reaching the screen", lit)
	}

	px, w, h, ok := win.ReadbackViewport(0)
	if !ok {
		t.Fatal("viewport readback unavailable")
	}
	dir := t.TempDir() // default: a throwaway dir so a CI run never litters the tree
	if p := os.Getenv("OBK_SHOT_DIR"); p != "" {
		dir = p // set OBK_SHOT_DIR to keep the PNG for manual inspection
	}
	out := filepath.Join(dir, "cap-crossing-cut.png")
	if err := writeBGRAPNG(px, w, h, out); err != nil {
		t.Fatalf("write PNG: %v", err)
	}
	t.Logf("saved cap-crossing cut render to %s", out)
}
