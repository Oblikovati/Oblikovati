// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

// TestDrawingZonedBorderOverWire drives the #1989 sheet-authoring handlers end to end over the
// wire: a zoned border, a moved title block, a revision stamp, then a reusable format and a sheet
// stamped from it. It asserts the handlers' effects surface on SheetInfo.
func TestDrawingZonedBorderOverWire(t *testing.T) {
	t.Parallel()
	r, s := drawingSession(t)

	var bordered wire.SheetResult
	call(t, r, s, wire.MethodDrawingAddDefaultBorder,
		mustJSON(t, wire.AddDefaultBorderArgs{HZones: 8, VZones: 6, HLabelMode: "numeric", VLabelMode: "alphabetical"}),
		&bordered)
	if !bordered.Sheet.HasBorder || bordered.Sheet.BorderHZones != 8 || bordered.Sheet.BorderVZones != 6 {
		t.Fatalf("bordered sheet = %+v, want 8x6 zoned border", bordered.Sheet)
	}

	var titled wire.SheetResult
	call(t, r, s, wire.MethodDrawingSetTitleBlock, mustJSON(t, wire.SetTitleBlockArgs{Location: "topLeft"}), &titled)
	if !titled.Sheet.HasTitleBlock || titled.Sheet.TitleBlockLocation != "topLeft" {
		t.Errorf("titled sheet = %+v, want title block at topLeft", titled.Sheet)
	}

	var revised wire.SheetResult
	call(t, r, s, wire.MethodDrawingSetSheetRevision, mustJSON(t, wire.SetSheetRevisionArgs{Revision: "C"}), &revised)
	if revised.Sheet.Revision != "C" {
		t.Errorf("revised sheet revision = %q, want C", revised.Sheet.Revision)
	}
}

// TestDrawingSheetFormatOverWire registers a reusable sheet format and stamps a new sheet from it,
// proving the format's size/orientation/zones travel onto the created sheet (#1989).
func TestDrawingSheetFormatOverWire(t *testing.T) {
	t.Parallel()
	r, s := drawingSession(t)

	call(t, r, s, wire.MethodDrawingDefineSheetFormat, mustJSON(t, wire.DefineSheetFormatArgs{
		Name: "titled-a4", Size: "a4", Orientation: "portrait",
		HZones: 4, VZones: 3, HLabelMode: "numeric", VLabelMode: "alphabetical", TitleBlockLocation: "bottomRight",
	}), nil)

	var stamped wire.SheetResult
	call(t, r, s, wire.MethodDrawingAddSheetUsingFormat, mustJSON(t, wire.AddSheetUsingFormatArgs{Format: "titled-a4"}), &stamped)
	if stamped.Sheet.Size != "a4" || stamped.Sheet.Orientation != "portrait" {
		t.Fatalf("stamped sheet = %+v, want A4 portrait", stamped.Sheet)
	}
	if stamped.Sheet.BorderHZones != 4 || !stamped.Sheet.HasTitleBlock {
		t.Errorf("stamped sheet = %+v, want the format's 4 zones + title block", stamped.Sheet)
	}
}

// TestDrawingSheetAuthoringRejectsBadSpellings covers the handlers' parse-error branches: unknown
// label modes, title-block corner, sheet size/orientation, and a nameless format (#1989).
func TestDrawingSheetAuthoringRejectsBadSpellings(t *testing.T) {
	t.Parallel()
	r, s := drawingSession(t)
	cases := []struct {
		name, method, args string
	}{
		{"bad h-label", wire.MethodDrawingAddDefaultBorder, `{"hZones":2,"vZones":2,"hLabelMode":"roman"}`},
		{"bad title corner", wire.MethodDrawingSetTitleBlock, `{"location":"middle"}`},
		{"bad format size", wire.MethodDrawingDefineSheetFormat, `{"name":"f","size":"a99"}`},
		{"bad format orient", wire.MethodDrawingDefineSheetFormat, `{"name":"f","orientation":"sideways"}`},
		{"nameless format", wire.MethodDrawingDefineSheetFormat, `{"size":"a4"}`},
		{"unknown format ref", wire.MethodDrawingAddSheetUsingFormat, `{"format":"nope"}`},
	}
	for _, c := range cases {
		if _, err := r.Handle(s, c.method, []byte(c.args)); err == nil {
			t.Errorf("%s: %s(%s) succeeded, want an error", c.name, c.method, c.args)
		}
	}
}
