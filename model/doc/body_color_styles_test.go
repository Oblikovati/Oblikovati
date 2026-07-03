// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"oblikovati.org/model/display"
)

// TestBodyColorStyleStoreAndClear pins the per-body color-style accessors (S5 #1640): set marks
// dirty and reads back; an empty style name clears the entry, reverting to "no assignment".
func TestBodyColorStyleStoreAndClear(t *testing.T) {
	d := NewPartDocument("/proj/case.obk")
	if _, ok := d.BodyColorStyle("k1"); ok {
		t.Error("a fresh document should report no color style for any body")
	}

	d.ClearDirty()
	d.SetBodyColorStyle("k1", "Red")
	if !d.Dirty() {
		t.Error("SetBodyColorStyle should mark the document dirty (styles round-trip in the .obk)")
	}
	if name, ok := d.BodyColorStyle("k1"); !ok || name != "Red" {
		t.Errorf("BodyColorStyle(k1) = (%q, %v), want (Red, true)", name, ok)
	}

	d.SetBodyColorStyle("k1", "") // empty reverts to "no assignment"
	if _, ok := d.BodyColorStyle("k1"); ok {
		t.Error("an empty style name should clear the stored entry")
	}
}

// TestBodyColorStylesCopyAndRestore: BodyColorStyles returns a defensive copy, and
// RestoreBodyColorStyles installs a map without dirtying (load and undo-restore paths).
func TestBodyColorStylesCopyAndRestore(t *testing.T) {
	d := NewPartDocument("/proj/case.obk")
	if d.BodyColorStyles() != nil {
		t.Error("BodyColorStyles on a document with no assignment should be nil")
	}

	d.SetBodyColorStyle("k1", "Blue")
	snap := d.BodyColorStyles()
	snap["k1"] = "tampered" // mutating the snapshot must not affect the document
	if name, _ := d.BodyColorStyle("k1"); name != "Blue" {
		t.Errorf("BodyColorStyles returned a live map: document style became %q", name)
	}

	restoreBodyColorStyles(t)
}

// restoreBodyColorStyles exercises RestoreBodyColorStyles' install and clear branches without
// dirtying (kept separate so both test functions stay within the funlen budget).
func restoreBodyColorStyles(t *testing.T) {
	t.Helper()
	d := NewPartDocument("/proj/other.obk")
	d.ClearDirty()
	d.RestoreBodyColorStyles(map[string]string{"a": "Green", "b": "Yellow"})
	if d.Dirty() {
		t.Error("RestoreBodyColorStyles must not dirty the document (load/undo path)")
	}
	if name, ok := d.BodyColorStyle("a"); !ok || name != "Green" {
		t.Errorf("after restore, BodyColorStyle(a) = (%q, %v), want (Green, true)", name, ok)
	}
	d.RestoreBodyColorStyles(nil) // restoring empty clears the store
	if d.BodyColorStyles() != nil {
		t.Error("RestoreBodyColorStyles(nil) should leave no stored styles")
	}
}

// TestClearSettingsRevertToDefaults covers the undo-restore clear paths (#1641): ClearDisplaySettings
// and ClearSketchSettings drop explicit settings back to "unset" without dirtying.
func TestClearSettingsRevertToDefaults(t *testing.T) {
	d := NewPartDocument("/proj/clr.obk")
	d.SetDisplaySettings(display.DefaultSettings())
	d.ClearDisplaySettings()
	if _, ok := d.DisplaySettings(); ok {
		t.Error("ClearDisplaySettings should leave the document with no explicit display settings")
	}

	d.SetSketchSettings(d.SketchSettings())
	if !d.SketchSettingsSet() {
		t.Fatal("SetSketchSettings should record explicit settings")
	}
	d.ClearSketchSettings()
	if d.SketchSettingsSet() {
		t.Error("ClearSketchSettings should revert to no explicit sketch settings")
	}
}
