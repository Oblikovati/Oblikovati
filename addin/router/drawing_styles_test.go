// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/wire"
)

func TestDrawingStylesListAndDefaultISO(t *testing.T) {
	r, s := drawingSession(t)
	var list wire.ListStandardsResult
	call(t, r, s, "drawingStyles.listStandards", "{}", &list)
	if list.Active != "iso" || len(list.Standards) != 2 {
		t.Fatalf("standards = %+v, want active iso + 2 standards", list)
	}

	var style wire.StandardStyleResult
	call(t, r, s, "drawingStyles.getActiveStyle", "{}", &style)
	if style.Style.Standard != "iso" || style.Style.Dimension.Unit != "mm" || style.Style.Dimension.DecimalPlaces != 2 {
		t.Errorf("ISO style = %+v, want iso/mm/2dp", style.Style.Dimension)
	}
}

// TestDrawingStylesSwitchStandardOverWire is the PBI-138 acceptance over the wire:
// switching to ANSI returns the ANSI preset (inches, 3 decimals).
func TestDrawingStylesSwitchStandardOverWire(t *testing.T) {
	r, s := drawingSession(t)
	var switched wire.StandardStyleResult
	call(t, r, s, "drawingStyles.setStandard", `{"standard":"ansi"}`, &switched)
	if switched.Style.Standard != "ansi" || switched.Style.Dimension.Unit != "in" || switched.Style.Dimension.DecimalPlaces != 3 {
		t.Fatalf("after switch = %+v, want ansi/in/3dp", switched.Style.Dimension)
	}
	// A fresh getActiveStyle reflects the switch.
	var now wire.StandardStyleResult
	call(t, r, s, "drawingStyles.getActiveStyle", "{}", &now)
	if now.Style.Standard != "ansi" {
		t.Errorf("active style after switch = %q, want ansi", now.Style.Standard)
	}
}

func TestDrawingStylesRejectsUnknownStandard(t *testing.T) {
	r, s := drawingSession(t)
	if _, err := r.Handle(s, "drawingStyles.setStandard", []byte(`{"standard":"klingon"}`)); err == nil {
		t.Error("unknown standard should error")
	}
}
