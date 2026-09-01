// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/renderer"
)

func TestRenderFrameDrawsTheModeledSolid(t *testing.T) {
	t.Parallel()
	// Build a box through the UI, then render — the null backend records the frame.
	s := extrudedBox(t, 2, 4)
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	if null.FrameCount() != 1 {
		t.Fatalf("frames = %d, want 1", null.FrameCount())
	}
	if tris := null.LastFrame().Triangles(); tris != 12 {
		t.Errorf("rendered triangles = %d, want 12 (the box)", tris)
	}
	if null.LastFrame().Lines() == 0 {
		t.Error("rendered frame has no wireframe lines")
	}
}

func TestPersistentOverlayInFrame(t *testing.T) {
	t.Parallel()
	s := extrudedBox(t, 2, 4)
	before := frameItems(s)
	s.AddOverlay(renderer.DrawItem{Primitive: renderer.Lines, Positions: nil, Indices: []int{0, 1}})
	if len(s.Overlays()) != 1 {
		t.Fatal("overlay not stored")
	}
	if frameItems(s) != before+1 {
		t.Error("overlay not included in the frame")
	}
	s.ClearOverlays()
	if frameItems(s) != before {
		t.Error("ClearOverlays did not remove the overlay")
	}
}

func TestExtrudeToolLivePreview(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithSquare(t, 2)
	s.SetPicker(stubPicker{sel: profile})
	ext := NewExtrudeTool()
	s.StartTool(ext)

	// No preview before a profile + distance are gathered.
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	preCount := len(null.LastFrame().Items)

	s.Click(0, 0)
	ext.SetDistance(5)
	// Now the tool contributes a translucent solid result preview to the frame.
	null.Reset()
	s.RenderFrame(null)
	if len(null.LastFrame().Items) <= preCount {
		t.Fatal("no preview item added once the extrude is ready")
	}
	prev := previewItems(null.LastFrame())
	if len(prev) == 0 {
		t.Fatal("expected translucent triangle preview items once the extrude is ready")
	}
	// Building a new body on an empty part adds volume → the preview is GREEN.
	if !sameHue(prev[0].Color, previewAddColor) {
		t.Errorf("additive extrude preview color = %v, want green %v", prev[0].Color, previewAddColor)
	}
	if previewTriangles(prev) == 0 {
		t.Error("preview has no triangles")
	}

	// A Cut of the same region previews RED (it would remove material from the new body).
	// (Here there is no base material yet, so this only exercises the color path on commit
	// order; the cut/remove color is covered directly in TestFeaturePreviewCutIsRed.)

	// After OK the preview is gone and the real solid is rendered.
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	null.Reset()
	s.RenderFrame(null)
	if null.LastFrame().Triangles() != 12 {
		t.Errorf("post-commit frame triangles = %d, want 12", null.LastFrame().Triangles())
	}
	if len(previewItems(null.LastFrame())) != 0 {
		t.Error("preview items should be gone after commit")
	}
}

// previewItems returns the translucent (Opacity<1) triangle items in a frame — the live
// feature preview, distinct from the opaque modeled solid.
func previewItems(frame renderer.DrawList) []renderer.DrawItem {
	var out []renderer.DrawItem
	for _, it := range frame.Items {
		if it.Primitive == renderer.Triangles && it.Opacity > 0 && it.Opacity < 1 {
			out = append(out, it)
		}
	}
	return out
}

func previewTriangles(items []renderer.DrawItem) int {
	n := 0
	for _, it := range items {
		n += len(it.Indices) / 3
	}
	return n
}

func TestRenderFrameNoActivePart(t *testing.T) {
	t.Parallel()
	s := NewSession() // no document
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	if null.FrameCount() != 1 || len(null.LastFrame().Items) != 0 {
		t.Error("empty session should render an empty frame")
	}
}

func TestRenderFrameSkipsHiddenBody(t *testing.T) {
	t.Parallel()
	s := extrudedBox(t, 2, 4)
	body := s.VisibleBodies()[0]
	s.SetBodyVisible(body, false)
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	if tris := null.LastFrame().Triangles(); tris != 0 {
		t.Fatalf("hidden body rendered %d triangles, want 0", tris)
	}
}

func frameItems(s *Session) int {
	null := &renderer.NullBackend{}
	s.RenderFrame(null)
	return len(null.LastFrame().Items)
}
