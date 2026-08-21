//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"
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
		// A member table taller than addInTableRows (8) with the selection BELOW the fold, so the
		// render exercises the #1933 scroll-into-view path (tableSelectionChanged → SetScrollHereY)
		// and not just the top-of-table case.
		client.PanelTable("members", []string{"designation", "d", "D", "B"}, offFoldMemberRows(), "62-10"),
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

// offFoldMemberRows returns a member table taller than addInTableRows, so the selection at index 10
// ("62-10") sits below the visible fold — the #1933 scenario the render must scroll into view.
func offFoldMemberRows() []wire.TableRow {
	rows := make([]wire.TableRow, 0, 12)
	for i := 0; i < 12; i++ {
		key := "62-" + strconv.Itoa(i)
		rows = append(rows, wire.TableRow{Key: key, Cells: []string{key, "10", "30", "9"}})
	}
	return rows
}
