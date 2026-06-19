// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/clientgraphics"
	"oblikovati.org/renderer"
)

// Client/preview graphics (M05-F05) + frame assembly. The session collects overlay
// primitives (persistent client graphics) and a tool may contribute a transient
// preview; RenderFrame assembles the active part's geometry plus these into one draw
// list and submits it to a backend. With the NullBackend this is fully headless, so
// a test asserts exactly what the viewport would show after a UI action.

// Previewable is an optional Tool capability: a tool that can show a live preview of
// its result before commit implements it (Inventor's in-canvas preview).
type Previewable interface {
	Preview(s *Session) []renderer.DrawItem
}

// ToolPreview returns the active tool's transient preview draw items for the UI head, which
// assembles its own frame draw list (the headless RenderFrame path uses the same source).
func (s *Session) ToolPreview() []renderer.DrawItem { return s.toolPreviewItems() }

// toolPreviewItems returns the active tool's transient preview: the translucent solid
// result preview when the tool can draft its feature (DraftPreviewable), otherwise the
// legacy wireframe (Previewable). Nil when no tool is active or it offers no preview.
func (s *Session) toolPreviewItems() []renderer.DrawItem {
	if s.tool == nil {
		return nil
	}
	if dp, ok := s.tool.tool.(DraftPreviewable); ok {
		if items := s.featurePreviewItems(dp); items != nil {
			return items
		}
	}
	if p, ok := s.tool.tool.(Previewable); ok {
		return p.Preview(s)
	}
	return nil
}

// AddOverlay adds a persistent client-graphics primitive (overlay line/points drawn
// over the model). ClearOverlays removes them; Overlays returns a snapshot.
func (s *Session) AddOverlay(item renderer.DrawItem) { s.overlays = append(s.overlays, item) }
func (s *Session) ClearOverlays()                    { s.overlays = nil }
func (s *Session) Overlays() []renderer.DrawItem {
	out := make([]renderer.DrawItem, len(s.overlays))
	copy(out, s.overlays)
	return out
}

// Graphics returns the add-in client/interaction graphics store (M05-F05) for the ACTIVE
// document — the seam the router mutates from clientGraphics.* / interactionGraphics.* calls.
// Persistent graphics are document-owned (the API contract), so each document gets its own
// store and overlays from one document never bleed into another's viewport; a scratch store
// backs the no-active-document case. The store is created lazily and shares the host body
// resolver used for surface overlays.
func (s *Session) Graphics() *clientgraphics.Store {
	d := s.ActiveDocument()
	if d == nil {
		return s.graphics
	}
	st, ok := s.graphicsByDoc[d.ID()]
	if !ok {
		st = clientgraphics.NewStore()
		st.SetBodyResolver(s.resolveOverlayMesh)
		s.graphicsByDoc[d.ID()] = st
	}
	return st
}

// GraphicsLabels returns the world-anchored text labels of the live client graphics, for
// the UI head to draw via its projected-ImGui label path (text is not draw-list geometry).
func (s *Session) GraphicsLabels() []clientgraphics.Label {
	_, labels, _ := s.Graphics().Build(s.camera)
	return labels
}

// RenderFrame builds the frame draw list — the active part's bodies, the persistent
// overlays, the active tool's transient preview, and the add-in client/interaction
// graphics — and submits it to the backend.
func (s *Session) RenderFrame(backend renderer.Backend) {
	list := renderer.BuildDrawListStyled(s.VisibleBodies(), s.camera, ops.DefaultQuality(), s.SurfaceLookup(), s.visualStyle)
	list.Items = append(list.Items, s.overlays...)
	list.Items = append(list.Items, s.toolPreviewItems()...)
	graphics, _, _ := s.Graphics().Build(s.camera)
	list.Items = append(list.Items, graphics...)
	backend.Render(list)
}

// sceneBodies returns the bodies to draw — the active part's, if any.
func (s *Session) sceneBodies() []*topo.Body { return s.VisibleBodies() }

// VisibleBodies returns the bodies the viewport renders and the picker hit-tests for the active
// document: a part's own bodies (minus those hidden by browser visibility state), or — for an
// assembly — its placed components transformed into assembly space (#769). An assembly's
// per-occurrence visibility is suppression, already applied when collecting placed bodies, so the
// part hidden-body filter does not apply to it.
func (s *Session) VisibleBodies() []*topo.Body {
	if asm, err := activeAssembly(s); err == nil {
		return s.worldAssemblyBodies(asm)
	}
	part, err := activePart(s)
	if err != nil {
		return nil
	}
	var out []*topo.Body
	for _, body := range part.SurfaceBodies().All() {
		if s.BodyVisible(body) {
			out = append(out, body)
		}
	}
	return out
}

// BodyVisible reports whether body is included in the active scene body list.
func (s *Session) BodyVisible(body *topo.Body) bool {
	if body == nil {
		return false
	}
	return !s.activeHiddenBodyKeys()[string(body.ReferenceKey())]
}

// SetBodyVisible updates the browser-driven display state for one body.
func (s *Session) SetBodyVisible(body *topo.Body, visible bool) {
	if body == nil {
		return
	}
	keys := s.activeHiddenBodyKeys()
	key := string(body.ReferenceKey())
	if visible {
		delete(keys, key)
		return
	}
	keys[key] = true
}

// activeHiddenBodyKeys returns the hidden-body set for the active document, creating it on first
// use; with no active document it returns the scratch set. Body visibility is per-document view
// state (#1105): a single session-wide map leaked one document's hidden bodies into another's
// viewport and never released them on close — the same class of bug as the client-graphics store.
func (s *Session) activeHiddenBodyKeys() map[string]bool {
	d := s.ActiveDocument()
	if d == nil {
		return s.hiddenBodyKeys
	}
	keys, ok := s.hiddenBodyKeysByDoc[d.ID()]
	if !ok {
		keys = map[string]bool{}
		s.hiddenBodyKeysByDoc[d.ID()] = keys
	}
	return keys
}

// ToggleBodyVisibility flips the browser-driven display state for one body.
func (s *Session) ToggleBodyVisibility(body *topo.Body) { s.SetBodyVisible(body, !s.BodyVisible(body)) }
