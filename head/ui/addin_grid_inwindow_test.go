//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"testing"

	"oblikovati.org/api/client"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// jobEditDemoControls is a faithful slice of FreeCAD's CAM Job Edit window built from the new
// container vocabulary: a tab strip (Setup/General/Output) whose Setup pane stacks two grouped
// grids — a 2-column [auto label | 1fr field] Stock form with a full-width spanning button, and
// a 3-column Alignment button pad. It exercises tabs + group + grid + auto/fraction tracks +
// column span together.
func jobEditDemoControls() []wire.PanelControlSpec {
	labelField := []types.GridTrack{client.TrackAuto(), client.TrackFr(1)}
	stock := client.PanelGroup("stock", "Stock",
		client.PanelGrid("stockform", labelField, 8, 4,
			client.PanelLabel("ml", "Method"),
			client.PanelDropdown("method", "", []string{"Box", "Cylinder", "Extend bbox", "Existing"}, "Extend bbox"),
			client.PanelLabel("xl", "Ext X"), client.PanelTextBox("extx", "", "1.0 mm"),
			client.PanelLabel("yl", "Ext Y"), client.PanelTextBox("exty", "", "1.0 mm"),
			client.PanelLabel("zl", "Ext Z"), client.PanelTextBox("extz", "", "1.0 mm"),
			client.PlaceAt(client.PanelButton("refresh", "Refresh stock", "CAM.Refresh"), 0, 2),
		),
	)
	thirds := []types.GridTrack{client.TrackFr(1), client.TrackFr(1), client.TrackFr(1)}
	align := client.PanelGroup("align", "Origin & Orientation",
		client.PanelGrid("alignpad", thirds, 4, 4,
			client.PanelButton("xaxis", "X-Axis", ""), client.PanelButton("yaxis", "Y-Axis", ""), client.PanelButton("zaxis", "Z-Axis", ""),
			client.PanelButton("x0", "X=0", ""), client.PanelButton("y0", "Y=0", ""), client.PanelButton("z0", "Z=0", ""),
		),
	)
	return []wire.PanelControlSpec{
		client.PanelTabs("jobtabs",
			client.PanelTab("Setup", stock, align),
			client.PanelTab("General", client.PanelLabel("g", "Label · Model · Description · Machine")),
			client.PanelTab("Output", client.PanelLabel("o", "Output file · Post Processor · WCS")),
		),
	}
}

// TestInWindowGridPanelScreenshot renders the demo Job Edit panel full-window in the live head
// and writes a PNG of the whole window so the grid/group/tabs layout can be inspected. It also
// proves the tree passes validation and renders without a crash. Skips without Vulkan.
func TestInWindowGridPanelScreenshot(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := app.NewSession()
	spec := wire.DockableWindowSpec{ID: "job.edit", Title: "CAM Job Edit", Visible: true, Controls: jobEditDemoControls()}
	if err := s.SetDockableWindow(spec); err != nil {
		t.Fatalf("SetDockableWindow (validation): %v", err)
	}

	for range 3 { // settle immediate-mode layout + tab selection
		win.BeginFrame()
		native.SetNextWindowPos(0, 0)
		native.SetNextWindowSize(inWinW, inWinH)
		visible, _ := native.BeginClosable("CAM Job Edit###addin-job.edit")
		if visible {
			drawControlList(s, "job.edit", spec.Controls)
		}
		native.End()
		win.EndFrame(0.12, 0.12, 0.13)
	}

	out := os.Getenv("GRID_SHOT")
	if out == "" {
		out = "/tmp/grid_panel.png"
	}
	if err := win.SaveWindowPNG(out); err != nil {
		t.Fatalf("SaveWindowPNG: %v", err)
	}
	if fi, err := os.Stat(out); err != nil || fi.Size() == 0 {
		t.Fatalf("screenshot not written: %v", err)
	}
	t.Logf("grid panel screenshot written to %s", out)
}
