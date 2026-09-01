// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// #2039: the Format panel is registered on the 3D Sketch tab from the same list as the 2D one,
// but nothing behind it worked in 3D — no creation mode was ever applied and the three lists
// resolved only the planar sketch, so all six controls answered ok and changed nothing.

// commitLine3D draws one 3D line through the tool + commit seam, which is where the armed
// creation modes are applied.
func commitLine3D(t *testing.T, s *Session, a, b math.Point3) *sketch.Line3D {
	t.Helper()
	tool := NewLine3DTool()
	s.StartTool(tool)
	tool.AddPoint(a)
	tool.AddPoint(b)
	if err := s.OK(); err != nil {
		t.Fatalf("commit 3D line: %v", err)
	}
	sk := s.ActiveSketch3D()
	ents := sk.Entities()
	l, ok := ents[len(ents)-1].(*sketch.Line3D)
	if !ok {
		t.Fatalf("last 3D entity is %T, want *sketch.Line3D", ents[len(ents)-1])
	}
	return l
}

func TestSketch3DConstructionModeMarksNewGeometry(t *testing.T) {
	t.Parallel()
	s, _ := sketch3DSession(t)
	if n := s.ToggleConstruction(); n != 0 {
		t.Fatalf("with nothing selected ToggleConstruction converted %d entities, want 0 (it should arm)", n)
	}
	l := commitLine3D(t, s, math.P3(0, 0, 0), math.P3(10, 0, 0))
	if !l.IsConstruction() {
		t.Error("a 3D line drawn with construction mode armed is not construction geometry")
	}
}

func TestSketch3DConstructionModeDisarmedLeavesGeometryNormal(t *testing.T) {
	t.Parallel()
	s, _ := sketch3DSession(t)
	l := commitLine3D(t, s, math.P3(0, 0, 0), math.P3(10, 0, 0))
	if l.IsConstruction() {
		t.Error("a 3D line drawn with no mode armed should be normal geometry")
	}
}

// The convert branch already worked in 3D — the ray picker puts a 3D entity in the selection as
// a SketchEntityHandle — so this pins it against a regression while the arm branch is added.
func TestSketch3DConstructionConvertsSelection(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(10, 0, 0))
	s.Selection().Add(SketchEntityHandle{Entity: l})
	if n := s.ToggleConstruction(); n != 1 {
		t.Fatalf("ToggleConstruction converted %d entities, want 1", n)
	}
	if !l.IsConstruction() {
		t.Error("the selected 3D line was not converted to construction")
	}
	if s.ConstructionMode() {
		t.Error("converting a selection must not also arm the creation mode")
	}
}

func TestSketch3DDrivenDimensionModeMarksNewDimensions(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(10, 0, 0))
	if n := s.ToggleDrivenDimension(); n != 0 {
		t.Fatalf("with nothing selected ToggleDrivenDimension changed %d dimensions, want 0", n)
	}
	if err := s.Execute("Sketch3D.Dimension"); err != nil {
		t.Fatalf("Sketch3D.Dimension: %v", err)
	}
	s.feedPick(SketchEntityHandle{Entity: l}) // one line is ready → auto-commits through OK
	if s.ActiveTool() != nil {
		t.Fatal("the dimension tool should have committed on the line pick")
	}
	dims := sk.DimensionConstraints3D().All()
	if len(dims) != 1 {
		t.Fatalf("3D sketch has %d dimensions, want 1", len(dims))
	}
	if !dims[0].Driven() {
		t.Error("a 3D dimension created with driven mode armed is not driven")
	}
}

func TestSketch3DFormatListsEditTheSelection(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(10, 0, 0))
	s.Selection().Add(SketchEntityHandle{Entity: l})

	if n := s.SetSelectionLineType(string(types.SketchLineDashed)); n != 1 {
		t.Fatalf("SetSelectionLineType changed %d entities, want 1", n)
	}
	if n := s.SetSelectionLineWeight(0.5); n != 1 {
		t.Fatalf("SetSelectionLineWeight changed %d entities, want 1", n)
	}
	if n := s.SetSelectionColor(types.NewColor(200, 30, 30)); n != 1 {
		t.Fatalf("SetSelectionColor changed %d entities, want 1", n)
	}
	f, ok := sk.EntityFormat(l.EntityID())
	if !ok {
		t.Fatal("the 3D line carries no format overrides after all three lists set one")
	}
	if f.LineType != string(types.SketchLineDashed) || f.LineWeight != 0.5 || !f.Color.IsOverride() {
		t.Errorf("3D entity format = %+v, want dashed / 0.5 / an override colour", f)
	}
	// The three lists compose rather than overwrite, so the panel reads back what it set.
	if got := s.SelectionFormat(); got != f {
		t.Errorf("SelectionFormat = %+v, want %+v", got, f)
	}
}

// The style resolver is what the 3D overlay draws through: a construction curve dashes, an
// override colour and weight reach the screen, and Show Format suppresses the overrides.
func TestSketch3DEntityStyleResolvesOverrides(t *testing.T) {
	t.Parallel()
	_, sk := sketch3DSession(t)
	l := sk.AddLine3D(math.P3(0, 0, 0), math.P3(10, 0, 0))

	if p := SketchEntityStyle(sk, l, false).Pattern; p != nil {
		t.Errorf("a plain 3D line should draw solid, got pattern %v", p)
	}
	l.SetConstruction(true)
	if p := SketchEntityStyle(sk, l, false).Pattern; len(p) == 0 {
		t.Error("a construction 3D line should draw dashed")
	}

	sk.SetEntityFormat(l.EntityID(), sketch.EntityFormat{
		LineType: string(types.SketchLineDashed), Color: types.NewColor(200, 30, 30), LineWeight: 0.5,
	})
	styled := SketchEntityStyle(sk, l, false)
	if !styled.Color.IsOverride() || styled.LineWeight != 0.5 {
		t.Errorf("3D style = %+v, want the per-entity colour and weight", styled)
	}
	suppressed := SketchEntityStyle(sk, l, true)
	if suppressed.Color.IsOverride() || suppressed.LineWeight != 0 {
		t.Errorf("Show Format should suppress the overrides, got %+v", suppressed)
	}
}
