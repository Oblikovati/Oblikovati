// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestAddNotePlainText: a note with no leader is just a text label.
func TestAddNotePlainText(t *testing.T) {
	c := NewContent()
	n, err := c.Sheets().Active().Annotations().AddNote("N", 100, 100, "DEBURR ALL EDGES", 0, 0)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if n.Kind() != types.DrawingNoteAnnotation || n.CurveCount() != 0 {
		t.Fatalf("plain note = (%v, %d curves), want a drawingNote with no leader", n.Kind(), n.CurveCount())
	}
	if len(n.Labels()) != 1 || n.Labels()[0].Text != "DEBURR ALL EDGES" {
		t.Errorf("note label = %v, want the note text", n.Labels())
	}
}

// TestAddNoteWithLeader: a note with a leader target draws a leader line + arrowhead to it.
func TestAddNoteWithLeader(t *testing.T) {
	c := NewContent()
	n, err := c.Sheets().Active().Annotations().AddNote("N", 100, 100, "SECTION A-A", 140, 130)
	if err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if n.CurveCount() < 2 { // leader line + at least one arrowhead barb
		t.Errorf("leader note curves = %d, want a leader line + arrowhead", n.CurveCount())
	}
}

// TestNoteNeedsText: an empty note errors.
func TestNoteNeedsText(t *testing.T) {
	c := NewContent()
	if _, err := c.Sheets().Active().Annotations().AddNote("N", 0, 0, "", 0, 0); err == nil {
		t.Error("AddNote with empty text = ok, want error")
	}
}

// TestAddCustomTable: a custom table lists its headers + rows in a grid and reports the row count.
func TestAddCustomTable(t *testing.T) {
	c := NewContent()
	headers := []string{"PARAM", "VALUE"}
	rows := [][]string{{"width", "60 mm"}, {"height", "40 mm"}}
	ct, err := c.Sheets().Active().Annotations().AddCustomTable("CT", 250, 60, headers, rows)
	if err != nil {
		t.Fatalf("AddCustomTable: %v", err)
	}
	if ct.Kind() != types.CustomTableAnnotation || ct.RowCount() != 2 || ct.CurveCount() == 0 {
		t.Fatalf("custom table = (%v, %d rows, %d curves), want a customTable with 2 rows + grid", ct.Kind(), ct.RowCount(), ct.CurveCount())
	}
	texts := map[string]bool{}
	for _, l := range ct.Labels() {
		texts[l.Text] = true
	}
	for _, want := range []string{"PARAM", "VALUE", "width", "60 mm"} {
		if !texts[want] {
			t.Errorf("custom table labels missing %q: %v", want, ct.Labels())
		}
	}
}

// TestCustomTableNeedsHeaders: a custom table with no columns errors.
func TestCustomTableNeedsHeaders(t *testing.T) {
	c := NewContent()
	if _, err := c.Sheets().Active().Annotations().AddCustomTable("CT", 0, 0, nil, nil); err == nil {
		t.Error("AddCustomTable with no headers = ok, want error")
	}
}

// TestCustomTablePadsRaggedRows: a row shorter than the header count is padded so it still aligns.
func TestCustomTablePadsRaggedRows(t *testing.T) {
	c := NewContent()
	headers := []string{"A", "B", "C"}
	ct, err := c.Sheets().Active().Annotations().AddCustomTable("CT", 0, 0, headers, [][]string{{"only-one"}})
	if err != nil {
		t.Fatalf("AddCustomTable: %v", err)
	}
	if ct.RowCount() != 1 {
		t.Errorf("ragged custom table rowCount = %d, want 1", ct.RowCount())
	}
}

// TestNoteAndCustomTablePersist: notes and custom tables are user data, so they survive a
// save/open round-trip.
func TestNoteAndCustomTablePersist(t *testing.T) {
	c := NewContent()
	if _, err := c.Sheets().Active().Annotations().AddNote("N", 100, 100, "DEBURR", 140, 130); err != nil {
		t.Fatalf("AddNote: %v", err)
	}
	if _, err := c.Sheets().Active().Annotations().AddCustomTable("CT", 250, 60, []string{"K", "V"}, [][]string{{"a", "1"}}); err != nil {
		t.Fatalf("AddCustomTable: %v", err)
	}
	data, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	restored := NewContent()
	if err := restored.ApplyRecipe(data); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	an := restored.Sheets().Active().Annotations()
	if an.Count() != 2 {
		t.Fatalf("restored = %d annotations, want 2 (note + custom table)", an.Count())
	}
	texts := map[string]bool{}
	for i := 0; i < an.Count(); i++ {
		for _, l := range an.Item(i).Labels() {
			texts[l.Text] = true
		}
	}
	if !texts["DEBURR"] || !texts["K"] || !texts["a"] {
		t.Errorf("restored note/table lost content: %v", texts)
	}
}
