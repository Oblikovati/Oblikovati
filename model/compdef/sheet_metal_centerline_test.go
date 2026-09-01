// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"testing"

	gmath "oblikovati.org/math"
)

// TestCosmeticCenterlines add, list, delete, and persist cosmetic centerlines.
func TestCosmeticCenterlines(t *testing.T) {
	t.Parallel()
	d := NewPartComponentDefinition()
	if _, err := d.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	if len(d.CosmeticCenterlines()) != 0 {
		t.Fatal("a fresh part should have no centerlines")
	}
	d.AddCosmeticCenterline(gmath.P2(0, 0), gmath.P2(4, 0))
	i := d.AddCosmeticCenterline(gmath.P2(2, -1), gmath.P2(2, 1))
	if i != 1 || len(d.CosmeticCenterlines()) != 2 {
		t.Fatalf("after adds: index=%d, count=%d, want 1 and 2", i, len(d.CosmeticCenterlines()))
	}

	if err := d.DeleteCosmeticCenterline(5); err == nil {
		t.Error("deleting an out-of-range centerline must error")
	}
	if err := d.DeleteCosmeticCenterline(0); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got := d.CosmeticCenterlines()
	if len(got) != 1 || got[0].Start != gmath.P2(2, -1) {
		t.Errorf("after delete = %+v, want the vertical centerline", got)
	}

	blob, _ := d.MarshalRecipe()
	dst := NewPartComponentDefinition()
	if err := dst.ApplyRecipe(blob); err != nil {
		t.Fatalf("apply: %v", err)
	}
	r := dst.CosmeticCenterlines()
	if len(r) != 1 || r[0].Start != gmath.P2(2, -1) || r[0].End != gmath.P2(2, 1) {
		t.Errorf("restored centerlines = %+v, want the vertical one", r)
	}
}
