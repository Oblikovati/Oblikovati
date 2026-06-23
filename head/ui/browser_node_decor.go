//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Browser-row decoration: the per-feature-type icon drawn before a node's label, and the
// in-place rename of a feature node. Both live here so browser_view.go stays focused on the
// tree structure (#1264).

// browserIconPx is the browser-row icon size in pixels. Sized at 200% of a tree row's text
// height so the per-feature glyphs read clearly (#1264).
const browserIconPx = 32

// drawNodeIcon draws the node's icon as a framed tile (the ribbon's button background, so the
// glyph reads against the dark tree) and then advances the cursor so the label the caller renders
// next sits to its right, vertically centred on the tile. Clicking the tile selects the node, like
// clicking its label. A node with no icon, or a glyph that fails to rasterize, draws nothing (the
// label still renders) so the tree degrades gracefully (#1264).
func drawNodeIcon(s *app.Session, n app.BrowserNode) {
	if n.Icon == "" || icons == nil {
		return
	}
	tex, ok := icons.texture(n.Icon, "", browserIconPx)
	if !ok {
		return
	}
	if native.ImageButton("##nodeicon", tex, browserIconPx, browserIconPx, identityTint) {
		s.SelectBrowserNode(n)
	}
	native.SameLine()
	m := native.Metrics()
	x, y := native.GetCursorScreenPos()
	native.SetCursorScreenPos(x, y+(browserIconPx+2*m.FramePadY-native.TextLineHeight())/2)
}

// browserRename holds the in-place feature rename: the node being edited (nil when none), the
// edit buffer, and a one-frame flag to grab keyboard focus when the field first appears.
type browserRenameState struct {
	target app.Selectable
	buf    [128]byte
	focus  bool
}

var browserRename browserRenameState

// beginRename opens the in-place editor for a renameable node, seeding the buffer with its
// current name and requesting focus on the next frame.
func beginRename(n app.BrowserNode) {
	browserRename.target = n.Select
	browserRename.focus = true
	setBuf(browserRename.buf[:], nodeName(n))
}

// renaming reports whether n is the node currently being renamed in place.
func renaming(n app.BrowserNode) bool {
	return browserRename.target != nil && browserRename.target == n.Select
}

// drawRenameField draws the in-place rename input for n in place of its label. Enter (or a
// committed focus-loss) applies the new name via app.Session.RenameFeature — which enforces a
// non-empty, document-unique name and keeps the stable id; Escape or an unedited focus-loss
// cancels. A rejected name (empty/duplicate) is logged and the edit is dropped.
func drawRenameField(s *app.Session, n app.BrowserNode) {
	first := browserRename.focus
	if first {
		native.SetKeyboardFocusHere()
		browserRename.focus = false
	}
	native.SetNextItemWidth(-1)
	enter := native.InputTextSubmit("##rename", browserRename.buf[:])
	switch {
	case enter || native.IsItemDeactivatedAfterEdit():
		applyRename(s, n)
	case !first && !native.IsItemActive():
		browserRename.target = nil // Escape or click-away with no committed edit: cancel
	}
}

// applyRename commits the buffer to the entity behind n, then closes the editor. The per-type
// backend keeps the stable id and enforces a non-empty, sibling-unique name.
func applyRename(s *app.Session, n app.BrowserNode) {
	if name := bufString(browserRename.buf[:]); name != "" {
		if err := renameNode(s, n, name); err != nil {
			fmt.Fprintf(os.Stderr, "rename: %v\n", err)
		}
	}
	browserRename.target = nil
}

// renameNode dispatches the rename to the session method for the node's handle type.
func renameNode(s *app.Session, n app.BrowserNode, name string) error {
	switch h := n.Select.(type) {
	case app.FeatureHandle:
		return s.RenameFeature(h.Feature, name)
	case app.SketchHandle:
		return s.RenameSketch(h.Sketch, name)
	case app.Sketch3DHandle:
		return s.RenameSketch3D(h.Sketch3D, name)
	case app.WorkPlaneHandle:
		return s.RenameWorkPlane(h.Plane, name)
	case app.WorkAxisHandle:
		return s.RenameWorkAxis(h.Axis, name)
	case app.WorkPointHandle:
		return s.RenameWorkPoint(h.Point, name)
	default:
		return nil
	}
}

// nodeName returns the editable name for a node — its handle's current name, falling back to the
// label (the label may carry a status badge, so prefer the raw name).
func nodeName(n app.BrowserNode) string {
	switch h := n.Select.(type) {
	case app.FeatureHandle:
		return h.Feature.Name()
	case app.SketchHandle:
		return h.Sketch.Name()
	case app.Sketch3DHandle:
		return h.Sketch3D.Name()
	case app.WorkPlaneHandle:
		return h.Plane.Name()
	case app.WorkAxisHandle:
		return h.Axis.Name()
	case app.WorkPointHandle:
		return h.Point.Name()
	default:
		return n.Label
	}
}

// isRenameableNode reports whether n is a node the in-place rename targets: a feature, a 2D/3D
// sketch, or a user work plane/axis/point. The grounded origin coordinate-system datums (the
// Origin folder) are excluded — their names are fixed (#1264).
func isRenameableNode(n app.BrowserNode) bool {
	switch h := n.Select.(type) {
	case app.FeatureHandle, app.SketchHandle, app.Sketch3DHandle:
		return true
	case app.WorkPlaneHandle:
		return !h.Plane.IsCoordinateSystemElement()
	case app.WorkAxisHandle:
		return !h.Axis.IsCoordinateSystemElement()
	case app.WorkPointHandle:
		return !h.Point.IsCoordinateSystemElement()
	default:
		return false
	}
}
