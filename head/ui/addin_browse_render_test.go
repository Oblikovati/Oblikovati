//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/api/client"
	"oblikovati.org/api/wire"
	"oblikovati.org/head/internal/native"
)

// TestAddInPanelTreeTableRender renders a dockable-panel control list holding the two new
// PanelTree/PanelTable widgets (plus a text box and a check box) through real frames, so the
// imgui draw code actually executes and is credited by the xvfb+lavapipe CI head job. The panel
// draw layer calls live imgui and cannot be covered by headless per-package tests — this mirrors
// parameter_input_render_test.go's in-window approach (#1519). It asserts nothing about pixels; it
// guards that the widgets draw without panicking and exercises the tree recursion (an expanded
// branch, a nested sub-branch, and a selected leaf), the scrolling member table (header + rows),
// and BOTH the first-render (Expanded seed via SetNextItemOpen) and the subsequent-render path
// (treeFirstUse == false). Skips cleanly with no display/Vulkan.
func TestAddInPanelTreeTableRender(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	icons = newIconCache(win)
	s := framedSession()

	const windowID = "addin-browse"
	delete(treeSeeded, windowID+"/catalog") // force the first-render Expanded-seed path

	controls := []wire.PanelControlSpec{
		client.PanelTextBox("search", "Search", "62"),
		client.PanelCheckBox("flag", "Flag", true),
		client.PanelTree("catalog", []wire.TreeNode{{
			ID: "bearings", Label: "Bearings", Expanded: true,
			Children: []wire.TreeNode{
				{ID: "fam-deep-groove", Label: "Deep Groove"}, // a selected leaf (Value below)
				{ID: "angular", Label: "Angular", Children: []wire.TreeNode{
					{ID: "fam-ang", Label: "7200"}, // a nested branch → leaf, so TreePop recurses
				}},
			},
		}}, "fam-deep-groove"),
		client.PanelTable("members", []string{"designation", "d", "D", "B"}, []wire.TableRow{
			{Key: "6200", Cells: []string{"6200", "10", "30", "9"}},
			{Key: "6202", Cells: []string{"6202", "15", "35", "11"}}, // selected row (Value below)
		}, "6202"),
	}

	frame := func() {
		win.BeginFrame()
		if native.Begin("##addin-browse-render") {
			drawControlList(s, windowID, controls)
		}
		native.End()
		win.EndFrame(0.1, 0.1, 0.1)
	}
	frame() // first render seeds the tree's open state via SetNextItemOpen(..., firstUse=true)
	frame() // second render takes the treeFirstUse == false path
}
