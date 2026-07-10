// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// treeSeeded tracks which tree controls have rendered once, so a node's Expanded flag seeds the
// disclosure state only on first render; afterwards imgui owns open/collapsed (a re-sent spec must
// not re-collapse what the user opened). Keyed windowID+"/"+controlID.
var treeSeeded = map[string]bool{}

// treeFirstUse reports (and records) whether key is rendering for the first time.
func treeFirstUse(key string) bool {
	if treeSeeded[key] {
		return false
	}
	treeSeeded[key] = true
	return true
}

// drawPanelTree renders a PanelTree: a hierarchy of selectable, expandable nodes. The disclosure
// arrow toggles a node (host-side, no event); a click on a node's label selects it and pushes the
// node ID to the add-in, which re-sends the spec (e.g. to populate a members table).
func drawPanelTree(s *app.Session, windowID string, control wire.PanelControlSpec) {
	firstUse := treeFirstUse(windowID + "/" + control.ID)
	for i := range control.Nodes {
		drawTreeNode(s, windowID, control.ID, control.Value, control.Nodes[i], firstUse)
	}
}

// drawTreeNode renders one node and recurses into its children. selected is the currently-selected
// node ID (for highlight). A leaf uses Selectable; a branch uses TreeNodeSelectable so the arrow
// expands while a label click selects.
func drawTreeNode(s *app.Session, windowID, controlID, selected string, node wire.TreeNode, firstUse bool) {
	if len(node.Children) == 0 {
		if native.Selectable(node.Label, node.ID == selected) {
			s.PanelValueChanged(windowID, controlID, node.ID)
		}
		return
	}
	if firstUse {
		native.SetNextItemOpen(node.Expanded, true)
	}
	open := native.TreeNodeSelectable(node.Label, node.ID == selected)
	if native.IsItemClicked(0) {
		s.PanelValueChanged(windowID, controlID, node.ID)
	}
	if !open {
		return
	}
	for i := range node.Children {
		drawTreeNode(s, windowID, controlID, selected, node.Children[i], firstUse)
	}
	native.TreePop()
}
