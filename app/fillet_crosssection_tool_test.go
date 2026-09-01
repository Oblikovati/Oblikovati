// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/model/feature"
)

// TestFilletToolCrossSectionIndex round-trips the cross-section dropdown selection.
func TestFilletToolCrossSectionIndex(t *testing.T) {
	t.Parallel()
	f := NewFilletTool()
	if f.CrossSectionIndex() != 0 {
		t.Errorf("default cross-section index = %d, want 0 (arc)", f.CrossSectionIndex())
	}
	for i := range FilletCrossSectionOptions() {
		f.SetCrossSectionIndex(i)
		if f.CrossSectionIndex() != i {
			t.Errorf("set cross-section %d, got %d", i, f.CrossSectionIndex())
		}
	}
	f.SetCrossSectionIndex(99) // out of range: ignored
	if f.CrossSectionIndex() != len(FilletCrossSectionOptions())-1 {
		t.Error("out-of-range cross-section index should be ignored")
	}
	f.SetRho(0.7)
	if f.Rho() != 0.7 {
		t.Errorf("Rho = %g, want 0.7", f.Rho())
	}
}

// TestFilletToolG2CommitsValidSolid: selecting the G2 cross-section and committing rounds the edge
// into a valid solid, and the committed feature carries the G2 cross-section.
func TestFilletToolG2CommitsValidSolid(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 2)
	s.SetPicker(stubPicker{sel: verticalEdgeOf(t, block)})

	f := NewFilletTool()
	s.StartTool(f)
	s.Click(50, 50)
	f.SetRadius(0.5)
	f.SetCrossSectionIndex(1) // G2
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := f.AddedFeature().Definition().(*feature.FilletFeature).Definition()
	if def.CrossSection != feature.FilletG2 {
		t.Errorf("committed cross-section = %v, want G2", def.CrossSection)
	}
	body := activePartDef(t, s).SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("G2-filleted body not a valid solid: %+v", r)
	}
	if got := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; got >= 8 {
		t.Errorf("volume after G2 fillet = %g, want < 8 (material removed)", got)
	}
}
