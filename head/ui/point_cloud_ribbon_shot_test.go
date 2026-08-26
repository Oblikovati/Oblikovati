//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

// pointCloudRibbonPanel extracts the Surfaces & Mesh ▸ Point Cloud panel — the single consolidated
// panel holding both the tool buttons and the folded-in display controls.
func pointCloudRibbonPanel(t *testing.T, s *app.Session) app.RibbonPanel {
	t.Helper()
	tab, ok := app.BuildRibbon(s).Tab("Surfaces & Mesh")
	if !ok {
		t.Fatal("no Surfaces & Mesh tab")
	}
	panel, ok := tab.Panel("Point Cloud")
	if !ok {
		t.Fatal("no Point Cloud panel")
	}
	return panel
}

// TestInWindowPointCloudPanelButtons renders the consolidated Point Cloud ribbon panel on its own
// (the full Surfaces & Mesh tab is far wider than the captured window) with an intensity cloud
// selected, and captures it — the visual confirmation that the tool buttons, the size/display-mode/
// density controls, and the intensity ramp all fit in one grid panel (#645).
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
	pc, _ := def.PointClouds().AddWithSamples("Scan", "s.xyz", rid, []pointcloud.PointSample{
		{Point: math.P3(0, 0, 5), HasIntensity: true, Intensity: 0},
		{Point: math.P3(2, 0, 5), HasIntensity: true, Intensity: 50},
		{Point: math.P3(0, 2, 5), HasIntensity: true, Intensity: 100},
	})
	pc.SetDisplayMode(types.PointCloudDisplayModeIntensity)
	s.Select(app.PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc}) // enable Fit Work Plane / Crop Box

	for range 4 {
		win.BeginFrame()
		DrawChrome(win, s) // binds the icon cache and draws the chrome
		native.SetNextWindowPos(40, 80)
		native.SetNextWindowSize(460, 210)
		if native.Begin("Surfaces & Mesh > Point Cloud") {
			m := native.Metrics()
			_, gridTop := native.GetCursorScreenPos()
			labelY := gridTop + ribbonGridHeight(m) + m.ItemSpacingY
			drawPanel(s, pointCloudRibbonPanel(t, s), labelY)
		}
		native.End()
		win.EndFrame(0.12, 0.12, 0.14)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-panel.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
