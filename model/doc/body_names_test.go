// SPDX-License-Identifier: GPL-2.0-only

package doc

import "testing"

// TestBodyNameStoreAndClear pins the per-body name accessors (#1078): set marks dirty and reads
// back; an empty name clears the entry; a renamed body reverts to "no stored name".
func TestBodyNameStoreAndClear(t *testing.T) {
	d := NewPartDocument("/proj/case.obk")
	if _, ok := d.BodyName("k1"); ok {
		t.Error("a fresh document should report no stored name for any body")
	}

	d.ClearDirty()
	d.SetBodyName("k1", "Housing")
	if !d.Dirty() {
		t.Error("SetBodyName should mark the document dirty (names round-trip in the .obk)")
	}
	if name, ok := d.BodyName("k1"); !ok || name != "Housing" {
		t.Errorf("BodyName(k1) = (%q, %v), want (Housing, true)", name, ok)
	}

	d.SetBodyName("k1", "") // empty reverts to the default
	if _, ok := d.BodyName("k1"); ok {
		t.Error("an empty name should clear the stored entry")
	}
}

// TestBodyNamesCopyAndRestore: BodyNames returns a defensive copy, and RestoreBodyNames installs a
// map without dirtying the document (the load path, where memory already matches disk).
func TestBodyNamesCopyAndRestore(t *testing.T) {
	d := NewPartDocument("/proj/case.obk")
	if d.BodyNames() != nil {
		t.Error("BodyNames on a document with no renamed body should be nil")
	}

	d.SetBodyName("k1", "Lid")
	snap := d.BodyNames()
	snap["k1"] = "tampered" // mutating the snapshot must not affect the document
	if name, _ := d.BodyName("k1"); name != "Lid" {
		t.Errorf("BodyNames returned a live map: document name became %q", name)
	}

	d2 := NewPartDocument("/proj/other.obk")
	d2.ClearDirty()
	d2.RestoreBodyNames(map[string]string{"a": "Boss", "b": "Rib"})
	if d2.Dirty() {
		t.Error("RestoreBodyNames must not dirty the document (it is the load path)")
	}
	if name, ok := d2.BodyName("a"); !ok || name != "Boss" {
		t.Errorf("after restore, BodyName(a) = (%q, %v), want (Boss, true)", name, ok)
	}
	d2.RestoreBodyNames(nil) // restoring empty clears the store
	if d2.BodyNames() != nil {
		t.Error("RestoreBodyNames(nil) should leave no stored names")
	}
}
