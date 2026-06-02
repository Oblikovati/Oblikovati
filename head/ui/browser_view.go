//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
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

// drawBrowser renders the model browser tree from the active document, then records the
// selection so the next frame can detect a change for scroll-sync.
func drawBrowser(s *app.Session) {
	if native.Begin("Model") {
		drawNode(s, app.BuildBrowser(s))
		browserSync.last = s.Selection().First()
	}
	native.End()
}

// drawNode renders a browser node and its children: a selectable node as a clickable,
// highlightable row; a branch as a collapsible tree node; a plain leaf as a bullet row.
func drawNode(s *app.Session, n app.BrowserNode) {
	switch {
	case n.Select != nil:
		drawSelectableNode(s, n)
	case len(n.Children) == 0:
		native.BulletText(n.Label)
	default:
		drawBranchNode(s, n)
	}
}

// drawSelectableNode draws a clickable row that selects the node, offers its context menu,
// and scrolls itself into view when it is the node the selection just changed to.
func drawSelectableNode(s *app.Session, n app.BrowserNode) {
	current := s.Selection().First()
	if native.Selectable(n.Label, current == n.Select) {
		s.SelectBrowserNode(n)
	}
	openFeatureEditOnDoubleClick(s, n)
	drawNodeMenu(s, n)
	if current != nil && n.Select == current && current != browserSync.last {
		native.SetScrollHereY()
	}
}

// openFeatureEditOnDoubleClick re-opens a feature node's parameter editor when its row is
// double-clicked (Inventor's edit-on-double-click). Only feature nodes carry an editor;
// double-clicking other nodes does nothing.
func openFeatureEditOnDoubleClick(s *app.Session, n app.BrowserNode) {
	if !native.IsItemHovered() || !native.IsMouseDoubleClicked(native.MouseLeft) {
		return
	}
	if h, ok := n.Select.(app.FeatureHandle); ok {
		s.BeginEditFeature(h)
	}
}

// drawBranchNode draws a collapsible folder and recurses into its children when open. A
// branch may still carry a context menu (e.g. a work-plane row that owns children later).
func drawBranchNode(s *app.Session, n app.BrowserNode) {
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
	items := app.BrowserMenu(n)
	if len(items) == 0 {
		return
	}
	if !native.BeginPopupContextItem("##menu-" + n.Kind + "/" + n.Label) {
		return
	}
	for _, it := range items {
		if native.MenuItem(it.Label) && it.Invoke != nil {
			if err := it.Invoke(s); err != nil {
				fmt.Fprintf(os.Stderr, "browser %q: %v\n", it.Label, err)
			}
		}
	}
	native.EndPopup()
}
