//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/compdef"
)

// clickSweepBody renders a panel body in a fixed-position probe window and walks a left and a centre
// column of full mouse clicks down it, so the panel's interactive controls (sliders set a value on
// click, "Done"/"Close"/"Save"/tab buttons fire) are actually exercised — the only way to cover the
// on-change/on-click handler bodies headlessly. It asserts nothing; a panic in any handler fails the
// test. (#1473 made these bodies render through the shared path, so they must stay click-safe.)
func clickSweepBody(win *native.Window, draw func()) {
	const x0, y0, w, h float32 = 40, 40, 360, 460
	frame := func() {
		win.BeginFrame()
		native.SetNextWindowPos(x0, y0)
		native.SetNextWindowSize(w, h)
		if native.Begin("##panel-probe") {
			draw()
		}
		native.End()
		win.EndFrame(0.1, 0.1, 0.1)
	}
	for gy := y0 + 18; gy < y0+h-6; gy += 18 {
		for _, gx := range []float32{x0 + 28, x0 + w*0.5} {
			native.InjectMousePos(gx, gy)
			frame()
			native.InjectMouseButton(native.MouseLeft, true)
			frame()
			native.InjectMouseButton(native.MouseLeft, false)
			frame()
		}
	}
}

// TestDockablePanelBodiesClickSafe sweeps real clicks over each registered panel body, covering the
// slider-change and button-click handlers that a passive render never reaches, and proving every
// panel survives interaction (no panic) now that they all render through the shared closable path.
func TestDockablePanelBodiesClickSafe(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = nil
	defer func(b, m, p bool) { showBrowser, showMaterials, showPreferences = b, m, p }(showBrowser, showMaterials, showPreferences)

	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "interact-test.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	_ = s.Workspace().SetActiveDocument(pd)
	defer clearBuf(namedViewNameBuf[:])
	copy(namedViewNameBuf[:], "v1\x00") // a non-empty name so a Save click captures a view

	// Bodies whose click handlers are otherwise unreachable. Materials is swept first/last so its
	// tab headers (Appearances/Physical) get activated and their content renders.
	bodies := []func(){
		func() { drawMaterialsBody(s) },
		func() { drawLightingBody(s) },
		func() { drawColorStylesBody(s) },
		func() { drawDisplaySettingsBody(s) },
		func() { drawUnitsSettingsBody(s) },
		func() { drawNamedViewsBody(s) },
		func() { drawHistoryBrowserBody(s) },
		func() { drawParametersBody(s) },
		func() { drawMaterialsBody(s) },
	}
	for _, draw := range bodies {
		clickSweepBody(win, draw)
	}
}
