// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// Unfold/Refold features (M13-F04, #377). These flatten and re-fold the running body in the
// feature stream so a cut placed while flat develops correctly on the folded part. The bends
// to act on are baked from the part's recorded bend placements at creation time, so the
// feature stays self-contained at recompute (it never reaches back into the other features).

// AddUnfold appends an unfold feature that flattens every edge bend of the part, leaving the
// part flat for subsequent features. It errors when the part is not sheet metal or has no
// bends to flatten.
func (d *PartComponentDefinition) AddUnfold() (*feature.PartFeature, error) {
	bends, err := d.bendTransforms("unfold")
	if err != nil {
		return nil, err
	}
	return feature.NewSheetMetalUnfoldFeatures(d.features).Add(&feature.SheetMetalUnfoldDefinition{Bends: bends}), nil
}

// AddRefold appends a refold feature that re-folds the same bends an earlier unfold flattened,
// restoring the folded part (and carrying any edits made while flat).
func (d *PartComponentDefinition) AddRefold() (*feature.PartFeature, error) {
	bends, err := d.bendTransforms("refold")
	if err != nil {
		return nil, err
	}
	return feature.NewSheetMetalRefoldFeatures(d.features).Add(&feature.SheetMetalRefoldDefinition{Bends: bends}), nil
}

// bendTransforms bakes one [feature.BendTransform] per recorded edge bend: the bend line, the
// base sheet normal (which fixes the split plane), and the fold angle.
func (d *PartComponentDefinition) bendTransforms(op string) ([]feature.BendTransform, error) {
	if d.sheetMetal == nil {
		return nil, fmt.Errorf("%s: the active part is not a sheet-metal part", op)
	}
	base, ok := d.baseFace()
	if !ok {
		return nil, fmt.Errorf("%s: the part has no base Face", op)
	}
	normal := base.Definition().Sketch.Plane().Normal().AsVector()
	var out []feature.BendTransform
	fs := d.features
	for i := 0; i < fs.Count(); i++ {
		if bt, ok := bakeBendTransform(fs.Item(i), normal); ok {
			out = append(out, bt)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: the part has no bends to act on", op)
	}
	return out, nil
}

// bakeBendTransform turns one feature's recorded bend placement into a transform, or ok=false
// when it is not a healthy placed edge bend.
func bakeBendTransform(pf *feature.PartFeature, baseNormal math.Vector3) (feature.BendTransform, bool) {
	if pf.Suppressed() || !pf.Health().OK() {
		return feature.BendTransform{}, false
	}
	placed, ok := pf.Definition().(feature.PlacedBend)
	if !ok {
		return feature.BendTransform{}, false
	}
	p, ok := placed.Placement()
	if !ok {
		return feature.BendTransform{}, false
	}
	return feature.BendTransform{
		LinePoint:  p.AxisStart,
		LineDir:    p.AxisStart.VectorTo(p.AxisEnd),
		BaseNormal: baseNormal,
		Angle:      p.Angle,
	}, true
}
