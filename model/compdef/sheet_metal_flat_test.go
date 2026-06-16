// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sheetmetal"
)

// flatVolume is the developed flat body's mesh volume at a fine chord tolerance.
func flatVolume(body *topo.Body) float64 {
	return ops.BodyGeometryProperties(body, ops.Quality{ChordTolerance: 1e-3}).Volume
}

// developedTabLength is the expected flat tab length for the default-rule fixture: the bend
// allowance of a 90° bend plus the flange's straight run.
func developedTabLength(rule *sheetmetal.Rule, height float64) float64 {
	return rule.Unfold().BendAllowance(math.Pi/2, rule.BendRadius(), rule.Thickness()) + height
}

// TestUnfoldDevelopsWatertightFlat a flanged sheet unfolds to one watertight flat solid whose
// footprint is the base square plus the developed flange tab, and whose extents grow with the
// bend allowance — the flat-pattern acceptance criterion.
func TestUnfoldDevelopsWatertightFlat(t *testing.T) {
	const side, height = 4.0, 1.0
	d, _ := sheetWithFlange(t) // square side=4, 90° flange height=1, default rule

	fp, err := d.Unfold()
	if err != nil {
		t.Fatalf("Unfold: %v", err)
	}
	if !fp.Body.IsSolid() {
		t.Fatal("flat pattern body is not a solid")
	}
	if open := ops.BoundaryEdges(fp.Body); len(open) != 0 {
		t.Errorf("flat has %d boundary edges, want 0 (watertight)", len(open))
	}
	if r := ops.Validate(fp.Body); !r.Valid {
		t.Errorf("flat failed validation: %+v", r.Issues)
	}

	// Footprint: base side² plus the tab (side × developed length); volume = footprint × gauge.
	rule := d.SheetMetal()
	tab := developedTabLength(rule, height)
	wantVol := (side*side + side*tab) * rule.Thickness()
	if got := flatVolume(fp.Body); math.Abs(got-wantVol)/wantVol > 1e-3 {
		t.Errorf("flat volume = %.5f, want %.5f", got, wantVol)
	}

	// Extents: one side stays at the base width; the other grows by the developed tab.
	dx, dy := float64(fp.Extents.Diagonal().X), float64(fp.Extents.Diagonal().Y)
	long, short := math.Max(dx, dy), math.Min(dx, dy)
	if math.Abs(short-side) > 1e-6 {
		t.Errorf("flat short extent = %.5f, want %.5f", short, side)
	}
	if math.Abs(long-(side+tab)) > 1e-6 {
		t.Errorf("flat long extent = %.5f, want %.5f (side + developed tab)", long, side+tab)
	}
	if len(fp.Bends) != 1 || math.Abs(fp.Bends[0].Angle-math.Pi/2) > 1e-12 {
		t.Errorf("flat bends = %+v, want one 90° fold line", fp.Bends)
	}
}

// TestUnfoldTracksKFactor the flat is associative on the rule: raising the K-factor lengthens
// the bend allowance and so the developed extent, without recomputing the folded model.
func TestUnfoldTracksKFactor(t *testing.T) {
	d, _ := sheetWithFlange(t)
	tight, err := d.Unfold()
	if err != nil {
		t.Fatalf("Unfold (tight): %v", err)
	}
	d.SheetMetal().SetUnfold(sheetmetal.KFactorMethod(0.9)) // neutral axis nearer the outside
	loose, err := d.Unfold()
	if err != nil {
		t.Fatalf("Unfold (loose): %v", err)
	}
	if !(flatVolume(loose.Body) > flatVolume(tight.Body)) {
		t.Errorf("looser K-factor flat volume %.5f should exceed %.5f", flatVolume(loose.Body), flatVolume(tight.Body))
	}
}

// TestUnfoldRejectsNonSheetMetal and a sheet-metal part with no base Face both error clearly.
func TestUnfoldRejectsNonSheetMetal(t *testing.T) {
	plain := NewPartComponentDefinition()
	if _, err := plain.Unfold(); err == nil {
		t.Error("Unfold on a plain part must error")
	}
	empty := NewPartComponentDefinition()
	if _, err := empty.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	if _, err := empty.Unfold(); err == nil {
		t.Error("Unfold with no base Face must error")
	}
}
