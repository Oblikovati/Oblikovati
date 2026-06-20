//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
)

// pointCloudRibbonPanel extracts the 3D Model ▸ Point Cloud panel from the built ribbon.
func pointCloudRibbonPanel(t *testing.T, s *app.Session) app.RibbonPanel {
	t.Helper()
	tab, ok := app.BuildRibbon(s).Tab("3D Model")
	if !ok {
		t.Fatal("no 3D Model tab")
	}
	panel, ok := tab.Panel("Point Cloud")
	if !ok {
		t.Fatal("no Point Cloud panel")
	}
	return panel
}

// TestInWindowPointCloudPanelButtons renders the Point Cloud ribbon panel on its own (the full 3D
// Model tab is far wider than the captured window) with a cloud selected, and captures it — the
// visual confirmation that its buttons (Import / Fit Work Plane / Work Point / Crop Box) render
// with icons and are enabled, hence clickable (#645).
func TestInWindowPointCloudPanelButtons(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "ribbon.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, _ := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(0, 0, 5), math.P3(2, 0, 5), math.P3(0, 2, 5)})
	s.Select(app.PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc}) // enable Fit Work Plane / Crop Box

	for i := 0; i < 4; i++ {
		win.BeginFrame()
		DrawChrome(win, s) // binds the icon cache and draws the chrome
		native.SetNextWindowPos(40, 80)
		native.SetNextWindowSize(380, 150)
		if native.Begin("3D Model > Point Cloud") {
			drawPanel(pointCloudRibbonPanel(t, s), 200)
		}
		native.End()
		win.EndFrame(0.12, 0.12, 0.14)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-panel.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
