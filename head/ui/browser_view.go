//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The model browser tree (Inventor's browser): each frame it reads app.BuildBrowser and
// draws it with Dear ImGui. Selecting a node syncs the 3D view (and a viewport pick syncs
// the tree back, since both set the same Session selection); right-clicking a node opens
// its app.BrowserMenu; when the selection changes the matching node is scrolled into view.

// browserSelectionSync remembers the selection from the previous frame so the tree scrolls
// to the synced node only when the selection actually changed — scrolling every frame
// would fight the user's own scrolling.
type browserSelectionSync struct{ last app.Selectable }

var browserSync browserSelectionSync

// browserNodeSeq numbers nodes in pre-order each frame so every row gets a unique Dear
// ImGui id (see drawNode). Reset at the top of each browser draw.
var browserNodeSeq int

// showBrowser toggles the Model browser (View ▸ Model Browser). It is head-local UI state and
// defaults on — the model tree is the primary browsing surface — but it is now closable and
// re-openable from the View menu like any other dockable window (Oblikovati#1473).
var showBrowser = true

// drawBrowserBody renders the model browser content: the model tree from the active document, or a
// tab bar of the tree plus any add-in browser panes (M05-F03 #256). It records the selection so the
// next frame can detect a change for scroll-sync, and brackets the tree-walk with the frame timer
// (M34-F3). The shared dockable-panel path (drawDockablePanel) owns the window chrome (#1473).
func drawBrowserBody(s *app.Session) {
	start := frameClock()
	if panes := s.BrowserPanes().List(); len(panes) > 0 {
		drawBrowserPaneTabs(s, panes)
	} else {
		drawModelTree(s)
	}
	recordBrowser(start)
}

// drawModelTree renders the built-in document tree (the Model pane's content).
func drawModelTree(s *app.Session) {
	browserNodeSeq = 0
	drawNode(s, app.BuildBrowser(s))
	browserSync.last = s.Selection().First()
}

// drawBrowserPaneTabs renders the Model tree and the add-in panes as tabs.
func drawBrowserPaneTabs(s *app.Session, panes []wire.BrowserPaneSpec) {
	if !native.BeginTabBar("##browser-panes") {
		return
	}
	if native.BeginTabItem("Model") {
		drawModelTree(s)
		native.EndTabItem()
	}
	for _, pane := range panes {
		if native.BeginTabItem(pane.Title + "##" + pane.ID) {
			for _, n := range pane.Nodes {
				drawAddInPaneNode(s, pane.ID, n)
			}
			native.EndTabItem()
		}
	}
	native.EndTabBar()
}

// drawAddInPaneNode renders one declared node: a leaf as a clickable row, a parent as a
// tree node. Every gesture (select, double-click, expand, collapse) is reported to the
// session so the owning add-in receives it as a browser.node event — the #256 acceptance:
// an add-in node that responds to clicks.
func drawAddInPaneNode(s *app.Session, paneID string, n wire.BrowserNodeSpec) {
	native.PushID("addin-node-" + n.ID)
	defer native.PopID()
	if len(n.Children) == 0 {
		drawAddInNodeIcon(s, paneID, n)
		if native.Selectable(n.Label, false) {
			_ = s.ActivateBrowserPaneNode(paneID, n.ID, app.BrowserGestureSelect)
		}
		reportDoubleClick(s, paneID, n.ID)
		drawAddInNodeMenu(s, paneID, n)
		return
	}
	drawAddInPaneBranch(s, paneID, n)
}

