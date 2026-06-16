// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sheetmetal"
)

// Bends projects the part's bend lineage (M13-F04, #377): every bend the wall/bend features
// introduced, in creation order, developed by the active rule's unfold method. It is the
// flat pattern's prerequisite — the architecture requires every bend to record its unfold
// parameters, and here the feature history IS the bend graph, so no faceted-geometry
// detection is needed. Returns nil when the part is not in the sheet-metal environment.
//
// Only healthy, unsuppressed features contribute: a sick or suppressed wall added no bend
// to the current folded body, so it adds none to the flat development either.
func (d *PartComponentDefinition) Bends() []sheetmetal.Bend {
	if d.sheetMetal == nil {
		return nil
	}
	thickness := d.sheetMetal.Thickness()
	unfold := d.sheetMetal.Unfold()
	fs := d.features
	var bends []sheetmetal.Bend
	for i := 0; i < fs.Count(); i++ {
		pf := fs.Item(i)
		if pf.Suppressed() || !pf.Health().OK() {
			continue
		}
		lineage, ok := pf.Definition().(feature.BendLineage)
		if !ok {
			continue
		}
		bends = append(bends, d.developBends(pf.Name(), lineage, thickness, unfold)...)
	}
	return bends
}

// developBends turns one feature's raw bend specs into developed bend records, filling a
// missing radius override with the rule's default bend radius.
func (d *PartComponentDefinition) developBends(name string, lineage feature.BendLineage, thickness float64, unfold sheetmetal.UnfoldMethod) []sheetmetal.Bend {
	out := make([]sheetmetal.Bend, 0, 1)
	for _, spec := range lineage.BendSpecs(thickness) {
		radius := spec.Radius
		if radius <= 0 {
			radius = d.sheetMetal.BendRadius()
		}
		out = append(out, sheetmetal.NewBend(name, spec.Angle, radius, thickness, unfold))
	}
	return out
}

// TotalBendAllowance sums the developed allowance over all bends — the total length the flat
// pattern adds for bending (the developed model is longer than the folded outside extents by
// this minus the deductions).
func (d *PartComponentDefinition) TotalBendAllowance() float64 {
	var total float64
	for _, b := range d.Bends() {
		total += b.Allowance
	}
	return total
}
