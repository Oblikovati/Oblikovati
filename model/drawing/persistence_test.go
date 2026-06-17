// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence"
)

// TestDrawingSurvivesStoreRoundTrip proves the document/persistence seam: a drawing
// created through the workspace (so the registered content factory yields live content)
// saves to a .odd package and reopens with its sheets and model reference intact.
func TestDrawingSurvivesStoreRoundTrip(t *testing.T) {
	store := persistence.NewPackageStore()
	path := filepath.Join(t.TempDir(), "drawing.odd")

	saveWS := doc.NewWorkspace(store)
	d, err := saveWS.Add(doc.Drawing, path, true)
	if err != nil {
		t.Fatalf("Add(Drawing): %v", err)
	}
	c, ok := d.Content().(*Content)
	if !ok {
		t.Fatalf("content is %T, want *drawing.Content (factory not registered?)", d.Content())
	}
	c.SetModelReference("widget.opd")
	if _, err := c.Sheets().Add(SheetSpec{Name: "Detail", Size: types.SheetSizeCustom, WidthMM: 500, HeightMM: 350}); err != nil {
		t.Fatalf("Add sheet: %v", err)
	}
	if err := saveWS.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reopened, err := doc.NewWorkspace(store).Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	rc, ok := reopened.Content().(*Content)
	if !ok {
		t.Fatalf("reopened content is %T, want *drawing.Content", reopened.Content())
	}
	if rc.ModelReference() != "widget.opd" {
		t.Errorf("reopened model reference = %q, want widget.opd", rc.ModelReference())
	}
	if rc.Sheets().Count() != 2 {
		t.Fatalf("reopened sheet count = %d, want 2 (default + Detail)", rc.Sheets().Count())
	}
	detail, ok := rc.Sheets().ByName("Detail")
	if !ok || detail.Size() != types.SheetSizeCustom || detail.WidthMM() != 500 {
		t.Errorf("reopened Detail sheet = %+v, want custom 500 wide", detail)
	}
}
