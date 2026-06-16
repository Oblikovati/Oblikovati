// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"math"
	"testing"

	"oblikovati.org/model/sheetmetal"
)

// TestFlatOrientationsRoundTrip orientations (and which is active) persist through the recipe.
func TestFlatOrientationsRoundTrip(t *testing.T) {
	src := NewPartComponentDefinition()
	if _, err := src.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	if err := src.FlatOrientations().Add(&sheetmetal.FlatPatternOrientation{
		Name: "Long Edge", AlignmentType: sheetmetal.VerticalAlignment, AlignmentRotation: 0.5, FlipBaseFace: true,
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := src.FlatOrientations().Activate("Long Edge"); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	blob, err := src.MarshalRecipe()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewPartComponentDefinition()
	if err := dst.ApplyRecipe(blob); err != nil {
		t.Fatalf("apply: %v", err)
	}

	set := dst.FlatOrientations()
	if set == nil || len(set.List()) != 2 {
		t.Fatalf("restored orientations = %v, want 2", set)
	}
	if set.Active().Name != "Long Edge" {
		t.Errorf("restored active = %q, want Long Edge", set.Active().Name)
	}
	long, _, ok := set.ByName("Long Edge")
	if !ok || long.AlignmentType != sheetmetal.VerticalAlignment || math.Abs(long.AlignmentRotation-0.5) > 1e-9 || !long.FlipBaseFace {
		t.Errorf("restored orientation = %+v, want vertical / rot 0.5 / flipBaseFace", long)
	}
}

// TestFlatLengthWidthSwapsWithAlignment a vertical orientation swaps the flat's length and
// width relative to the horizontal default.
func TestFlatLengthWidthSwapsWithAlignment(t *testing.T) {
	d, _ := sheetWithFlange(t)
	lh, wh, err := d.FlatLengthWidth(d.FlatOrientations().Active()) // default horizontal
	if err != nil {
		t.Fatalf("FlatLengthWidth: %v", err)
	}
	if lh <= 0 || wh <= 0 {
		t.Fatalf("length/width must be positive: %g × %g", lh, wh)
	}
	if err := d.FlatOrientations().Add(&sheetmetal.FlatPatternOrientation{Name: "V", AlignmentType: sheetmetal.VerticalAlignment}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	v, _, _ := d.FlatOrientations().ByName("V")
	lv, wv, err := d.FlatLengthWidth(v)
	if err != nil {
		t.Fatalf("FlatLengthWidth(V): %v", err)
	}
	if math.Abs(lv-wh) > 1e-9 || math.Abs(wv-lh) > 1e-9 {
		t.Errorf("vertical (%g×%g) should swap horizontal (%g×%g)", lv, wv, lh, wh)
	}
}
