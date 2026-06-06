// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/kernel/topo"
	"oblikovati/renderer"
)

// Viewport highlight of the active selection. A draw item is tagged with its body's
// ObjectID (renderer.BuildDrawList); recoloring the items whose id belongs to the
// selection paints the selected body cyan, so a browser pick lights up the 3D view (and
// a viewport pick, which sets the same selection, lights up the same body). Work-plane
// and sketch highlight are handled by their own overlays (planesOverlay / sketchOverlay).

// highlightSelection recolors the draw items belonging to the selected body/bodies. It
// returns the (in-place mutated) list for call-site convenience; the list is rebuilt
// each frame so the mutation is not observable across frames.
func highlightSelection(list renderer.DrawList, sel app.Selectable, partBodies []*topo.Body) renderer.DrawList {
	ids := bodyHighlightIDs(sel, partBodies)
	if len(ids) == 0 {
		return list
	}
	return recolorByObjectID(list, ids, selectionHighlight)
}

// bodyHighlightIDs returns the set of body ObjectIDs to highlight for a selection: the
// picked body itself, the body owning a picked face, or — since features do not yet map
// to the faces they created — every body of the part when a feature is selected.
func bodyHighlightIDs(sel app.Selectable, partBodies []*topo.Body) map[uint64]bool {
	switch h := sel.(type) {
	case app.BodyHandle:
		return singleID(h.Body)
	case app.FaceHandle:
		return singleID(h.Body)
	case app.FeatureHandle:
		ids := make(map[uint64]bool, len(partBodies))
		for _, b := range partBodies {
			ids[b.ID()] = true
		}
		return ids
	default:
		return nil
	}
}

// singleID is the one-element id set for a body, or empty when the body is nil.
func singleID(b *topo.Body) map[uint64]bool {
	if b == nil {
		return nil
	}
	return map[uint64]bool{b.ID(): true}
}

// recolorByObjectID paints every draw item whose ObjectID is in ids the given color.
func recolorByObjectID(list renderer.DrawList, ids map[uint64]bool, color [4]float32) renderer.DrawList {
	for i := range list.Items {
		if ids[list.Items[i].ObjectID] {
			list.Items[i].Color = color
		}
	}
	return list
}
