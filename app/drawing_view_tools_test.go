// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/drawing"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// drawingWithModelSession returns a session with a boxed part "box.opd" and an active drawing
// that references it — the fixture for the drawing-view tools.
func drawingWithModelSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("commands: %v", err)
	}
	part, err := compdef.AddPart(s.Workspace(), "box.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	def := part.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(-2, -2))
	c1 := sk.Points().Add(math.P2(2, -2))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(-2, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()

	if _, err := s.NewDrawing(); err != nil {
		t.Fatalf("NewDrawing: %v", err)
	}
	c, err := ActiveDrawing(s)
	if err != nil {
		t.Fatalf("ActiveDrawing: %v", err)
	}
	c.SetModelReference("box.opd")
	return s
}

func TestBaseViewToolPlacesViaCursor(t *testing.T) {
	s := drawingWithModelSession(t)
	tool := NewBaseViewTool()
	tool.Start(s)
	// Choose iso, then place at the cursor sheet position.
	tool.Params().Choices[0].Set(6) // index 6 = Isometric
	if got := tool.PreviewCurves(s); len(got) == 0 {
		t.Fatal("BaseViewTool produced no preview curves")
	}
	tool.SetPlacement(150, 100)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	c, _ := ActiveDrawing(s)
	views := c.Sheets().Active().Views()
	if views.Count() != 1 {
		t.Fatalf("view count = %d, want 1", views.Count())
	}
	v := views.Item(0)
	if v.Orientation() != types.BaseViewIso || v.CurveCount() == 0 {
		t.Errorf("placed view = %v with %d curves, want iso with curves", v.Orientation(), v.CurveCount())
	}
	x, y := v.CenterMM()
	if x != 150 || y != 100 {
		t.Errorf("placed at (%g,%g), want the cursor (150,100)", x, y)
	}
}

func TestAuxiliaryViewToolFoldsOffBase(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	if _, err := c.Sheets().Active().Views().AddBase(drawing.BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	tool := NewAuxiliaryViewTool()
	tool.Start(s)
	if !tool.CanCommit() {
		t.Fatal("auxiliary tool cannot commit with a base view present")
	}
	tool.Params().Floats[0].Set(30) // fold 30°
	if got := tool.PreviewCurves(s); len(got) == 0 {
		t.Fatal("auxiliary tool produced no preview curves")
	}
	tool.SetPlacement(260, 200)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	views := c.Sheets().Active().Views()
	v, ok := views.ByName("VIEW:2")
	if !ok {
		t.Fatalf("auxiliary view not added (have %d views)", views.Count())
	}
	if v.Type() != types.DrawingViewAuxiliary || v.BaseViewName() != "FRONT" {
		t.Errorf("placed view = type %v parent %q, want auxiliary off FRONT", v.Type(), v.BaseViewName())
	}
	if x, y := v.CenterMM(); x != 260 || y != 200 {
		t.Errorf("placed at (%g,%g), want the cursor (260,200)", x, y)
	}
}

func TestPickSelectEditDeleteDrawingView(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(drawing.BaseViewSpec{Orientation: types.BaseViewIso, Scale: 1, CenterX: 150, CenterY: 100}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}

	// Pick at the view centre returns its handle.
	h, ok := s.PickDrawingViewAt(150, 100)
	if !ok {
		t.Fatal("PickDrawingViewAt(150,100) found no view")
	}
	s.SelectDrawingViewHandle(h)
	if _, ok := s.Selection().First().(DrawingViewHandle); !ok {
		t.Error("selecting a drawing view did not put its handle in the selection")
	}

	// Edit opens an edit tool bound to the view.
	s.BeginEditDrawingView(h)
	if s.ActiveTool() == nil {
		t.Error("BeginEditDrawingView did not start an edit tool")
	}
	s.CancelTool()

	// Delete removes it.
	if err := s.DeleteDrawingView(h); err != nil {
		t.Fatalf("DeleteDrawingView: %v", err)
	}
	if views.Count() != 0 {
		t.Errorf("view count after delete = %d, want 0", views.Count())
	}
}

func TestProjectedViewToolAndEditCommit(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(drawing.BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}

	// Projected view tool: previews and places off the base.
	pt := NewProjectedViewTool()
	pt.Start(s)
	if !pt.CanCommit() || len(pt.PreviewCurves(s)) == 0 {
		t.Fatal("ProjectedViewTool should be committable with a base view and produce a preview")
	}
	pt.SetPlacement(250, 100)
	if err := pt.Commit(s); err != nil {
		t.Fatalf("projected Commit: %v", err)
	}
	if views.Count() != 2 {
		t.Fatalf("view count = %d, want 2 (base + projected)", views.Count())
	}

	// Edit the base view's scale via the edit tool's Commit.
	front, _ := views.ByName("FRONT")
	edit := newDrawingViewEditTool(views, front)
	for _, f := range edit.Params().Floats {
		if f.Label == "Scale" {
			f.Set(2)
		}
	}
	if err := edit.Commit(s); err != nil {
		t.Fatalf("edit Commit: %v", err)
	}
	if v, _ := views.ByName("FRONT"); v.Scale() != 2 {
		t.Errorf("edited scale = %g, want 2", v.Scale())
	}
}

func TestDrawingViewBrowserNodesAndMenu(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	if _, err := c.Sheets().Active().Views().AddBase(drawing.BaseViewSpec{Orientation: types.BaseViewFront, Scale: 1}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	root := BuildBrowser(s)
	// root → Sheet:1 → the view node.
	if len(root.Children) == 0 || len(root.Children[0].Children) == 0 {
		t.Fatalf("browser tree = %+v, want a sheet with a view node", root)
	}
	viewNode := root.Children[0].Children[0]
	if viewNode.Kind != "drawingView" {
		t.Fatalf("view node kind = %q, want drawingView", viewNode.Kind)
	}
	menu := BrowserMenu(viewNode)
	if len(menu) != 2 || menu[0].Label != "Edit" || menu[1].Label != "Delete" {
		t.Errorf("view menu = %+v, want [Edit Delete]", menu)
	}
}
