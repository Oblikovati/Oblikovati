// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"oblikovati.org/model/feature"
	"oblikovati.org/model/sheetmetal"
)

// Unfold/Refold features (M13-F04, #377). These flatten and re-fold the running body in the
// feature stream so a cut placed while flat develops correctly on the folded part. The bends
// to act on are baked from the part's recorded bend placements at creation time, so the
// feature stays self-contained at recompute (it never reaches back into the other features).

// AddUnfold appends an unfold feature that flattens every edge bend of the part, leaving the
// part flat for subsequent features. It errors when the part is not sheet metal or has no
// bends to flatten.
func (d *PartComponentDefinition) AddUnfold() (*feature.PartFeature, error) {
	return d.AddUnfoldInto(d.features)
}

// AddUnfoldInto builds the unfold feature into engine fs — Commit builds into the part's own
// features (AddUnfold) while the tool preview builds into a scratch engine, so both run the
// same construction and the commit gate can inspect the exact feature OK creates (#1626).
func (d *PartComponentDefinition) AddUnfoldInto(fs *feature.PartFeatures) (*feature.PartFeature, error) {
	bends, err := d.bendTransforms("unfold")
	if err != nil {
		return nil, err
	}
	return feature.NewSheetMetalUnfoldFeatures(fs).Add(&feature.SheetMetalUnfoldDefinition{Bends: bends}), nil
}

// AddRefold appends a refold feature that re-folds the same bends an earlier unfold flattened,
// restoring the folded part (and carrying any edits made while flat).
func (d *PartComponentDefinition) AddRefold() (*feature.PartFeature, error) {
	return d.AddRefoldInto(d.features)
}

// AddRefoldInto builds the refold feature into engine fs — the scratch-engine seam the tool
// preview shares with AddRefold, mirroring AddUnfoldInto (#1626).
func (d *PartComponentDefinition) AddRefoldInto(fs *feature.PartFeatures) (*feature.PartFeature, error) {
	bends, err := d.bendTransforms("refold")
	if err != nil {
		return nil, err
	}
	return feature.NewSheetMetalRefoldFeatures(fs).Add(&feature.SheetMetalRefoldDefinition{Bends: bends}), nil
}

// bendTransforms bakes one [feature.BendTransform] per recorded edge bend: the bend line, its
// fold frame, and the rule-developed neutral-fibre radius (the developed length per radian), so
// the unfold/refold map unrolls each bend at the correct length for the active unfold method.
func (d *PartComponentDefinition) bendTransforms(op string) ([]feature.BendTransform, error) {
	if d.sheetMetal == nil {
		return nil, fmt.Errorf("%s: the active part is not a sheet-metal part", op)
	}
	if _, ok := d.baseFace(); !ok {
		return nil, fmt.Errorf("%s: the part has no base Face", op)
	}
	var out []feature.BendTransform
	fs := d.features
	for i := 0; i < fs.Count(); i++ {
		out = append(out, bakeBendTransforms(fs.Item(i), d.sheetMetal)...)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: the part has no bends to act on", op)
	}
	return out, nil
}

// bakeBendTransforms turns a feature's recorded bend placements into transforms — one per bend, so a
// multi-edge flange bakes every wall (#2071); empty when it is not a healthy placed edge bend. The
// neutral-fibre radius is the rule's developed length for each bend divided by its angle (so it
// honours the active unfold method — K-factor, bend table, or equation — not just a fixed K-factor).
func bakeBendTransforms(pf *feature.PartFeature, rule *sheetmetal.Rule) []feature.BendTransform {
	if pf.Suppressed() || !pf.Health().OK() {
		return nil
	}
	var out []feature.BendTransform
	for _, p := range featurePlacements(pf.Definition()) {
		if p.Angle <= 0 {
			continue
		}
		out = append(out, feature.BendTransform{
			LinePoint: p.AxisStart,
			LineDir:   p.AxisStart.VectorTo(p.AxisEnd),
			Up:        p.Up.AsVector(),
			Out:       p.Outward.AsVector(),
			Angle:     p.Angle,
			Radius:    p.Radius,
			Thickness: p.Thickness,
			Neutral:   rule.BendAllowance(p.Angle, p.Radius) / p.Angle,
		})
	}
	return out
}
