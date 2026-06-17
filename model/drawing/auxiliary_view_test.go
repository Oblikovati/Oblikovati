// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/subd"
)

// TestAuxiliaryFoldZeroEqualsTopView checks an auxiliary folded off a FRONT view about a
// horizontal line (0°) projects the same direction as the TOP base view — so the two produce
// the same visible/hidden edge split (the fold generalises the orthographic projections).
func TestAuxiliaryFoldZeroEqualsTopView(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1}); err != nil {
		t.Fatalf("AddBase FRONT: %v", err)
	}
	top, err := views.AddBase(BaseViewSpec{Name: "TOP", Orientation: types.BaseViewTop, Scale: 1})
	if err != nil {
		t.Fatalf("AddBase TOP: %v", err)
	}
	aux, err := views.AddAuxiliary(AuxiliaryViewSpec{Name: "AUX", ParentView: "FRONT", FoldAngleRad: 0, CenterX: 250, CenterY: 100})
	if err != nil {
		t.Fatalf("AddAuxiliary: %v", err)
	}
	if aux.Type() != types.DrawingViewAuxiliary || aux.IsProjected() {
		t.Errorf("aux view type = %v (projected=%v), want auxiliary/not-projected", aux.Type(), aux.IsProjected())
	}
	av, ah := aux.VisibleHidden()
	tv, th := top.VisibleHidden()
	if av != tv || ah != th {
		t.Errorf("aux(front,0°) = %d/%d, want it to match TOP %d/%d", av, ah, tv, th)
	}
}

// TestAuxiliaryFoldNinetyEqualsRightView checks folding about a vertical line (90°) matches the
// RIGHT view's projection direction.
func TestAuxiliaryFoldNinetyEqualsRightView(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1}); err != nil {
		t.Fatalf("AddBase FRONT: %v", err)
	}
	right, err := views.AddBase(BaseViewSpec{Name: "RIGHT", Orientation: types.BaseViewRight, Scale: 1})
	if err != nil {
		t.Fatalf("AddBase RIGHT: %v", err)
	}
	aux, err := views.AddAuxiliary(AuxiliaryViewSpec{ParentView: "FRONT", FoldAngleRad: deg90, CenterX: 250, CenterY: 250})
	if err != nil {
		t.Fatalf("AddAuxiliary: %v", err)
	}
	av, ah := aux.VisibleHidden()
	rv, rh := right.VisibleHidden()
	if av != rv || ah != rh {
		t.Errorf("aux(front,90°) = %d/%d, want it to match RIGHT %d/%d", av, ah, rv, rh)
	}
}

const deg90 = 1.5707963267948966 // π/2

// TestAuxiliaryRejectsNonBaseParent checks an auxiliary can only fold off a base view.
func TestAuxiliaryRejectsNonBaseParent(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	if _, err := views.AddProjected(ProjectedViewSpec{Name: "RIGHT", BaseView: "FRONT", Direction: types.ProjectRight}); err != nil {
		t.Fatalf("AddProjected: %v", err)
	}
	if _, err := views.AddAuxiliary(AuxiliaryViewSpec{ParentView: "RIGHT", FoldAngleRad: 0}); err == nil {
		t.Error("auxiliary off a projected view = ok, want error (parent must be a base view)")
	}
	if _, err := views.AddAuxiliary(AuxiliaryViewSpec{ParentView: "NOPE", FoldAngleRad: 0}); err == nil {
		t.Error("auxiliary off a missing parent = ok, want error")
	}
}

// TestAuxiliaryCascadesOnParentDelete checks deleting the base view removes the auxiliary that
// folds off it.
func TestAuxiliaryCascadesOnParentDelete(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	if _, err := views.AddAuxiliary(AuxiliaryViewSpec{Name: "AUX", ParentView: "FRONT", FoldAngleRad: 0}); err != nil {
		t.Fatalf("AddAuxiliary: %v", err)
	}
	if err := views.Remove("FRONT"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if views.Count() != 0 {
		t.Errorf("views after deleting the base = %d, want 0 (auxiliary cascades)", views.Count())
	}
}

// TestAuxiliaryRecipeRoundTrip checks an auxiliary view's type, parent and fold angle survive
// persistence (its curves re-project on open).
func TestAuxiliaryRecipeRoundTrip(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddBase(BaseViewSpec{Name: "FRONT", Orientation: types.BaseViewFront, Scale: 1}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}
	if _, err := views.AddAuxiliary(AuxiliaryViewSpec{Name: "AUX", ParentView: "FRONT", FoldAngleRad: deg90, CenterX: 200, CenterY: 150}); err != nil {
		t.Fatalf("AddAuxiliary: %v", err)
	}
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
	v, ok := restored.Sheets().Active().Views().ByName("AUX")
	if !ok {
		t.Fatal("restored drawing has no AUX view")
	}
	if v.Type() != types.DrawingViewAuxiliary || v.BaseViewName() != "FRONT" {
		t.Errorf("restored AUX = type %v parent %q, want auxiliary off FRONT", v.Type(), v.BaseViewName())
	}
	if v.CurveCount() == 0 {
		t.Error("restored AUX re-projected no curves")
	}
}