// drawAddInNodeIcon rasterises a node's inline themed glyph (BrowserNodeSpec.IconSVG) beside the
// label, like a document-tree node icon — clicking it selects the node. Nodes without a glyph
// draw nothing. The cache key is the node id (its svg also keys the texture, so re-skins update).
func drawAddInNodeIcon(s *app.Session, paneID string, n wire.BrowserNodeSpec) {
	if n.IconSVG == "" || icons == nil {
		return
	}
	tex, ok := icons.texture("addin/"+paneID+"/"+n.ID, n.IconSVG, browserIconPx)
	if !ok {
		return
	}
	if native.ImageButton("##nodeicon", tex, browserIconPx, browserIconPx, identityTint) {
		_ = s.ActivateBrowserPaneNode(paneID, n.ID, app.BrowserGestureSelect)
	}
	native.SameLine()
	m := native.Metrics()
	x, y := native.GetCursorScreenPos()
	native.SetCursorScreenPos(x, y+(browserIconPx+2*m.FramePadY-native.TextLineHeight())/2)
}

// drawAddInNodeMenu opens the node's right-click context menu (BrowserNodeSpec.Menu); choosing an
// item reports it to the owning add-in as a "menu" gesture carrying the item id. Nodes with no
// menu draw nothing.
func drawAddInNodeMenu(s *app.Session, paneID string, n wire.BrowserNodeSpec) {
	if len(n.Menu) == 0 {
		return
	}
	if !native.BeginPopupContextItem("##addin-menu-" + n.ID) {
		return
	}
	for _, it := range n.Menu {
		if native.MenuItemEx(it.Label, "", !it.Disabled) {
			_ = s.ActivateBrowserPaneNodeMenu(paneID, n.ID, it.ID)
		}
	}
	native.EndPopup()
}

// drawAddInPaneBranch renders a parent node, reporting expand/collapse and select
// gestures, and recurses into its children while open.
func drawAddInPaneBranch(s *app.Session, paneID string, n wire.BrowserNodeSpec) {
	drawAddInNodeIcon(s, paneID, n) // before SetNextItemOpen, so the open-state targets the TreeNode, not the icon
	if n.Expanded {
		native.SetNextItemOpen(true, true)
	}
	open := native.TreeNode(n.Label)
	if native.IsItemToggledOpen() {
		gesture := app.BrowserGestureCollapse
		if open {
			gesture = app.BrowserGestureExpand
		}
		_ = s.ActivateBrowserPaneNode(paneID, n.ID, gesture)
	}
	if native.IsItemClicked(native.MouseLeft) {
		_ = s.ActivateBrowserPaneNode(paneID, n.ID, app.BrowserGestureSelect)
	}
	reportDoubleClick(s, paneID, n.ID)
	drawAddInNodeMenu(s, paneID, n)
	if !open {
		return
	}
	for _, child := range n.Children {
		drawAddInPaneNode(s, paneID, child)
	}
	native.TreePop()
}

// reportDoubleClick reports a double-click gesture on the last item.
func reportDoubleClick(s *app.Session, paneID, nodeID string) {
	if native.IsItemHovered() && native.IsMouseDoubleClicked(native.MouseLeft) {
		_ = s.ActivateBrowserPaneNode(paneID, nodeID, app.BrowserGestureDouble)
	}
}

// browserClipThreshold is the child count above which a node's (all-leaf) children are drawn through
// an ImGuiListClipper, so a body/occurrence list with thousands of rows only makes cgo calls for the
// rows on screen — not all of them every frame (M34-F3). Below it the clipper's overhead isn't worth
// it and the rows draw normally.
const browserClipThreshold = 64

// drawChildren renders a branch's children, virtualizing any long CONTIGUOUS RUN of leaf siblings
// with a clipper and walking branches (and short runs) normally. Clipping runs — not just whole
// all-leaf lists — means a flat assembly's 10k occurrences are virtualized even though they sit
// beside the Origin/Parameters branches under the root (M34-F3). A run is uniform-height (every row
// is a leaf), which is what the clipper requires; branches have variable height (an open branch is
// taller) so they stay recursive. Leaves are contiguous in pre-order, so a run's ids stay correct.
func drawChildren(s *app.Session, children []app.BrowserNode) {
	for i := 0; i < len(children); {
		if len(children[i].Children) > 0 { // a branch: recurse (advances browserNodeSeq by its subtree)
			drawNode(s, children[i])
			i++
			continue
		}
		j := i
		for j < len(children) && len(children[j].Children) == 0 {
			j++
		}
		drawLeafRun(s, children[i:j])
		i = j
	}
}

