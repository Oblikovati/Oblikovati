//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// camPaneSvg is a sentinel-coloured glyph (the ribbon convention: #00ff00 backplate, #000
// primary, #ff0000 accent) so the rasterised node icon is themed like a document icon.
const camPaneSvg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="1" y="1" width="22" height="22" rx="4" fill="#00ff00" stroke="none"/><rect x="5" y="7" width="10" height="10" rx="1"/><path d="M17 5 V9 M15 7 H19" stroke="#ff0000"/></svg>`

// camDemoPane is a faithful CAM Job tree: a Job node with icons and per-node right-click menus,
// nesting Stock and an Operations branch.
func camDemoPane() wire.BrowserPaneSpec {
	menu := []wire.BrowserMenuItem{{ID: "edit", Label: "Edit…"}, {ID: "del", Label: "Delete"}}
	return wire.BrowserPaneSpec{ID: "cam", Title: "CAM", Nodes: []wire.BrowserNodeSpec{{
		ID: "job", Label: "Job", IconSVG: camPaneSvg, Expanded: true, Menu: menu,
		Children: []wire.BrowserNodeSpec{
			{ID: "stock", Label: "Stock", IconSVG: camPaneSvg, Menu: menu},
			{ID: "ops", Label: "Operations", IconSVG: camPaneSvg, Expanded: true, Children: []wire.BrowserNodeSpec{
				{ID: "profile", Label: "Profile", IconSVG: camPaneSvg, Menu: menu},
				{ID: "pocket", Label: "Pocket", IconSVG: camPaneSvg, Menu: menu},
			}},
		},
	}}}
}

// TestInWindowCamBrowserPaneScreenshot renders the CAM browser pane (icons + nested tree) full
// window and writes a PNG so the document-style node icons can be inspected. Proves the pane
// renders without a crash. Skips without Vulkan.
func TestInWindowCamBrowserPaneScreenshot(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := app.NewSession()
	if err := s.BrowserPanes().Set(camDemoPane()); err != nil {
		t.Fatalf("Set pane: %v", err)
	}
	icons = newIconCache(win) // bind the icon cache to this fresh window/context so glyphs rasterise

	for i := 0; i < 3; i++ {
		win.BeginFrame()
		native.SetNextWindowPos(0, 0)
		native.SetNextWindowSize(inWinW, inWinH)
		if vis, _ := native.BeginClosable("CAM##browser"); vis {
			browserNodeSeq = 0
			for _, n := range camDemoPane().Nodes { // render the CAM pane nodes directly (skip the Model tab)
				drawAddInPaneNode(s, "cam", n)
			}
		}
		native.End()
		win.EndFrame(0.12, 0.12, 0.13)
	}

	out := os.Getenv("CAM_PANE_SHOT")
	if out == "" {
		out = "/tmp/cam_pane.png"
	}
	if err := win.SaveWindowPNG(out); err != nil {
		t.Fatalf("SaveWindowPNG: %v", err)
	}
	if fi, err := os.Stat(out); err != nil || fi.Size() == 0 {
		t.Fatalf("screenshot not written: %v", err)
	}
	t.Logf("CAM browser pane screenshot written to %s", out)
}
