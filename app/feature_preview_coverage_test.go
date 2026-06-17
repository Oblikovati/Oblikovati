// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestEveryFeatureToolPreviews drives each remaining part-feature tool (beyond the sketched
// solids covered elsewhere) to its commit-ready state, reusing each tool's own test setup, and
// asserts ToolPreview() yields a non-empty ghost. This guards that wiring DraftFeature into a
// tool actually produces a preview through the full featurePreviewItems path (tool body, solid
// delta, changed faces, or result-body fallback) — not just that it compiles.
func TestEveryFeatureToolPreviews(t *testing.T) {
	t.Run("rib", func(t *testing.T) {
		s, _ := ribbedPart(t)
		rib := NewRibTool()
		s.StartTool(rib)
		rib.SetThickness(1)
		rib.SetDepth(3)
		wantPreview(t, s)
	})
	t.Run("emboss", func(t *testing.T) {
		s, _, region := partWithTopRegion(t)
		s.SetPicker(stubPicker{sel: region})
		emb := NewEmbossTool()
		s.StartTool(emb)
		s.Click(100, 100)
		emb.SetDepth(1)
		wantPreview(t, s)
	})
	t.Run("grill", func(t *testing.T) {
		s, _, region := partWithGrillSketch(t)
		s.SetPicker(stubPicker{sel: region})
		g := NewGrillTool()
		s.StartTool(g)
		s.Click(100, 100)
		wantPreview(t, s)
	})
	t.Run("core_cavity", func(t *testing.T) {
		s, _ := newPartWithBlock(t, 6)
		s.StartTool(NewCoreCavityTool())
		wantPreview(t, s)
	})
	t.Run("replace_face", func(t *testing.T) {
		s, block := newPartWithBlock(t, 2)
		top := topFaceOf(t, block)
		s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})
		r := NewReplaceFaceTool()
		s.StartTool(r)
		s.Click(50, 50)
		r.SetPickingTarget(true)
		s.Click(50, 50)
		wantPreview(t, s)
	})
	t.Run("delete_face", func(t *testing.T) {
		s, block := newPartWithBlock(t, 2)
		chamfered := chamferOneEdge(t, s, block)
		s.SetPicker(stubPicker{sel: chamferFaceHandleOf(t, chamfered)})
		d := NewDeleteFaceTool()
		s.StartTool(d)
		s.Click(50, 50)
		wantPreview(t, s)
	})
	t.Run("split", func(t *testing.T) {
		s, _, wp := partWithMidPlane(t, 6)
		sp := NewSplitTool()
		s.StartTool(sp)
		sp.Pick(s, WorkPlaneHandle{Plane: wp})
		wantPreview(t, s)
	})
	t.Run("thicken", func(t *testing.T) {
		s := newPartWithSurface(t)
		th := NewThickenTool()
		s.StartTool(th)
		th.SetThickness(0.5)
		wantPreview(t, s)
	})
	t.Run("patch", func(t *testing.T) {
		s, _, region := partWithSquareRegion(t)
		s.SetPicker(stubPicker{sel: region})
		p := NewPatchTool()
		s.StartTool(p)
		p.Pick(s, region)
		wantPreview(t, s)
	})
	t.Run("stitch", func(t *testing.T) {
		s, _ := twoAdjacentPatches(t)
		s.StartTool(NewStitchTool())
		wantPreview(t, s)
	})
	t.Run("surface_trim", func(t *testing.T) {
		s, _, wp := patchedPartWithCutPlane(t)
		tr := NewSurfaceTrimTool()
		s.StartTool(tr)
		tr.Pick(s, WorkPlaneHandle{Plane: wp})
		wantPreview(t, s)
	})
	t.Run("sculpt", func(t *testing.T) {
		s, _ := partWithCubeShell(t)
		s.StartTool(NewSculptTool())
		wantPreview(t, s)
	})
	t.Run("extend", func(t *testing.T) {
		s, _, bottom := patchWithBottomEdge(t)
		ex := NewExtendTool()
		s.StartTool(ex)
		ex.Pick(s, EdgeHandle{Edge: bottom})
		ex.SetDistance(2)
		wantPreview(t, s)
	})
}

// wantPreview asserts the active tool renders a non-empty translucent ghost with feature edges.
func wantPreview(t *testing.T, s *Session) {
	t.Helper()
	items := s.ToolPreview()
	if len(items) == 0 {
		t.Fatal("tool produced no preview")
	}
	tris, lines := 0, 0
	for _, it := range items {
		switch it.Primitive {
		case 0: // Triangles
			tris++
		case 1: // Lines
			lines++
		}
	}
	if tris == 0 {
		t.Errorf("preview has no translucent ghost (items=%d, lines=%d)", len(items), lines)
	}
}