// drawLeafRun draws a contiguous run of leaf siblings: clipped when long enough to be worth it,
// else one by one (the clipper has fixed per-list overhead not worth paying for a few rows).
func drawLeafRun(s *app.Session, run []app.BrowserNode) {
	if len(run) >= browserClipThreshold {
		drawClippedLeaves(s, run)
		return
	}
	for k := range run {
		drawNode(s, run[k])
	}
}

// leafRunLengths returns the lengths of the maximal contiguous leaf runs in children, in order —
// the structure drawChildren virtualizes. Exposed for testing the run partition without a window.
func leafRunLengths(children []app.BrowserNode) []int {
	var runs []int
	for i := 0; i < len(children); {
		if len(children[i].Children) > 0 {
			i++
			continue
		}
		j := i
		for j < len(children) && len(children[j].Children) == 0 {
			j++
		}
		runs = append(runs, j-i)
		i = j
	}
	return runs
}

// drawClippedLeaves draws a wide leaf list through the clipper. It reserves exactly the per-row
// ImGui ids the recursive walk would (one per leaf, contiguous from browserNodeSeq), force-includes
// the selected row so scroll-to-selection still works when it is off-screen, and advances
// browserNodeSeq past the whole list (each leaf consumes one id and has no descendants), so sibling
// nodes after this branch keep the ids they had before.
func drawClippedLeaves(s *app.Session, leaves []app.BrowserNode) {
	base := browserNodeSeq
	native.ClipperBegin(len(leaves))
	if i := selectedLeafIndex(s, leaves); i >= 0 {
		native.ClipperIncludeItem(i) // force-submit the selection so its SetScrollHereY runs (after Begin, before Step)
	}
	for native.ClipperStep() {
		lo, hi := native.ClipperRange()
		for i := lo; i < hi; i++ {
			native.PushIDInt(base + 1 + i)
			drawLeafRow(s, leaves[i])
			native.PopID()
		}
	}
	native.ClipperEnd()
	browserNodeSeq = base + len(leaves)
}

// drawLeafRow draws one leaf: a selectable row (selection + double-click edit + context menu +
// scroll-into-view) or a plain bullet.
func drawLeafRow(s *app.Session, n app.BrowserNode) {
	if n.Select != nil {
		drawSelectableNode(s, n)
		return
	}
	native.BulletText(n.Label)
}

// selectedLeafIndex returns the index of the leaf that is the current selection, or -1 — so the
// clipper can force-submit it and its SetScrollHereY runs even when it is scrolled out of view.
func selectedLeafIndex(s *app.Session, leaves []app.BrowserNode) int {
	current := s.Selection().First()
	if current == nil {
		return -1
	}
	for i := range leaves {
		if leaves[i].Select == current {
			return i
		}
	}
	return -1
}

// drawNode renders a browser node and its children: a selectable node as a clickable,
// highlightable row; a branch as a collapsible tree node; a plain leaf as a bullet row.
//
// Each node pushes a unique id (its pre-order number) so two rows with the SAME display
// label — several imported bodies, or two features of the same kind — never collide on one
// Dear ImGui id, which would break the row's selection, expansion, and context menu. The
// label is still what's shown; only the id is disambiguated.
func drawNode(s *app.Session, n app.BrowserNode) {
	browserNodeSeq++
	native.PushIDInt(browserNodeSeq)
	defer native.PopID()
	switch {
	case n.Select != nil && len(n.Children) > 0:
		drawSelectableBranchNode(s, n)
	case n.Select != nil:
		drawSelectableNode(s, n)
	case len(n.Children) == 0:
		native.BulletText(n.Label)
	default:
		drawBranchNode(s, n)
	}
}

