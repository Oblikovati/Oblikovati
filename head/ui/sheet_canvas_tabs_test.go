//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/drawing"
)

// TestSheetTabLabelShowsNameWithStableID: a sheet tab displays just the sheet name, with a
// "###id" suffix so ImGui keys the tab uniquely without showing the id.
func TestSheetTabLabelShowsNameWithStableID(t *testing.T) {
	c := drawing.NewContent()
	sh := c.Sheets().Active()
	label := sheetTabLabel(sh)
	visible, _, found := strings.Cut(label, "###")
	if !found || visible != sh.Name() {
		t.Errorf("label %q: want visible text %q before a \"###\" id suffix", label, sh.Name())
	}
}

// TestSheetSwitchingChangesActiveSheet proves the canvas's switch path: activating another sheet
// changes which sheet Sheets().Active() returns (the sheet the canvas draws). This is the model
// behaviour a sheet-tab click invokes via SetActive.
func TestSheetSwitchingChangesActiveSheet(t *testing.T) {
	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Drawing, "deck.odd", true)
	if err != nil {
		t.Fatalf("add drawing: %v", err)
	}
	c := d.Content().(*drawing.Content)
	first := c.Sheets().Active().Name()
	second, err := c.Sheets().Add(drawing.SheetSpec{Size: types.SheetSizeA3, Orientation: types.SheetLandscape})
	if err != nil {
		t.Fatalf("add second sheet: %v", err)
	}
	if c.Sheets().Active().Name() != second.Name() {
		t.Fatalf("a new sheet should become active, got %q", c.Sheets().Active().Name())
	}

	if err := c.Sheets().SetActive(first); err != nil {
		t.Fatalf("switch back to %q: %v", first, err)
	}
	if got := c.Sheets().Active().Name(); got != first {
		t.Errorf("active sheet = %q after switching, want %q", got, first)
	}
}
