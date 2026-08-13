// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"reflect"
	"testing"

	"oblikovati.org/api/types"
)

// TestZonedBorderLabels: a zoned border emits its column labels per the horizontal mode and its row
// labels per the vertical mode — numeric 1.. across, alphabetical A.. down (#1989).
func TestZonedBorderLabels(t *testing.T) {
	c := NewContent()
	sh := c.Sheets().Active()
	if err := sh.SetZonedBorder(4, 3, types.NumericBorderLabel, types.AlphabeticalBorderLabel); err != nil {
		t.Fatalf("SetZonedBorder: %v", err)
	}
	b := sh.border
	if h, v := b.ZoneCounts(); h != 4 || v != 3 {
		t.Fatalf("zone counts = %d×%d, want 4×3", h, v)
	}
	if cols := b.ColumnLabels(); !reflect.DeepEqual(cols, []string{"1", "2", "3", "4"}) {
		t.Errorf("column labels = %v, want [1 2 3 4]", cols)
	}
	if rows := b.RowLabels(); !reflect.DeepEqual(rows, []string{"A", "B", "C"}) {
		t.Errorf("row labels = %v, want [A B C]", rows)
	}
	// 4 columns → 3 interior vertical lines; 3 rows → 2 horizontal → 5 division lines.
	if div := b.ZoneDivisions(sh.WidthMM(), sh.HeightMM()); len(div) != 5 {
		t.Errorf("zone divisions = %d, want 5 (3 vertical + 2 horizontal)", len(div))
	}
}

// TestAlphaLabelWraps: the alphabetical labels wrap A..Z then AA.. past 26 zones.
func TestAlphaLabelWraps(t *testing.T) {
	cases := map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB"}
	for i, want := range cases {
		if got := alphaLabel(i); got != want {
			t.Errorf("alphaLabel(%d) = %q, want %q", i, got, want)
		}
	}
}

// TestZonedBorderRejectsNonPositive: zone counts below 1 error.
func TestZonedBorderRejectsNonPositive(t *testing.T) {
	sh := NewContent().Sheets().Active()
	if err := sh.SetZonedBorder(0, 3, types.NumericBorderLabel, types.AlphabeticalBorderLabel); err == nil {
		t.Error("SetZonedBorder(0,3) = ok, want error")
	}
}

// TestSheetRevisionAndTitleBlockLocation: a sheet carries a revision string and its title block moves
// to the chosen corner (#1989).
func TestSheetRevisionAndTitleBlockLocation(t *testing.T) {
	sh := NewContent().Sheets().Active()
	sh.SetRevision("C")
	if sh.Revision() != "C" {
		t.Errorf("revision = %q, want C", sh.Revision())
	}
	sh.SetTitleBlockLocation(types.TopLeftTitleBlock)
	if sh.TitleBlockRef().Location() != types.TopLeftTitleBlock {
		t.Errorf("title block location = %v, want topLeft", sh.TitleBlockRef().Location())
	}
}

// TestAddSheetUsingFormat: a registered format stamps a new sheet with its size, zoned border and
// title-block corner (#1989).
func TestAddSheetUsingFormat(t *testing.T) {
	c := NewContent()
	sheets := c.Sheets()
	format := SheetFormat{
		Name: "TitleD", Size: types.SheetSizeA2, Orientation: types.SheetLandscape,
		HZones: 6, VZones: 4, HLabelMode: types.NumericBorderLabel, VLabelMode: types.AlphabeticalBorderLabel,
		TitleBlockLocation: types.BottomLeftTitleBlock,
	}
	if err := sheets.DefineFormat(format); err != nil {
		t.Fatalf("DefineFormat: %v", err)
	}
	sh, err := sheets.AddUsingFormat("Plate", "TitleD")
	if err != nil {
		t.Fatalf("AddUsingFormat: %v", err)
	}
	if sh.Name() != "Plate" || sh.Size() != types.SheetSizeA2 {
		t.Errorf("stamped sheet = (%q, %v), want (Plate, a2)", sh.Name(), sh.Size())
	}
	if h, v := sh.border.ZoneCounts(); h != 6 || v != 4 {
		t.Errorf("stamped border zones = %d×%d, want 6×4", h, v)
	}
	if sh.TitleBlockRef().Location() != types.BottomLeftTitleBlock {
		t.Errorf("stamped title block = %v, want bottomLeft", sh.TitleBlockRef().Location())
	}
	if _, err := sheets.AddUsingFormat("X", "Ghost"); err == nil {
		t.Error("AddUsingFormat with an unknown format = ok, want error")
	}
}

// TestSheetAuthoringSurvivesRoundTrip: revision, zoned border and title-block corner persist through
// the recipe (#1989).
func TestSheetAuthoringSurvivesRoundTrip(t *testing.T) {
	c := NewContent()
	sh := c.Sheets().Active()
	sh.SetRevision("B")
	if err := sh.SetZonedBorder(5, 2, types.NumericBorderLabel, types.AlphabeticalBorderLabel); err != nil {
		t.Fatalf("SetZonedBorder: %v", err)
	}
	sh.SetTitleBlockLocation(types.TopRightTitleBlock)

	model, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	rc := NewContent()
	if err := rc.ApplyRecipe(model); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	rsh := rc.Sheets().Active()
	if rsh.Revision() != "B" {
		t.Errorf("reopened revision = %q, want B", rsh.Revision())
	}
	if h, v := rsh.border.ZoneCounts(); h != 5 || v != 2 {
		t.Errorf("reopened border zones = %d×%d, want 5×2", h, v)
	}
	if rsh.TitleBlockRef().Location() != types.TopRightTitleBlock {
		t.Errorf("reopened title block = %v, want topRight", rsh.TitleBlockRef().Location())
	}
}
