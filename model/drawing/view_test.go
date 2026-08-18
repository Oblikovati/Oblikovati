// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
)

// fakeBodyResolver is a named fake (CLAUDE.md: no inline stubs) returning one body for any
// reference — the referenced model in drawing-view tests.
type fakeBodyResolver struct{ body *topo.Body }

func (f fakeBodyResolver) Body(string) (*topo.Body, bool) { return f.body, f.body != nil }

func drawingWithBox(t *testing.T) *Content {
	t.Helper()
	c := NewContent()
	// Distinct dimensions (2×3×4) so the isometric projection is in general position — no two
	// corners project coincident, keeping the visible/hidden counts platform-stable (a perfect
	// cube under iso is FP-degenerate).
	c.SetBodyResolver(fakeBodyResolver{body: subd.ToBody(subd.Box(2, 3, 4), "box")})
	c.SetModelReference("box.opd")
	return c
}

// frontBase adds a standard FRONT base view named "FRONT" — the parent the derived-view tests
// build on.
func frontBase(t *testing.T, views *DrawingViews) {
	t.Helper()
	if _, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100}); err != nil {
		t.Fatalf("AddBase FRONT: %v", err)
	}
}

// reopen marshals a drawing and restores it into a fresh box-backed content, re-projecting its
// views — the persistence round-trip the view tests share.
func reopen(t *testing.T, c *Content) *Content {
	t.Helper()
	data, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	restored := NewContent()
	restored.SetBodyResolver(fakeBodyResolver{body: subd.ToBody(subd.Box(2, 3, 4), "box")})
	if err := restored.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	restored.RecomputeViews()
	return restored
}

// TestAddBaseViewIsoProjectsCube checks a base iso view of a cube produces the textbook 9
// visible / 3 hidden edges, placed on the sheet and keyed for associativity.
func TestAddBaseViewIsoProjectsCube(t *testing.T) {
	c := drawingWithBox(t)
	v, err := c.Sheets().Active().Views().AddBase(BaseViewSpec{Orientation: types.BaseViewIso, Scale: 1, CenterX: 150, CenterY: 100})
	if err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	visible, hidden := v.VisibleHidden()
	if visible != 9 || hidden != 3 {
		t.Fatalf("iso cube view = %d visible / %d hidden, want 9/3", visible, hidden)
	}
	for _, curve := range v.Curves() {
		if len(curve.EdgeKey()) == 0 {
			t.Error("drawing curve has no source edge key (not associative)")
		}
	}
}

// TestHiddenLineRemovedDropsHiddenCurves the hidden-line-removed style keeps the same visible edges
// as hidden-line but produces ZERO hidden curves (#1985): the same iso cube that shows 9/3 hidden-
// line shows 9/0.
func TestHiddenLineRemovedDropsHiddenCurves(t *testing.T) {
	c := drawingWithBox(t)
	v, err := c.Sheets().Active().Views().AddBase(BaseViewSpec{
		Orientation: types.BaseViewIso, Scale: 1, CenterX: 150, CenterY: 100, Style: types.HiddenLineRemovedViewStyle,
	})
	if err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	if visible, hidden := v.VisibleHidden(); visible != 9 || hidden != 0 {
		t.Errorf("hidden-line-removed iso cube = %d visible / %d hidden, want 9/0", visible, hidden)
	}
}

// TestFromBaseResolvesToParentStyle a FromBase view renders with its base view's style, and falls
// back to the hidden-line default when the base is missing (#1985).
func TestFromBaseResolvesToParentStyle(t *testing.T) {
	v := &DrawingView{style: types.FromBaseViewStyle, baseView: "FRONT"}
	v.resolveEffectiveStyle(func(name string) (types.DrawingViewStyle, bool) {
		if name == "FRONT" {
			return types.HiddenLineRemovedViewStyle, true
		}
		return 0, false
	})
	if v.EffectiveStyle() != types.HiddenLineRemovedViewStyle {
		t.Errorf("FromBase effective style = %v, want the base's hiddenLineRemoved", v.EffectiveStyle())
	}
	orphan := &DrawingView{style: types.FromBaseViewStyle, baseView: "GONE"}
	orphan.resolveEffectiveStyle(func(string) (types.DrawingViewStyle, bool) { return 0, false })
	if orphan.EffectiveStyle() != types.HiddenLineViewStyle {
		t.Errorf("orphan FromBase effective style = %v, want the hidden-line default", orphan.EffectiveStyle())
	}
}

// TestViewLabelDefaultAndOverrides the default caption carries the view name and a scale note; the
// show flags drop each part; a free-text override replaces it; hiding the label empties it (#1983).
func TestViewLabelDefaultAndOverrides(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	v, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 0.5, CenterX: 100, CenterY: 100})
	if err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	if got := v.Label(); !strings.Contains(got, "FRONT") || !strings.Contains(got, "1:2") {
		t.Errorf("default label = %q, want the name FRONT and scale 1:2", got)
	}
	no := false
	if err := views.SetLabel("FRONT", ViewLabelStyle{ShowScale: &no}); err != nil {
		t.Fatalf("SetLabel: %v", err)
	}
	if got := v.Label(); strings.Contains(got, "1:2") || !strings.Contains(got, "FRONT") {
		t.Errorf("scale-hidden label = %q, want the name without the scale note", got)
	}
	if err := views.SetLabel("FRONT", ViewLabelStyle{Text: strPtr("DETAIL A")}); err != nil {
		t.Fatalf("SetLabel override: %v", err)
	}
	if got := v.Label(); got != "DETAIL A" {
		t.Errorf("override label = %q, want DETAIL A", got)
	}
	if err := views.SetLabel("FRONT", ViewLabelStyle{ShowLabel: &no}); err != nil {
		t.Fatalf("SetLabel hide: %v", err)
	}
	if got := v.Label(); got != "" {
		t.Errorf("hidden label = %q, want empty", got)
	}
}

