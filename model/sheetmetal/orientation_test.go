// SPDX-License-Identifier: GPL-2.0-only

package sheetmetal

import "testing"

// TestOrientationsSeededWithDefault a fresh set has the active default orientation.
func TestOrientationsSeededWithDefault(t *testing.T) {
	o := NewOrientations()
	if len(o.List()) != 1 || o.Active().Name != DefaultOrientationName {
		t.Fatalf("seeded set = %d items, active %q; want 1 default", len(o.List()), o.Active().Name)
	}
}

// TestOrientationsAddActivate add a named orientation, activate it, and confirm it is current.
func TestOrientationsAddActivate(t *testing.T) {
	o := NewOrientations()
	if err := o.Add(&FlatPatternOrientation{Name: "Long Edge"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if o.IsActive(o.List()[1]) {
		t.Error("new orientation should not auto-activate")
	}
	if err := o.Activate("Long Edge"); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if o.Active().Name != "Long Edge" {
		t.Errorf("active = %q, want Long Edge", o.Active().Name)
	}
	if err := o.Add(&FlatPatternOrientation{Name: "Long Edge"}); err == nil {
		t.Error("duplicate name must error")
	}
	if err := o.Add(&FlatPatternOrientation{Name: ""}); err == nil {
		t.Error("blank name must error")
	}
}

// TestOrientationsCopy duplicates an orientation's fields under a new name.
func TestOrientationsCopy(t *testing.T) {
	o := NewOrientations()
	_ = o.Add(&FlatPatternOrientation{Name: "A", AlignmentType: VerticalAlignment, FlipBaseFace: true})
	dup, err := o.Copy("A", "B")
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}
	if dup.Name != "B" || dup.AlignmentType != VerticalAlignment || !dup.FlipBaseFace {
		t.Errorf("copy = %+v, want B with A's fields", dup)
	}
	if _, err := o.Copy("missing", "C"); err == nil {
		t.Error("copy of an unknown orientation must error")
	}
}

// TestOrientationsDelete delete a custom orientation; the default is undeletable; deleting the
// active falls back to the default.
func TestOrientationsDelete(t *testing.T) {
	o := NewOrientations()
	_ = o.Add(&FlatPatternOrientation{Name: "Temp"})
	_ = o.Activate("Temp")
	if err := o.Delete(DefaultOrientationName); err == nil {
		t.Error("deleting the default orientation must error")
	}
	if err := o.Delete("Temp"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(o.List()) != 1 || o.Active().Name != DefaultOrientationName {
		t.Errorf("after delete: %d items, active %q; want default active", len(o.List()), o.Active().Name)
	}
	if err := o.Delete("Temp"); err == nil {
		t.Error("deleting an unknown orientation must error")
	}
}
