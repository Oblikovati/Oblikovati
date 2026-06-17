// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/attr"
)

// fakeModelProperties is a named fake (CLAUDE.md: no inline stubs) standing in for a
// referenced model's iProperties. Mutating it between reads exercises the "title block
// updates with iProperties" acceptance.
type fakeModelProperties struct {
	values map[string]string // "Set:Prop" → value
}

func newFakeModelProperties() *fakeModelProperties {
	return &fakeModelProperties{values: map[string]string{}}
}

func (f *fakeModelProperties) set(setName, prop, value string) { f.values[setName+":"+prop] = value }

func (f *fakeModelProperties) Property(setName, prop string) (string, bool) {
	v, ok := f.values[setName+":"+prop]
	return v, ok
}

func TestNewContentHasDefaultSheet(t *testing.T) {
	c := NewContent()
	if got := c.Sheets().Count(); got != 1 {
		t.Fatalf("new drawing sheet count = %d, want 1", got)
	}
	sh := c.Sheets().Active()
	if sh == nil {
		t.Fatal("new drawing has no active sheet")
	}
	if sh.Size() != types.SheetSizeA3 || sh.Orientation() != types.SheetLandscape {
		t.Errorf("default sheet = %v/%v, want A3/landscape", sh.Size(), sh.Orientation())
	}
	if sh.WidthMM() != 420 || sh.HeightMM() != 297 {
		t.Errorf("A3 landscape = %g×%g mm, want 420×297", sh.WidthMM(), sh.HeightMM())
	}
	if sh.Border() == nil || sh.TitleBlock() == nil {
		t.Error("default sheet should have a border and a title block")
	}
}

func TestAddSheetAutoNamesAndActivates(t *testing.T) {
	c := NewContent()
	sh, err := c.Sheets().Add(SheetSpec{Size: types.SheetSizeA4, Orientation: types.SheetPortrait})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if sh.Name() != "Sheet:2" {
		t.Errorf("auto name = %q, want Sheet:2", sh.Name())
	}
	if c.Sheets().Active() != sh {
		t.Error("added sheet should become active")
	}
	if sh.WidthMM() != 210 || sh.HeightMM() != 297 {
		t.Errorf("A4 portrait = %g×%g, want 210×297", sh.WidthMM(), sh.HeightMM())
	}
}

func TestAddCustomSheetRequiresPositiveDims(t *testing.T) {
	c := NewContent()
	if _, err := c.Sheets().Add(SheetSpec{Size: types.SheetSizeCustom}); err == nil {
		t.Error("custom sheet with zero dimensions should error")
	}
	sh, err := c.Sheets().Add(SheetSpec{Size: types.SheetSizeCustom, WidthMM: 300, HeightMM: 200})
	if err != nil {
		t.Fatalf("custom Add: %v", err)
	}
	if sh.WidthMM() != 300 || sh.HeightMM() != 200 {
		t.Errorf("custom = %g×%g, want 300×200", sh.WidthMM(), sh.HeightMM())
	}
}

// TestTitleBlockResolvesAndUpdatesWithModel is the PBI-137 acceptance: a title block's
// fields resolve from the referenced model's iProperties, and tracking a later edit.
func TestTitleBlockResolvesAndUpdatesWithModel(t *testing.T) {
	c := NewContent()
	props := newFakeModelProperties()
	props.set(attr.DesignTracking, "Part Number", "PN-1001")
	c.SetModelProperties(props)
	c.SetModelReference("widget.opd")

	tb := c.Sheets().Active().titleBlock
	if v, _ := tb.FieldValue("Part Number"); v != "PN-1001" {
		t.Fatalf("Part Number = %q, want PN-1001", v)
	}
	// Edit the model's iProperty: the title block must reflect it on the next read.
	props.set(attr.DesignTracking, "Part Number", "PN-2002")
	if v, _ := tb.FieldValue("Part Number"); v != "PN-2002" {
		t.Errorf("after edit Part Number = %q, want PN-2002", v)
	}
}

func TestTitleBlockBlankWithoutModelReference(t *testing.T) {
	c := NewContent()
	props := newFakeModelProperties()
	props.set(attr.SummaryInformation, "Title", "Bracket")
	c.SetModelProperties(props)
	// No SetModelReference: property fields resolve to "".
	if v, _ := c.Sheets().Active().titleBlock.FieldValue("Title"); v != "" {
		t.Errorf("Title without model reference = %q, want empty", v)
	}
}

func TestRemoveKeepsAtLeastOneSheet(t *testing.T) {
	c := NewContent()
	if err := c.Sheets().Remove("Sheet:1"); err == nil {
		t.Error("removing the last sheet should error")
	}
	if _, err := c.Sheets().Add(SheetSpec{Size: types.SheetSizeA4}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := c.Sheets().Remove("Sheet:1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if c.Sheets().Count() != 1 || c.Sheets().Active() == nil {
		t.Error("after remove, one sheet should remain active")
	}
}

func TestRecipeRoundTrip(t *testing.T) {
	c := NewContent()
	c.SetModelReference("widget.opd")
	if _, err := c.Sheets().Add(SheetSpec{Name: "Detail", Size: types.SheetSizeCustom, WidthMM: 500, HeightMM: 350, Orientation: types.SheetLandscape}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := c.Sheets().SetActive("Sheet:1"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}

	blob, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	restored := NewContent()
	props := newFakeModelProperties()
	props.set(attr.SummaryInformation, "Title", "Widget")
	restored.SetModelProperties(props)
	if err := restored.ApplyRecipe(blob); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}

	if restored.ModelReference() != "widget.opd" {
		t.Errorf("restored model ref = %q, want widget.opd", restored.ModelReference())
	}
	if restored.Sheets().Count() != 2 {
		t.Fatalf("restored sheet count = %d, want 2", restored.Sheets().Count())
	}
	if restored.Sheets().Active().Name() != "Sheet:1" {
		t.Errorf("restored active = %q, want Sheet:1", restored.Sheets().Active().Name())
	}
	detail, ok := restored.Sheets().ByName("Detail")
	if !ok || detail.WidthMM() != 500 || detail.HeightMM() != 350 {
		t.Errorf("restored custom sheet = %+v, want 500×350", detail)
	}
	// Title block re-resolves against the (re-injected) model after restore.
	if v, _ := detail.titleBlock.FieldValue("Title"); v != "Widget" {
		t.Errorf("restored title resolves to %q, want Widget", v)
	}
}
