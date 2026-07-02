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

// drawingWithFrontBase returns a session whose active drawing already has a FRONT base view —
// the parent the derived-view tools (projected/auxiliary/section) build on.
func drawingWithFrontBase(t *testing.T) *Session {
	t.Helper()
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	if _, err := c.Sheets().Active().Views().AddBase(drawing.BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	return s
}

func TestAuxiliaryViewToolFoldsOffBase(t *testing.T) {
	s := drawingWithFrontBase(t)
	c, _ := ActiveDrawing(s)
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

func TestAuxiliaryViewToolWithoutBaseView(t *testing.T) {
	s := drawingWithModelSession(t)
	tool := NewAuxiliaryViewTool()
	if tool.Name() != "Auxiliary View" {
		t.Errorf("Name() = %q, want Auxiliary View", tool.Name())
	}
	tool.Start(s) // no base view present
	if tool.CanCommit() {
		t.Error("auxiliary tool can commit with no base view, want it disabled")
	}
	if got := tool.PreviewCurves(s); got != nil {
		t.Errorf("preview with no base view = %d curves, want none", len(got))
	}
	tool.Pick(s, nil)
	tool.Cancel(s)
	if err := tool.Commit(s); err == nil {
		t.Error("Commit with no base view = ok, want error")
	}
}

func TestSectionViewToolCutsThroughBase(t *testing.T) {
	s := drawingWithFrontBase(t)
	c, _ := ActiveDrawing(s)
	tool := NewSectionViewTool()
	tool.Start(s)
	if !tool.CanCommit() {
		t.Fatal("section tool cannot commit with a base view present")
	}
	tool.Params().Choices[1].Set(1) // vertical cut
	if got := tool.PreviewCurves(s); len(got) == 0 {
		t.Fatal("section tool produced no preview curves")
	}
	tool.SetPlacement(100, 250)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	v, ok := c.Sheets().Active().Views().ByName("VIEW:2")
	if !ok || v.Type() != types.DrawingViewSection {
		t.Fatalf("section view not added as a section (have %d views)", c.Sheets().Active().Views().Count())
	}
}

func TestSectionViewToolWithoutBaseView(t *testing.T) {
	s := drawingWithModelSession(t)
	tool := NewSectionViewTool()
	if tool.Name() != "Section View" {
		t.Errorf("Name() = %q, want Section View", tool.Name())
	}
	tool.Start(s) // no base view
	if tool.CanCommit() {
		t.Error("section tool can commit with no base view, want it disabled")
	}
	if got := tool.PreviewCurves(s); got != nil {
		t.Errorf("preview with no base view = %d curves, want none", len(got))
	}
	tool.Pick(s, nil)
	tool.Cancel(s)
	if err := tool.Commit(s); err == nil {
		t.Error("Commit with no base view = ok, want error")
	}
}

func TestDetailViewToolMagnifiesBase(t *testing.T) {
	s := drawingWithFrontBase(t)
	c, _ := ActiveDrawing(s)
	tool := NewDetailViewTool()
	tool.Start(s)
	if tool.Name() != "Detail View" || !tool.CanCommit() {
		t.Fatalf("detail tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	tool.Params().Floats[0].Set(3) // 3× magnification
	tool.PreviewCurves(s)          // exercise the preview path (may be empty for a hollow box centre)
	tool.SetPlacement(260, 220)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	v, ok := c.Sheets().Active().Views().ByName("VIEW:2")
	if !ok {
		t.Fatal("detail view not added")
	}
	if v.Type() != types.DrawingViewDetail || v.Scale() != 3 {
		t.Fatalf("added view = type %v scale %g, want detail at 3×", v.Type(), v.Scale())
	}
}

func TestDetailViewToolWithoutBaseView(t *testing.T) {
	s := drawingWithModelSession(t)
	tool := NewDetailViewTool()
	tool.Start(s)
	if tool.CanCommit() {
		t.Error("detail tool can commit with no base view, want it disabled")
	}
	if got := tool.PreviewCurves(s); got != nil {
		t.Errorf("preview with no base view = %d curves, want none", len(got))
	}
	if err := tool.Commit(s); err == nil {
		t.Error("Commit with no base view = ok, want error")
	}
}

func TestBreakViewToolCompressesBase(t *testing.T) {
	s := drawingWithFrontBase(t)
	c, _ := ActiveDrawing(s)
	tool := NewBreakViewTool()
	tool.Start(s)
	if tool.Name() != "Break View" || !tool.CanCommit() {
		t.Fatalf("break tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	tool.Params().Choices[1].Set(1) // vertical break
	tool.PreviewCurves(s)
	tool.SetPlacement(260, 220)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	v, ok := c.Sheets().Active().Views().ByName("VIEW:2")
	if !ok || v.Type() != types.DrawingViewBreak {
		t.Fatalf("break view not added correctly (ok=%v)", ok)
	}
}

func TestBreakViewToolWithoutBaseView(t *testing.T) {
	s := drawingWithModelSession(t)
	tool := NewBreakViewTool()
	tool.Start(s)
	if tool.CanCommit() {
		t.Error("break tool can commit with no base view, want it disabled")
	}
	if got := tool.PreviewCurves(s); got != nil {
		t.Errorf("preview with no base view = %d curves, want none", len(got))
	}
	if err := tool.Commit(s); err == nil {
		t.Error("Commit with no base view = ok, want error")
	}
}

func TestSliceAndBreakoutViewTools(t *testing.T) {
	s := drawingWithFrontBase(t)
	c, _ := ActiveDrawing(s)
	slice := NewSliceViewTool()
	slice.Start(s)
	if slice.Name() != "Slice View" || !slice.CanCommit() {
		t.Fatalf("slice tool name/commit wrong: %q / %v", slice.Name(), slice.CanCommit())
	}
	slice.PreviewCurves(s)
	slice.SetPlacement(260, 220)
	if err := slice.Commit(s); err != nil {
		t.Fatalf("slice Commit: %v", err)
	}
	bo := NewBreakoutViewTool()
	bo.Start(s)
	if bo.Name() != "Breakout View" || !bo.CanCommit() {
		t.Fatalf("breakout tool name/commit wrong: %q / %v", bo.Name(), bo.CanCommit())
	}
	bo.PreviewCurves(s)
	bo.SetPlacement(260, 320)
	if err := bo.Commit(s); err != nil {
		t.Fatalf("breakout Commit: %v", err)
	}
	views := c.Sheets().Active().Views()
	if v, ok := views.ByName("VIEW:2"); !ok || v.Type() != types.DrawingViewSlice {
		t.Errorf("VIEW:2 not a slice view (ok=%v)", ok)
	}
	if v, ok := views.ByName("VIEW:3"); !ok || v.Type() != types.DrawingViewBreakout {
		t.Errorf("VIEW:3 not a breakout view (ok=%v)", ok)
	}
}

func TestDraftViewToolNeedsNoModel(t *testing.T) {
	s := drawingWithModelSession(t) // a drawing; draft ignores the model
	c, _ := ActiveDrawing(s)
	tool := NewDraftViewTool()
	tool.Start(s)
	if tool.Name() != "Draft View" || !tool.CanCommit() {
		t.Fatalf("draft tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	tool.Params().Floats[0].Set(120)
	if got := tool.PreviewCurves(s); len(got) != 4 {
		t.Errorf("draft preview = %d curves, want a 4-edge frame", len(got))
	}
	tool.Pick(s, nil)
	tool.Cancel(s)
	tool.SetPlacement(200, 200)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if v, ok := c.Sheets().Active().Views().ByName("VIEW:1"); !ok || v.Type() != types.DrawingViewDraft {
		t.Errorf("draft view not added (ok=%v)", ok)
	}
}

func TestSliceBreakoutToolsWithoutBase(t *testing.T) {
	s := drawingWithModelSession(t)
	for _, tool := range []interface {
		Start(*Session)
		CanCommit() bool
		Commit(*Session) error
		PreviewCurves(*Session) []drawing.DrawingCurve
	}{NewSliceViewTool(), NewBreakoutViewTool()} {
		tool.Start(s)
		if tool.CanCommit() {
			t.Error("derived view tool can commit with no base view")
		}
		if got := tool.PreviewCurves(s); got != nil {
			t.Errorf("preview with no base = %d curves, want none", len(got))
		}
		if err := tool.Commit(s); err == nil {
			t.Error("Commit with no base = ok, want error")
		}
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
	menu := BrowserMenu(s, viewNode)
	if len(menu) != 2 || menu[0].Label != "Edit" || menu[1].Label != "Delete" {
		t.Errorf("view menu = %+v, want [Edit Delete]", menu)
	}
}