// drawSelectableBranchNode draws a node that is both selectable and a parent — a feature row
// that nests its consumed sketch. The disclosure arrow toggles it open; a click on the label
// selects it (and a double-click opens its editor), so expansion and selection don't fight.
func drawSelectableBranchNode(s *app.Session, n app.BrowserNode) {
	if renaming(n) {
		drawRenameField(s, n)
		return
	}
	current := s.Selection().First()
	drawNodeIcon(s, n)
	open := native.TreeNodeSelectable(n.Label, current == n.Select)
	if native.IsItemClicked(native.MouseLeft) {
		s.SelectBrowserNode(n)
	}
	openEditOnDoubleClick(s, n)
	drawNodeMenu(s, n)
	if current != nil && n.Select == current && current != browserSync.last {
		native.SetScrollHereY()
	}
	if !open {
		return
	}
	drawChildren(s, n.Children)
	native.TreePop()
}

// drawSelectableNode draws a clickable row that selects the node, offers its context menu,
// and scrolls itself into view when it is the node the selection just changed to.
func drawSelectableNode(s *app.Session, n app.BrowserNode) {
	if renaming(n) {
		drawRenameField(s, n)
		return
	}
	current := s.Selection().First()
	drawNodeIcon(s, n)
	if native.Selectable(n.Label, current == n.Select) {
		s.SelectBrowserNode(n)
	}
	openEditOnDoubleClick(s, n)
	drawNodeMenu(s, n)
	if current != nil && n.Select == current && current != browserSync.last {
		native.SetScrollHereY()
	}
}

// openEditOnDoubleClick runs the double-clicked node's activation (Inventor's edit-on-double
// -click) through the NodeActivatable capability: a feature opens its parameter editor, a sketch
// re-enters the sketch environment, an occurrence opens the placed component, a representation /
// model state activates, a drawing view edits its settings. head/ui invokes the capability rather
// than switching on concrete handle types (#1630); a node with nothing to activate does nothing.
func openEditOnDoubleClick(s *app.Session, n app.BrowserNode) {
	if !native.IsItemHovered() || !native.IsMouseDoubleClicked(native.MouseLeft) {
		return
	}
	if a, ok := n.Select.(app.NodeActivatable); ok {
		a.Activate(s)
	}
}

// drawBranchNode draws a collapsible folder and recurses into its children when open. A
// branch may still carry a context menu (e.g. a work-plane row that owns children later).
func drawBranchNode(s *app.Session, n app.BrowserNode) {
	if n.Kind == "document" {
		native.SetNextItemOpen(true, true)
	}
	open := native.TreeNode(n.Label)
	drawNodeMenu(s, n)
	if !open {
		return
	}
	drawChildren(s, n.Children)
	native.TreePop()
}

// drawNodeMenu opens the node's right-click context menu (if it has any entries) and runs
// the action behind a clicked item. Nodes whose kind has no menu draw nothing.
func drawNodeMenu(s *app.Session, n app.BrowserNode) {
	items := app.BrowserMenuFor(s, n) // built-ins + add-in injections (M05-F12)
	renameable := isRenameableNode(n)
	if len(items) == 0 && !renameable {
		return
	}
	if !native.BeginPopupContextItem("##menu-" + n.Kind + "/" + n.Label) {
		return
	}
	for _, it := range items {
		if native.MenuItemEx(it.Label, "", it.Enabled) && it.Invoke != nil {
			if err := it.Invoke(s); err != nil {
				fmt.Fprintf(os.Stderr, "browser %q: %v\n", it.Label, err)
			}
		}
	}
	if renameable && native.MenuItemEx("Rename", "", true) {
		beginRename(n) // in-place edit; commit enforces a document-unique name (#1264)
	}
	native.EndPopup()
}
