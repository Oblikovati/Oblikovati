// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/compdef"
)

// drawingSession returns a session with a part "widget.opd" carrying iProperties and an
// active drawing created through Session.NewDrawing (so its resolver is wired).
func drawingSession(t *testing.T) *Session {
	t.Helper()
	s := NewSession()
	part, err := compdef.AddPart(s.Workspace(), "widget.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	props := part.Content().(*compdef.PartComponentDefinition).Properties()
	props.Set(attr.DesignTracking).Put("Part Number", attr.StringValue("PN-7"))
	if _, err := s.NewDrawing(); err != nil {
		t.Fatalf("NewDrawing: %v", err)
	}
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	return s
}

func TestNewDrawingActivatesDrawingWithDefaultSheet(t *testing.T) {
	s := drawingSession(t)
	c, err := ActiveDrawing(s)
	if err != nil {
		t.Fatalf("ActiveDrawing: %v", err)
	}
	if c.Sheets().Count() != 1 || c.Sheets().Active().Size() != types.SheetSizeA3 {
		t.Errorf("new drawing = %d sheets / %v, want 1 / A3", c.Sheets().Count(), c.Sheets().Active().Size())
	}
	if !hasActiveDrawing(s) {
		t.Error("hasActiveDrawing should be true with a drawing active")
	}
}

func TestDrawingRibbonTabAndPanels(t *testing.T) {
	tab, ok := BuildRibbon(drawingSession(t)).Tab("Drawing")
	if !ok {
		t.Fatal("an active drawing should show the Drawing ribbon tab")
	}
	for _, panel := range []string{"Sheets", "Setup"} {
		if _, ok := tab.Panel(panel); !ok {
			t.Errorf("Drawing tab has no %s panel", panel)
		}
	}
}

func TestDrawingTabAbsentForPart(t *testing.T) {
	s := NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	if _, ok := BuildRibbon(s).Tab("Drawing"); ok {
		t.Error("a part ribbon should not show the Drawing tab")
	}
}

func TestAddSheetToolAddsConfiguredSheet(t *testing.T) {
	s := drawingSession(t)
	tool := NewAddSheetTool()
	tool.Start(s)
	// Choose A4 portrait via the exposed params, as the dialog would.
	p := tool.Params()
	p.Choices[0].Set(sheetSizeIndexOf(types.SheetSizeA4))
	p.Choices[1].Set(int(types.SheetPortrait))
	if !tool.CanCommit() {
		t.Fatal("AddSheetTool should be committable")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	c, _ := ActiveDrawing(s)
	if c.Sheets().Count() != 2 {
		t.Fatalf("sheet count = %d, want 2", c.Sheets().Count())
	}
	added := c.Sheets().Active()
	if added.Size() != types.SheetSizeA4 || added.WidthMM() != 210 {
		t.Errorf("added sheet = %v %g wide, want A4 210", added.Size(), added.WidthMM())
	}
}

// TestModelReferenceToolDrivesTitleBlock is the PBI-137 acceptance through the UI tool:
// referencing the part fills the title block from its iProperties.
func TestModelReferenceToolDrivesTitleBlock(t *testing.T) {
	s := drawingSession(t)
	tool := NewModelReferenceTool()
	tool.Start(s)
	if !tool.CanCommit() {
		t.Fatal("ModelReferenceTool should be committable with an open model")
	}
	// The only referenceable model is widget.opd.
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	c, _ := ActiveDrawing(s)
	if c.ModelReference() != "widget.opd" {
		t.Fatalf("model reference = %q, want widget.opd", c.ModelReference())
	}
	tb := c.Sheets().Active().TitleBlock()
	if v, _ := tb.FieldValue("Part Number"); v != "PN-7" {
		t.Errorf("title block Part Number = %q, want PN-7", v)
	}
}

func TestDraftingStandardToolSwitchesStandard(t *testing.T) {
	s := drawingSession(t)
	c, _ := ActiveDrawing(s)
	if c.Styles().ActiveStandard() != types.DraftingISO {
		t.Fatalf("default standard = %v, want ISO", c.Styles().ActiveStandard())
	}
	tool := NewDraftingStandardTool()
	if tool.Name() == "" || !tool.CanCommit() {
		t.Fatal("DraftingStandardTool should be named and committable")
	}
	tool.Start(s)
	tool.Pick(s, nil) // no-op for a dialog-only tool
	tool.Params().Choices[0].Set(draftingStandardIndexOf(types.DraftingANSI))
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if c.Styles().ActiveStandard() != types.DraftingANSI {
		t.Errorf("after tool = %v, want ANSI", c.Styles().ActiveStandard())
	}
	if c.Styles().ActiveStyle().DimensionStyle().Unit() != types.DimensionInch {
		t.Error("ANSI dimension style should report inches")
	}
	NewDraftingStandardTool().Cancel(s) // abandoning leaves the standard unchanged
	if c.Styles().ActiveStandard() != types.DraftingANSI {
		t.Error("Cancel must not change the active standard")
	}
}

func TestDeleteSheetKeepsAtLeastOne(t *testing.T) {
	s := drawingSession(t)
	if canDeleteSheet(s) {
		t.Error("a single-sheet drawing should not allow delete")
	}
	if err := NewAddSheetTool().Commit(s); err != nil {
		t.Fatalf("add sheet: %v", err)
	}
	if !canDeleteSheet(s) {
		t.Fatal("a two-sheet drawing should allow delete")
	}
	if err := deleteActiveSheet(s); err != nil {
		t.Fatalf("deleteActiveSheet: %v", err)
	}
	c, _ := ActiveDrawing(s)
	if c.Sheets().Count() != 1 {
		t.Errorf("sheet count after delete = %d, want 1", c.Sheets().Count())
	}
}
