// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

func TestNewContentDefaultsToISO(t *testing.T) {
	c := NewContent()
	if c.Styles().ActiveStandard() != types.DraftingISO {
		t.Errorf("default standard = %v, want ISO", c.Styles().ActiveStandard())
	}
	dim := c.Styles().ActiveStyle().DimensionStyle()
	if dim.Unit() != types.DimensionMillimeter || dim.DecimalPlaces() != 2 || dim.TextHeightMM() != 3.5 {
		t.Errorf("ISO dimension = %v/%ddp/%gmm, want mm/2/3.5", dim.Unit(), dim.DecimalPlaces(), dim.TextHeightMM())
	}
}

// TestSwitchStandardChangesAppearance is the PBI-138 acceptance: switching the standard
// re-points the active preset so the dimension appearance changes globally.
func TestSwitchStandardChangesAppearance(t *testing.T) {
	c := NewContent()
	c.Styles().SetActiveStandard(types.DraftingANSI)
	if c.Styles().ActiveStandard() != types.DraftingANSI {
		t.Fatalf("active standard = %v, want ANSI", c.Styles().ActiveStandard())
	}
	dim := c.Styles().ActiveStyle().DimensionStyle()
	if dim.Unit() != types.DimensionInch || dim.DecimalPlaces() != 3 {
		t.Errorf("ANSI dimension = %v/%ddp, want in/3", dim.Unit(), dim.DecimalPlaces())
	}
	if c.Styles().ActiveStyle().Standard() != types.DraftingANSI {
		t.Error("active style standard should be ANSI after switch")
	}
	// Switch back: appearance reverts.
	c.Styles().SetActiveStandard(types.DraftingISO)
	if c.Styles().ActiveStyle().DimensionStyle().Unit() != types.DimensionMillimeter {
		t.Error("switching back to ISO should restore millimetres")
	}
}

// TestStandardStyleExposesTextAndLineStyles covers the text/line style accessors of each
// preset (the ISO and ANSI presets carry their own named text and line styles).
func TestStandardStyleExposesTextAndLineStyles(t *testing.T) {
	c := NewContent()
	for _, std := range c.Styles().Standards() {
		c.Styles().SetActiveStandard(std)
		style := c.Styles().ActiveStyle()
		txt, line := style.TextStyle(), style.LineStyle()
		if txt.Name() == "" || txt.FontName() != "Arial" || txt.HeightMM() <= 0 {
			t.Errorf("%v text style = %q/%q/%g, want a named Arial style with positive height", std, txt.Name(), txt.FontName(), txt.HeightMM())
		}
		if line.Name() == "" || line.WeightMM() <= 0 {
			t.Errorf("%v line style = %q/%g, want a named style with positive weight", std, line.Name(), line.WeightMM())
		}
		dim := style.DimensionStyle()
		if dim.Name() == "" || dim.ArrowSizeMM() <= 0 || dim.LineWeightMM() <= 0 {
			t.Errorf("%v dimension style missing name/arrow size/line weight", std)
		}
	}
}

func TestStylesManagerListsBothStandards(t *testing.T) {
	got := NewContent().Styles().Standards()
	if len(got) != 2 || got[0] != types.DraftingISO || got[1] != types.DraftingANSI {
		t.Errorf("standards = %v, want [ISO ANSI]", got)
	}
}

func TestActiveStandardSurvivesRecipeRoundTrip(t *testing.T) {
	c := NewContent()
	c.Styles().SetActiveStandard(types.DraftingANSI)
	blob, err := c.MarshalRecipe()
	if err != nil {
		t.Fatalf("MarshalRecipe: %v", err)
	}
	restored := NewContent()
	if err := restored.ApplyRecipe(blob); err != nil {
		t.Fatalf("ApplyRecipe: %v", err)
	}
	if restored.Styles().ActiveStandard() != types.DraftingANSI {
		t.Errorf("restored standard = %v, want ANSI", restored.Styles().ActiveStandard())
	}
}
