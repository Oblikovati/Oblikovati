// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

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
	t.Run("face_offset", func(t *testing.T) {
		s, block := newPartWithBlock(t, 4)
		s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
		off := NewFaceOffsetTool()
		s.StartTool(off)
		s.Click(100, 100)
		off.SetDistance(1)
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

// TestSweptAndDressUpToolsPreview drives the sketched-solid (revolve/sweep/loft/coil/hole) and
// dress-up (fillet/chamfer/shell/draft/thread) tools to commit-ready state, reusing each tool's
// own end-to-end setup, and asserts ToolPreview() yields a ghost. This pins the DraftFeature
// path of every tool that does NOT go through the changed-face/result-body fallbacks.
func TestSweptAndDressUpToolsPreview(t *testing.T) {
	t.Run("revolve", func(t *testing.T) {
		s, profile := newPartWithOffsetSquare(t, 2, 2)
		s.SetPicker(stubPicker{sel: profile})
		rv := NewRevolveTool()
		s.StartTool(rv)
		s.Click(120, 90)
		wantPreview(t, s)
	})
	t.Run("coil", func(t *testing.T) {
		s, profile := newPartWithOffsetSquare(t, 4, 1)
		s.SetPicker(stubPicker{sel: profile})
		c := NewCoilTool()
		s.StartTool(c)
		s.Click(120, 90)
		c.SetPitch(2)
		c.SetRevolutions(3)
		wantPreview(t, s)
	})
	t.Run("sweep", func(t *testing.T) {
		s, profile, path := newPartWithProfileAndPath(t)
		s.SetPicker(&seqPicker{sels: []Selectable{profile, path}})
		sw := NewSweepTool()
		s.StartTool(sw)
		s.Click(10, 10)
		s.Click(10, 200)
		wantPreview(t, s)
	})
	t.Run("loft", func(t *testing.T) {
		s, bottom, top := newPartWithStackedSquares(t)
		s.SetPicker(&seqPicker{sels: []Selectable{bottom, top}})
		l := NewLoftTool()
		s.StartTool(l)
		s.Click(10, 10)
		s.Click(10, 200)
		wantPreview(t, s)
	})
	t.Run("hole", func(t *testing.T) {
		s, block := newPartWithBlock(t, 4)
		s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
		hole := NewHoleTool()
		s.StartTool(hole)
		s.Click(100, 100)
		hole.SetDiameter(2)
		hole.SetDepth(3)
		wantPreview(t, s)
	})
	t.Run("fillet", func(t *testing.T) {
		s, block := newPartWithBlock(t, 2)
		s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
		f := NewFilletTool()
		s.StartTool(f)
		s.Click(50, 50)
		f.SetRadius(0.5)
		wantPreview(t, s)
	})
	t.Run("chamfer", func(t *testing.T) {
		s, block := newPartWithBlock(t, 2)
		s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})
		ch := NewChamferTool()
		s.StartTool(ch)
		s.Click(50, 50)
		ch.SetDistance(0.5)
		wantPreview(t, s)
	})
	t.Run("shell", func(t *testing.T) {
		s, block := newPartWithBlock(t, 4)
		s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
		sh := NewShellTool()
		s.StartTool(sh)
		s.Click(100, 100)
		sh.SetThickness(0.5)
		wantPreview(t, s)
	})
	t.Run("draft", func(t *testing.T) {
		s, block := newPartWithBlock(t, 2)
		s.SetPicker(stubPicker{sel: FaceHandle{Face: plusXFaceOf(t, block), Body: block}})
		d := NewDraftTool()
		s.StartTool(d)
		s.Click(50, 50)
		d.SetAngleDegrees(-10)
		wantPreview(t, s)
	})
	t.Run("thread", func(t *testing.T) {
		s, cyl := newPartWithCylinder(t)
		s.SetPicker(stubPicker{sel: FaceHandle{Face: cylinderFaceOf(t, cyl), Body: cyl}})
		tool := NewThreadTool()
		s.StartTool(tool)
		s.Click(100, 100)
		tool.SetStandardIndex(0)
		tool.SetSizeIndex(6)
		tool.SetPitchIndex(0)
		tool.SetCut(true)
		wantPreview(t, s)
	})
}

// TestSheetMetalToolsPreview drives the sheet-metal tools to commit-ready state and asserts a
// preview ghost, pinning each sheet-metal DraftFeature wiring.
func TestSheetMetalToolsPreview(t *testing.T) {
	t.Run("face", func(t *testing.T) {
		s, part := sheetMetalSession(t)
		face := NewSheetMetalFaceTool()
		s.StartTool(face)
		face.Pick(s, squareProfile(part, 4))
		wantPreview(t, s)
	})
	t.Run("flange", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		flange := NewSheetMetalFlangeTool()
		s.StartTool(flange)
		flange.Pick(s, EdgeHandle{Edge: topXEdge(t, part.Features().Result()[0])})
		flange.SetHeight(1)
		wantPreview(t, s)
	})
	t.Run("lip", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		lip := NewSheetMetalLipTool()
		s.StartTool(lip)
		lip.Pick(s, EdgeHandle{Edge: topXEdge(t, part.Features().Result()[0])})
		lip.SetHeight(1.0)
		lip.SetReturnLength(0.4)
		lip.SetAngle(halfPiAngle)
		wantPreview(t, s)
	})
	t.Run("rip", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		rip := NewSheetMetalRipTool()
		s.StartTool(rip)
		rip.Pick(s, lineSketch(part, gmath.P2(1, 1.5), gmath.P2(3, 1.5)))
		rip.SetGap(0.05)
		wantPreview(t, s)
	})
	t.Run("punch", func(t *testing.T) {
		s, part := faceSheet(t, 4)
		holes := part.Sketches().Add(sketch.XYPlane())
		q := []gmath.Point2{gmath.P2(0.7, 0.7), gmath.P2(1.3, 0.7), gmath.P2(1.3, 1.3), gmath.P2(0.7, 1.3)}
		var pts []*sketch.Point
		for _, p := range q {
			pts = append(pts, holes.Points().Add(p))
		}
		for i := range pts {
			holes.Lines().Add(pts[i], pts[(i+1)%len(pts)])
		}
		punch := NewSheetMetalPunchTool()
		s.StartTool(punch)
		punch.Pick(s, ProfileHandle{Sketch: holes, ProfileIndex: 0})
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
