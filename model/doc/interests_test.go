// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestInterestRegistryRules: identity is (ClientID, Name); re-adding updates
// in place; HasInterest matches either id or name; removal reports existence.
func TestInterestRegistryRules(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	d, _ := ws.Add(Part, "bracket.obk", true)
	d.ClearDirty()

	if err := d.Interests().Add(types.DocumentInterestRecord{ClientID: "", Name: "x"}); err == nil {
		t.Error("an interest without a client id must fail")
	}
	rec := types.DocumentInterestRecord{
		ClientID: "com.x.toolpaths", Name: "toolpath-recipes", DataVersion: 1, ClientData: "v1",
	}
	if err := d.Interests().Add(rec); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !d.Dirty() {
		t.Error("registering an interest must mark the document dirty")
	}
	got := d.Interests().Records()
	if len(got) != 1 || got[0].InterestType != types.Interested {
		t.Fatalf("records = %+v, want one defaulted to interested", got)
	}

	rec.DataVersion = 2
	if err := d.Interests().Add(rec); err != nil {
		t.Fatalf("re-Add: %v", err)
	}
	if got := d.Interests().Records(); len(got) != 1 || got[0].DataVersion != 2 {
		t.Errorf("records after update = %+v, want the version-2 record in place", got)
	}

	if !d.Interests().HasInterest("com.x.toolpaths") || !d.Interests().HasInterest("toolpath-recipes") {
		t.Error("HasInterest must match the client id and the interest name")
	}
	if d.Interests().HasInterest("com.x.ghost") {
		t.Error("HasInterest must miss unknown clients")
	}

	view, err := d.Interests().At(0)
	if err != nil || view.ClientID() != "com.x.toolpaths" || view.DataVersion() != 2 {
		t.Errorf("At(0) = (%+v, %v), want the scalar view of the record", view, err)
	}
	if _, err := d.Interests().At(5); err == nil {
		t.Error("At must reject an out-of-range index")
	}

	if !d.Interests().Remove("com.x.toolpaths", "toolpath-recipes") {
		t.Error("Remove must report the record existed")
	}
	if d.Interests().Remove("com.x.toolpaths", "toolpath-recipes") {
		t.Error("Remove must miss an already-removed record")
	}
}
