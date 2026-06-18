// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestAddRevisionTable: a revision table lists its rows in a grid and reports the row count.
func TestAddRevisionTable(t *testing.T) {
	c := NewContent()
	rows := []RevisionRow{
		{Revision: "A", Date: "2026-06-01", Description: "Initial release"},
		{Revision: "B", Date: "2026-06-18", Description: "Added hole pattern"},
	}
	rt, err := c.Sheets().Active().Annotations().AddRevisionTable("RT", 250, 60, rows)
	if err != nil {
		t.Fatalf("AddRevisionTable: %v", err)
	}
	if rt.Kind() != types.RevisionTableAnnotation || rt.RowCount() != 2 || rt.CurveCount() == 0 {
		t.Fatalf("revision table = (%v, %d rows, %d curves), want a revisionTable with 2 rows + grid", rt.Kind(), rt.RowCount(), rt.CurveCount())
	}
	texts := map[string]bool{}
	for _, l := range rt.Labels() {
		texts[l.Text] = true
	}
	for _, want := range []string{"REV", "DATE", "DESCRIPTION", "B", "Added hole pattern"} {
		if !texts[want] {
			t.Errorf("revision table labels missing %q: %v", want, rt.Labels())
		}
	}
}

// TestRevisionTableNeedsRows: an empty revision table errors.
func TestRevisionTableNeedsRows(t *testing.T) {
	c := NewContent()
	if _, err := c.Sheets().Active().Annotations().AddRevisionTable("RT", 0, 0, nil); err == nil {
		t.Error("AddRevisionTable with no rows = ok, want error")
	}
}

// TestAddRevisionTag: a revision tag is a triangle holding the revision letter.
func TestAddRevisionTag(t *testing.T) {
	c := NewContent()
	tag, err := c.Sheets().Active().Annotations().AddRevisionTag("RT1", 120, 90, "B")
	if err != nil {
		t.Fatalf("AddRevisionTag: %v", err)
	}
	if tag.Kind() != types.RevisionTagAnnotation || tag.Tag() != "B" || tag.CurveCount() != 3 {
		t.Fatalf("revision tag = (%v, tag %q, %d curves), want a revisionTag triangle holding B", tag.Kind(), tag.Tag(), tag.CurveCount())
	}
	if len(tag.Labels()) != 1 || tag.Labels()[0].Text != "B" {
		t.Errorf("revision tag label = %v, want one label B", tag.Labels())
	}
}

// TestRevisionTagNeedsRevision: a revision tag with no letter errors.
func TestRevisionTagNeedsRevision(t *testing.T) {
	c := NewContent()
	if _, err := c.Sheets().Active().Annotations().AddRevisionTag("RT", 0, 0, ""); err == nil {
		t.Error("AddRevisionTag with no revision = ok, want error")
	}
}

// TestRevisionTablePersistsRows: a revision table's rows are user data, so they must survive a
// save/open round-trip (unlike model-derived tables, which re-derive).
func TestRevisionTablePersistsRows(t *testing.T) {
	c := NewContent()
	rows := []RevisionRow{{Revision: "A", Date: "2026-06-01", Description: "Initial release"}}
	if _, err := c.Sheets().Active().Annotations().AddRevisionTable("RT", 250, 60, rows); err != nil {
		t.Fatalf("AddRevisionTable: %v", err)
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
	if an.Count() != 1 || an.Item(0).RowCount() != 1 {
		t.Fatalf("restored revision table = %d annotations / %d rows, want 1/1", an.Count(), rowCountOf(an))
	}
	texts := map[string]bool{}
	for _, l := range an.Item(0).Labels() {
		texts[l.Text] = true
	}
	if !texts["Initial release"] {
		t.Errorf("restored revision table lost its row text: %v", an.Item(0).Labels())
	}
}

// rowCountOf is a guard helper for the round-trip assertion when the table is missing.
func rowCountOf(an *DrawingAnnotations) int {
	if an.Count() == 0 {
		return 0
	}
	return an.Item(0).RowCount()
}
