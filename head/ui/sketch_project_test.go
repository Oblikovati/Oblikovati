//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// projectedCurveCount counts the sketch's projected reference curves.
func projectedCurveCount(sk *sketch.Sketch) int {
	n := 0
	for _, e := range sk.Entities() {
		if _, ok := e.(*sketch.ProjectedCurve); ok {
			n++
		}
	}
	return n
}

// TestSketchProjectHighlightsAndProjectsModelFace is the end-to-end regression for the reported
// Project Geometry gaps while editing a sketch: (1) the model face under the cursor must HIGHLIGHT
// (the sketch overlay path never drew the model-reference hover highlight, so a projectable face
// looked dead, #2158), and (2) a single click must PROJECT it — no dialog, no OK — with the tool
// staying armed. A straight-down camera puts the box's top face under the centre pixel.
func TestSketchProjectHighlightsAndProjectsModelFace(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := boxHostSession(t) // box [0,6]³ on XY
	cam := scene.NewCamera(inWinW, inWinH)
	cam.Eye, cam.Target, cam.Up = math.P3(3, 3, 30), math.P3(3, 3, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
	s.SetPicker(app.NewRayPicker(cam, func() []*topo.Body { return s.VisibleBodies() }))
	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if !s.InSketch() {
		t.Fatal("expected to be editing the sketch")
	}
	s.StartTool(app.NewProjectGeometryTool())
	sk := s.ActiveSketch()

	cx, cy := float32(inWinW/2), float32(inWinH/2)
	for i := 0; i < 10; i++ {
		native.InjectMousePos(cx, cy)
		viewportFrame(win, s)
	}
	// (1) The model face under the cursor resolves through the tool's filter — so it highlights and
	// is projectable while editing the sketch (the sketch overlay path now draws the model-reference
	// hover highlight for a model-reference tool, #2158).
	ox, oy := native.ItemRectMin()
	lx, ly := float64(cx-ox), float64(cy-oy)
	if !s.ToolPicksModelReferences() {
		t.Fatal("Project Geometry must pick model references while in-sketch")
	}
	if sel, ok := s.PickAt(lx, ly, s.Selection().Filter()); !ok {
		t.Fatal("no model geometry under the cursor through the tool filter")
	} else if _, isFace := sel.(app.FaceHandle); !isFace {
		t.Fatalf("hover resolved to %T, want a FaceHandle (face highlight/projection)", sel)
	}

	// (2) A single click projects the face's perimeter — no OK — and the tool stays armed.
	before := projectedCurveCount(sk)
	s.Click(lx, ly)
	if got := projectedCurveCount(sk) - before; got != 4 {
		t.Fatalf("a click projected %d curves, want 4 (the top face's edges) — no OK needed", got)
	}
	if s.ActiveTool() == nil {
		t.Error("Project Geometry deactivated after one click; it must stay armed for the next pick")
	}
}
