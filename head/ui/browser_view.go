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

// drawBrowser renders the model browser tree from the active document, then records the
// selection so the next frame can detect a change for scroll-sync. When add-ins have
// declared browser panes (M05-F03 #256), the window becomes a tab bar: the Model tree
// first, then one tab per add-in pane.
func drawBrowser(s *app.Session) {
	if native.Begin("Model") {
		if panes := s.BrowserPanes().List(); len(panes) > 0 {
			drawBrowserPaneTabs(s, panes)
		} else {
			drawModelTree(s)
		}
	}
	native.End()
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
		if native.Selectable(n.Label, false) {
			_ = s.ActivateBrowserPaneNode(paneID, n.ID, app.BrowserGestureSelect)
		}
		reportDoubleClick(s, paneID, n.ID)
		return
	}
	drawAddInPaneBranch(s, paneID, n)
}

// drawAddInPaneBranch renders a parent node, reporting expand/collapse and select
// gestures, and recurses into its children while open.
func drawAddInPaneBranch(s *app.Session, paneID string, n wire.BrowserNodeSpec) {
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
	current := s.Selection().First()
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
	for _, child := range n.Children {
		drawNode(s, child)
	}
	native.TreePop()
}

// drawSelectableNode draws a clickable row that selects the node, offers its context menu,
// and scrolls itself into view when it is the node the selection just changed to.
func drawSelectableNode(s *app.Session, n app.BrowserNode) {
	current := s.Selection().First()
	if native.Selectable(n.Label, current == n.Select) {
		s.SelectBrowserNode(n)
	}
	openEditOnDoubleClick(s, n)
	drawNodeMenu(s, n)
	if current != nil && n.Select == current && current != browserSync.last {
		native.SetScrollHereY()
	}
}

// openEditOnDoubleClick opens the edit mode for the double-clicked node (Inventor's
// edit-on-double-click), dispatching by node type: a feature opens its parameter editor, a
// sketch re-enters the sketch environment, and a user work plane opens its redefine tool.
// A node with nothing to edit (the origin frame, a non-editable feature) does nothing.
func openEditOnDoubleClick(s *app.Session, n app.BrowserNode) {
	if !native.IsItemHovered() || !native.IsMouseDoubleClicked(native.MouseLeft) {
		return
	}
	switch h := n.Select.(type) {
	case app.FeatureHandle:
		s.BeginEditFeature(h)
	case app.AssemblyFeatureHandle:
		s.BeginEditAssemblyFeature(h) // edit a committed assembly machining feature (#766)
	case app.SketchHandle:
		s.BeginEditSketch(h)
	case app.WorkPlaneHandle:
		s.BeginEditWorkPlane(h)
	case app.OccurrenceHandle:
		_ = s.OpenOccurrenceDocument(h.Occurrence) // open the placed component in a tab (#764)
	case app.RepresentationHandle:
		_ = s.ActivateRepresentation(h) // double-click activates a representation (M12-F04)
	case app.ModelStateHandle:
		_ = s.ActivateModelState(h) // double-click switches the model state (M12-F04)
	case app.DrawingViewHandle:
		s.BeginEditDrawingView(h) // double-click a drawing view edits its settings (M14-F02)
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
	for _, child := range n.Children {
		drawNode(s, child)
	}
	native.TreePop()
}

// drawNodeMenu opens the node's right-click context menu (if it has any entries) and runs
// the action behind a clicked item. Nodes whose kind has no menu draw nothing.
func drawNodeMenu(s *app.Session, n app.BrowserNode) {
	items := app.BrowserMenuFor(s, n) // built-ins + add-in injections (M05-F12)
	if len(items) == 0 {
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
	native.EndPopup()
}
