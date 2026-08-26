//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
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

// topCapFace returns the box's +Z planar cap face.
func topCapFace(t *testing.T, body *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range body.Faces() {
		if pl, ok := f.Geometry().(geom.Plane); ok && float64(pl.Normal().Z) > 0.9 {
			return f
		}
	}
	t.Fatal("no +Z cap face on the box")
	return nil
}

// TestSketchProjectPerPickAndRenders is the end-to-end regression for the reported Project Geometry
// gaps while editing a sketch: the tool picks model references (so the head draws the model-face
// hover highlight in the sketch overlay path, #2158), a single pick PROJECTS the face's perimeter
// with no dialog/OK and the tool stays armed, and the sketch-edit frame — now carrying the projected
// curves and the model-reference highlight — renders without crashing. It picks the face directly to
// stay independent of viewport pixel layout.
func TestSketchProjectPerPickAndRenders(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := boxHostSession(t) // box [0,6]³ on XY
	body := s.VisibleBodies()[0]
	top := topCapFace(t, body)
	frameCameraOn(s, body.RangeBox())
	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	if !s.InSketch() {
		t.Fatal("expected to be editing the sketch")
	}
	tool := app.NewProjectGeometryTool()
	s.StartTool(tool)

	// The head draws the model-reference hover highlight in-sketch only for a tool that picks model
	// references — the wiring that made a projectable face visible again (#2158).
	if !s.ToolPicksModelReferences() {
		t.Fatal("Project Geometry must pick model references while in-sketch")
	}

	// A single pick projects the face's four boundary edges immediately — no OK — and the tool stays
	// armed for the next pick.
	sk := s.ActiveSketch()
	before := projectedCurveCount(sk)
	tool.Pick(s, app.FaceHandle{Face: top, Body: body})
	if got := projectedCurveCount(sk) - before; got != 4 {
		t.Fatalf("picking the face projected %d curves, want 4 (its edges) — no OK needed", got)
	}
	if s.ActiveTool() == nil {
		t.Error("Project Geometry deactivated after one pick; it must stay armed")
	}

	// The sketch-edit frame (projected-curve overlay + model-reference highlight wiring) renders.
	for range 4 {
		native.InjectMousePos(float32(inWinW/2), float32(inWinH/2))
		viewportFrame(win, s)
	}
}
