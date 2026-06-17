// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/compdef"
)

// drawingSession seeds a part "widget.opd" with iProperties and an active drawing
// "sheet.odd" that references it — the fixture for the title-block resolution test.
func drawingSession(t *testing.T) (*Router, *app.Session) {
	t.Helper()
	r := New(opregistry.Default())
	s := app.NewSession()

	part, err := compdef.AddPart(s.Workspace(), "widget.opd", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	props := part.Content().(*compdef.PartComponentDefinition).Properties()
	props.Set(attr.DesignTracking).Put("Part Number", attr.StringValue("PN-1001"))
	props.Set(attr.SummaryInformation).Put("Title", attr.StringValue("Mounting Bracket"))

	// Create the drawing through the wire create path: this proves the registered
	// content factory yields live drawing content, and leaves the drawing active.
	var info wire.DocumentInfo
	call(t, r, s, "documents.create", `{"type":"drawing","name":"sheet.odd"}`, &info)
	if info.Type != "drawing" {
		t.Fatalf("created document type = %q, want drawing", info.Type)
	}
	return r, s
}

func TestDrawingDefaultSheetOverWire(t *testing.T) {
	r, s := drawingSession(t)

	var sheets wire.ListSheetsResult
	call(t, r, s, "drawing.listSheets", "{}", &sheets)
	if len(sheets.Sheets) != 1 {
		t.Fatalf("default sheet count = %d, want 1", len(sheets.Sheets))
	}
	sh := sheets.Sheets[0]
	if sh.Size != "a3" || sh.Orientation != "landscape" || sh.WidthMM != 420 || !sh.Active {
		t.Errorf("default sheet = %+v, want active A3 landscape 420 wide", sh)
	}
	if !sh.HasBorder || !sh.HasTitleBlock {
		t.Errorf("default sheet = %+v, want a border and title block", sh)
	}
}

func TestDrawingSheetLifecycleOverWire(t *testing.T) {
	r, s := drawingSession(t)

	var added wire.SheetResult
	call(t, r, s, "drawing.addSheet", `{"size":"a4","orientation":"portrait"}`, &added)
	if added.Sheet.Name != "Sheet:2" || added.Sheet.HeightMM != 297 || !added.Sheet.Active {
		t.Fatalf("added = %+v, want active A4 Sheet:2", added.Sheet)
	}

	var activated wire.SheetResult
	call(t, r, s, "drawing.setActiveSheet", `{"name":"Sheet:1"}`, &activated)
	if !activated.Sheet.Active || activated.Sheet.Name != "Sheet:1" {
		t.Errorf("activated = %+v, want Sheet:1 active", activated.Sheet)
	}

	var afterRemove wire.ListSheetsResult
	call(t, r, s, "drawing.removeSheet", `{"name":"Sheet:2"}`, &afterRemove)
	if len(afterRemove.Sheets) != 1 || afterRemove.Sheets[0].Name != "Sheet:1" {
		t.Errorf("after remove = %+v, want only Sheet:1", afterRemove.Sheets)
	}
}

// TestDrawingTitleBlockResolvesReferencedModel is the PBI-137 acceptance over the wire:
// a title block's fields resolve from the referenced model's iProperties.
func TestDrawingTitleBlockResolvesReferencedModel(t *testing.T) {
	r, s := drawingSession(t)

	var ref wire.SetModelReferenceResult
	call(t, r, s, "drawing.setModelReference", `{"fullDocumentName":"widget.opd"}`, &ref)
	if ref.ModelReference != "widget.opd" {
		t.Fatalf("model reference = %q, want widget.opd", ref.ModelReference)
	}

	var fields wire.TitleBlockFieldsResult
	call(t, r, s, "drawing.titleBlockFields", "{}", &fields)
	got := map[string]string{}
	for _, f := range fields.Fields {
		got[f.Name] = f.Value
	}
	if got["Part Number"] != "PN-1001" {
		t.Errorf("Part Number = %q, want PN-1001 (fields=%+v)", got["Part Number"], fields.Fields)
	}
	if got["Title"] != "Mounting Bracket" {
		t.Errorf("Title = %q, want Mounting Bracket", got["Title"])
	}
}

func TestDrawingTitleBlockBlankWithoutReference(t *testing.T) {
	r, s := drawingSession(t)
	var fields wire.TitleBlockFieldsResult
	call(t, r, s, "drawing.titleBlockFields", "{}", &fields)
	for _, f := range fields.Fields {
		if f.Value != "" {
			t.Errorf("field %q = %q without a model reference, want empty", f.Name, f.Value)
		}
	}
}
