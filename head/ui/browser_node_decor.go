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

// browserIconPx is the browser-row icon size in pixels (matches a tree row's text height).
const browserIconPx = 16

// drawNodeIcon draws the node's icon glyph followed by SameLine, so the label that the caller
// renders next sits to its right. A node with no icon, or a glyph that fails to rasterize, draws
// nothing (the label still renders) so the tree degrades gracefully.
func drawNodeIcon(n app.BrowserNode) {
	if n.Icon == "" || icons == nil {
		return
	}
	tex, ok := icons.texture(n.Icon, "", browserIconPx)
	if !ok {
		return
	}
	native.Image(tex, browserIconPx, browserIconPx)
	native.SameLine()
}

// browserRename holds the in-place feature rename: the node being edited (nil when none), the
// edit buffer, and a one-frame flag to grab keyboard focus when the field first appears.
type browserRenameState struct {
	target app.Selectable
	buf    [128]byte
	focus  bool
}

var browserRename browserRenameState

// beginRename opens the in-place editor for a feature node, seeding the buffer with its current
// name and requesting focus on the next frame.
func beginRename(n app.BrowserNode) {
	browserRename.target = n.Select
	browserRename.focus = true
	setBuf(browserRename.buf[:], featureNodeName(n))
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

// applyRename commits the buffer to the feature behind n, then closes the editor.
func applyRename(s *app.Session, n app.BrowserNode) {
	if h, ok := n.Select.(app.FeatureHandle); ok {
		if name := bufString(browserRename.buf[:]); name != "" {
			if err := s.RenameFeature(h.Feature, name); err != nil {
				fmt.Fprintf(os.Stderr, "rename feature: %v\n", err)
			}
		}
	}
	browserRename.target = nil
}

// featureNodeName returns the editable feature name for a node — its handle's current name,
// falling back to the label (the label may carry a status badge, so prefer the raw name).
func featureNodeName(n app.BrowserNode) string {
	if h, ok := n.Select.(app.FeatureHandle); ok {
		return h.Feature.Name()
	}
	return n.Label
}

// isRenameableFeature reports whether n is a feature node (the only node kind the in-place
// rename currently targets — its backend enforces document-unique names, #1264).
func isRenameableFeature(n app.BrowserNode) bool {
	_, ok := n.Select.(app.FeatureHandle)
	return ok
}
