//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// boxHostSession builds a part with one extruded box and a production-like picker (bodies AND work
// planes), so the Create 2D Sketch hover path resolves real geometry with the origin planes hidden
// behind the solid — the setting where the face-host bug showed up.
func boxHostSession(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("register: %v", err)
	}
	pd, err := compdef.AddPart(s.Workspace(), "facehost.opd", true)
	if err != nil {
		t.Fatalf("add part: %v", err)
	}
	_ = s.Workspace().SetActiveDocument(pd)
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(6, 0))
	c2 := sk.Points().Add(math.P2(6, 6))
	c3 := sk.Points().Add(math.P2(0, 6))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 6 })
	def.Recompute()
	s.SetPicker(app.NewRayPicker(s.Camera(), func() []*topo.Body { return def.SurfaceBodies().All() }).
		WithPlanes(func() []*feature.WorkPlane { return s.PickableWorkPlanes() }))
	return s
}

// TestInWindowCreateSketchHoversFaceHighlight is the live confirmation for the reported bug: with
// Create 2D Sketch active and the cursor over a solid's planar face, the face must highlight (the
// tool now accepts faces as sketch hosts, not just work planes). It drives real DrawChrome frames
// with the cursor parked over the box in the live window and saves a PNG to read back. Skips
// cleanly without a display/Vulkan.
func TestInWindowCreateSketchHoversFaceHighlight(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := boxHostSession(t)
	frameCameraOn(s, s.VisibleBodies()[0].RangeBox())
	s.StartTool(app.NewCreateSketchTool())
	if s.ActiveTool() == nil {
		t.Fatal("Create 2D Sketch tool did not start")
	}

	cx, cy := float32(480), float32(400) // fullscreen viewport ⇒ over the box's front-right face

	// Settle the layout + camera and establish hover over the face (an ImGui item is not reported
	// hovered on its first appearance), then draw several frames so the hover-highlight overlay
	// (toolHoverHighlight → drawSelectable on the FaceHandle) renders into the captured frame.
	for range 10 {
		native.InjectMousePos(cx, cy)
		viewportFrame(win, s)
	}

	// Live regression for the reported bug: with the tool active, the cursor over the box must
	// resolve to the FACE through the production picker — not the XY origin plane behind the solid.
	ox, oy := native.ItemRectMin()
	sel, ok := s.PickAt(float64(cx-ox), float64(cy-oy), s.Selection().Filter())
	if _, isFace := sel.(app.FaceHandle); !ok || !isFace {
		t.Fatalf("Create 2D Sketch hover over a face resolved to %T (ok=%v); want a FaceHandle", sel, ok)
	}

	if err := win.SaveWindowPNG(filepath.Join(outDir(), "create-sketch-face-hover.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}

// TestInWindowCreateSketchRevealsOriginPlane is the live confirmation for #1752: on a brand-new part
// (no body), starting Create 2D Sketch reveals the default-hidden origin frame, so the XY plane both
// draws in the viewport AND resolves under the cursor through the production picker — clicking it opens
// the sketch. Before the fix the viewport was empty and the only way into the first sketch was the
// browser Origin folder. Saves a PNG for eyeball confirmation. Skips cleanly without display/Vulkan.
func TestInWindowCreateSketchRevealsOriginPlane(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession() // part; camera looks down −Z at the origin; origin planes default hidden
	s.SetPicker(app.NewRayPicker(s.Camera(), func() []*topo.Body { return nil }).
		WithPlanes(func() []*feature.WorkPlane { return s.PickableWorkPlanes() }))
	s.StartTool(app.NewCreateSketchTool())

	cx, cy := float32(inWinW/2), float32(inWinH/2)
	for range 10 { // settle layout + hover so the reveal overlay renders into the frame
		native.InjectMousePos(cx, cy)
		viewportFrame(win, s)
	}

	ox, oy := native.ItemRectMin()
	sel, ok := s.PickAt(float64(cx-ox), float64(cy-oy), s.Selection().Filter())
	if _, isPlane := sel.(app.WorkPlaneHandle); !ok || !isPlane {
		t.Fatalf("Create Sketch over the revealed origin frame resolved to %T (ok=%v); want a WorkPlaneHandle", sel, ok)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "create-sketch-origin-reveal.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}

	native.InjectMouseButton(native.MouseLeft, true) // click the revealed plane → the sketch opens on it
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, false)
	viewportFrame(win, s)
	if !s.InSketch() {
		t.Error("clicking the revealed origin plane should enter the sketch")
	}
}

// TestInWindowToolFaceHighlightGeneralizes proves the unified face highlight is not bespoke to
// Create Sketch: a DIFFERENT face-picking tool (Shell, which declares AcceptedKinds = {Face}) lights
// the hovered face through the same engine path, with no Shell-specific head code (ADR-0041).
func TestInWindowToolFaceHighlightGeneralizes(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := boxHostSession(t)
	frameCameraOn(s, s.VisibleBodies()[0].RangeBox())
	s.StartTool(app.NewShellTool())

	cx, cy := float32(480), float32(400)
	for range 10 {
		native.InjectMousePos(cx, cy)
		viewportFrame(win, s)
	}
	ox, oy := native.ItemRectMin()
	sel, ok := s.PickAt(float64(cx-ox), float64(cy-oy), s.Selection().Filter())
	if _, isFace := sel.(app.FaceHandle); !ok || !isFace {
		t.Fatalf("Shell hover over a face resolved to %T (ok=%v); want a FaceHandle", sel, ok)
	}

	// Click the face so the tool holds a pick, then render more frames — exercising the SELECTED
	// highlight path (toolSelectedHighlight → ToolPicks → the per-kind renderer) too.
	native.InjectMouseButton(native.MouseLeft, true)
	viewportFrame(win, s)
	native.InjectMouseButton(native.MouseLeft, false)
	for range 3 {
		native.InjectMousePos(cx, cy)
		viewportFrame(win, s)
	}
	if picks := s.ToolPicks(); len(picks) == 0 {
		t.Error("clicking the face did not register a Shell pick for the selected highlight")
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "shell-face-hover.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
