// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"oblikovati.org/api/types"
)

// TestDocumentSketchSettings pins the per-document sketch settings accessors (#147): defaults when
// unset, SetSketchSettings marks dirty + stores, RestoreSketchSettings stores without dirtying.
func TestDocumentSketchSettings(t *testing.T) {
	d := newDocument(Part, "t.obk", nil, true)
	if got := d.SketchSettings(); got != types.DefaultSketchSettings() {
		t.Errorf("unset settings = %+v, want defaults", got)
	}
	if d.SketchSettingsSet() {
		t.Error("a fresh document should report no explicit sketch settings")
	}

	set := types.SketchSettings{InferConstraints: false, AutoApplyConstraints: true, ConstraintPriority: types.PriorityNone}
	d.ClearDirty()
	d.SetSketchSettings(set)
	if !d.Dirty() {
		t.Error("SetSketchSettings should mark the document dirty")
	}
	if !d.SketchSettingsSet() || d.SketchSettings() != set {
		t.Errorf("SetSketchSettings did not store %+v (got %+v)", set, d.SketchSettings())
	}

	d.ClearDirty()
	d.RestoreSketchSettings(types.DefaultSketchSettings())
	if d.Dirty() {
		t.Error("RestoreSketchSettings should NOT mark the document dirty (load path)")
	}
	if d.SketchSettings() != types.DefaultSketchSettings() {
		t.Errorf("RestoreSketchSettings did not store the value: %+v", d.SketchSettings())
	}
}