// TestViewLabelRoundTrip checks a view's caption override and hidden-scale flag survive save +
// reopen (#1983) — otherwise a customised label reverts to the default on open.
func TestViewLabelRoundTrip(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 0.5, CenterX: 100, CenterY: 100}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	no := false
	if err := views.SetLabel("FRONT", ViewLabelStyle{Text: strPtr("DETAIL A"), ShowScale: &no}); err != nil {
		t.Fatalf("SetLabel: %v", err)
	}
	v, ok := reopen(t, c).Sheets().Active().Views().ByName("FRONT")
	if !ok {
		t.Fatal("reopened drawing lost view FRONT")
	}
	if got := v.Label(); got != "DETAIL A" {
		t.Errorf("restored label = %q, want the DETAIL A override", got)
	}
}

func strPtr(s string) *string { return &s }

func TestAddBaseViewRequiresModel(t *testing.T) {
	c := NewContent() // no body resolver / reference
	if _, err := c.Sheets().Active().Views().AddBase(BaseViewSpec{Orientation: types.BaseViewFront}); err == nil {
		t.Error("AddBase without a referenced model should error")
	}
}

func TestAddProjectedViewFromBase(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	base, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1})
	if err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	proj, err := views.AddProjected(ProjectedViewSpec{Name: "RIGHT", BaseView: "FRONT", Direction: types.ProjectRight, CenterX: 200})
	if err != nil {
		t.Fatalf("AddProjected: %v", err)
	}
	if !proj.IsProjected() || proj.BaseViewName() != "FRONT" {
		t.Errorf("projected view = %+v, want projected off FRONT", proj)
	}
	if proj.CurveCount() == 0 {
		t.Error("projected view produced no curves")
	}
	// Removing the base view also removes the view projected from it.
	if err := views.Remove("FRONT"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if views.Count() != 0 {
		t.Errorf("after removing base, %d views remain, want 0 (projected cascades)", views.Count())
	}
	_ = base
}

func TestPreviewAndEditViews(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()

	// Preview (origin-centred, not added) for a base view.
	if prev, ok := views.PreviewBase(types.BaseViewFront, types.HiddenLineViewStyle, 1); !ok || len(prev) == 0 {
		t.Fatalf("PreviewBase = (%d curves, %v), want curves", len(prev), ok)
	}
	base, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1, CenterX: 100, CenterY: 100})
	if err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	if prev, ok := views.PreviewProjected("FRONT", types.ProjectRight); !ok || len(prev) == 0 {
		t.Fatalf("PreviewProjected = (%d curves, %v), want curves", len(prev), ok)
	}

	// EditBase changes orientation + scale and re-projects in place.
	if err := views.EditBase("FRONT", types.BaseViewTop, types.WireframeViewStyle, 0.5, 120, 80); err != nil {
		t.Fatalf("EditBase: %v", err)
	}
	if base.Orientation() != types.BaseViewTop || base.Scale() != 0.5 {
		t.Errorf("edited base = %v/%g, want top/0.5", base.Orientation(), base.Scale())
	}

	// EditProjected on a projected view changes its direction.
	proj, err := views.AddProjected(ProjectedViewSpec{Name: "SIDE", BaseView: "FRONT", Direction: types.ProjectRight})
	if err != nil {
		t.Fatalf("AddProjected: %v", err)
	}
	if err := views.EditProjected("SIDE", types.ProjectUp, 200, 150); err != nil {
		t.Fatalf("EditProjected: %v", err)
	}
	if proj.Direction() != types.ProjectUp {
		t.Errorf("edited projected direction = %v, want up", proj.Direction())
	}
	// Editing a missing view errors.
	if err := views.EditBase("ghost", types.BaseViewFront, types.HiddenLineViewStyle, 1, 0, 0); err == nil {
		t.Error("EditBase on a missing view should error")
	}
}

func TestViewsSurviveRecipeRoundTrip(t *testing.T) {
	c := drawingWithBox(t)
	if _, err := c.Sheets().Active().Views().AddBase(BaseViewSpec{Name: "TOP", Orientation: types.BaseViewTop, Scale: 0.5, CenterX: 120, CenterY: 90}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	blob, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	restored := NewContent()
	restored.SetBodyResolver(fakeBodyResolver{body: subd.ToBody(subd.Box(2, 2, 2), "box")})
	if err := restored.ApplyRecipe(blob); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	v, ok := restored.Sheets().Active().Views().ByName("TOP")
	if !ok || v.Orientation() != types.BaseViewTop || v.Scale() != 0.5 {
		t.Fatalf("restored view = %+v, want TOP scale 0.5", v)
	}
	// Curves are re-projected, not stored: empty until recompute, then populated.
	if v.CurveCount() != 0 {
		t.Errorf("restored view should have no curves before recompute, got %d", v.CurveCount())
	}
	restored.RecomputeViews()
	if v.CurveCount() == 0 {
		t.Error("RecomputeViews should re-project the restored view's curves")
	}
}
