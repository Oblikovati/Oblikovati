// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/display"
)

// TestDocumentDisplaySettings checks the per-document display settings store, that Set marks the
// document dirty, and that Restore (the load path) does not (M16-F07 #643).
func TestDocumentDisplaySettings(t *testing.T) {
	d := newDocument(Part, "p.obk", nil, true)
	if _, ok := d.DisplaySettings(); ok {
		t.Error("a fresh document should have no display settings")
	}

	set := display.DefaultSettings()
	set.BackgroundType = types.OneColorBackground
	d.SetDisplaySettings(set)
	if !d.Dirty() {
		t.Error("SetDisplaySettings should mark the document dirty")
	}
	got, ok := d.DisplaySettings()
	if !ok || got.BackgroundType != types.OneColorBackground {
		t.Errorf("DisplaySettings = (%+v, %v), want the stored value", got, ok)
	}

	d.ClearDirty()
	set.BackgroundType = types.GradientBackground
	d.RestoreDisplaySettings(set)
	if d.Dirty() {
		t.Error("RestoreDisplaySettings should NOT mark the document dirty (load path)")
	}
	if got, _ := d.DisplaySettings(); got.BackgroundType != types.GradientBackground {
		t.Errorf("RestoreDisplaySettings did not store the value: %+v", got)
	}
}
