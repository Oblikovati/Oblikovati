// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/math"
)

// Serialized forms of the sheet-metal Unfold/Refold features (M13-F04): the list of bend
// transforms each acts on. Points and directions persist as [x,y,z] triples; the angle is in
// radians. The transforms are self-contained (baked from placements at creation), so the
// round-trip needs no reference to the part's other features.

// bendTransformData is one persisted bend transform.
type bendTransformData struct {
	LinePoint []float64 `yaml:"linePoint"`
	LineDir   []float64 `yaml:"lineDir"`
	Up        []float64 `yaml:"up"`
	Out       []float64 `yaml:"out"`
	Angle     float64   `yaml:"angle"`
	Radius    float64   `yaml:"radius"`
	Thickness float64   `yaml:"thickness"`
	Neutral   float64   `yaml:"neutral"`
}

// SheetMetalUnfoldData / SheetMetalRefoldData persist the bend list of an unfold / refold.
type SheetMetalUnfoldData struct {
	Bends []bendTransformData `yaml:"bends"`
}
type SheetMetalRefoldData struct {
	Bends []bendTransformData `yaml:"bends"`
}

// serializeBendTransforms projects bend transforms to their persisted form.
func serializeBendTransforms(bends []BendTransform) []bendTransformData {
	out := make([]bendTransformData, len(bends))
	for i, b := range bends {
		out[i] = bendTransformData{
			LinePoint: []float64{b.LinePoint.X, b.LinePoint.Y, b.LinePoint.Z},
			LineDir:   []float64{b.LineDir.X, b.LineDir.Y, b.LineDir.Z},
			Up:        []float64{b.Up.X, b.Up.Y, b.Up.Z},
			Out:       []float64{b.Out.X, b.Out.Y, b.Out.Z},
			Angle:     b.Angle,
			Radius:    b.Radius,
			Thickness: b.Thickness,
			Neutral:   b.Neutral,
		}
	}
	return out
}

// restoreBendTransforms rebuilds bend transforms, erroring on a malformed triple.
func restoreBendTransforms(data []bendTransformData) ([]BendTransform, error) {
	out := make([]BendTransform, len(data))
	for i, d := range data {
		if len(d.LinePoint) != 3 || len(d.LineDir) != 3 || len(d.Up) != 3 || len(d.Out) != 3 {
			return nil, fmt.Errorf("sheet-metal unfold: bend %d has a malformed [x,y,z] triple", i)
		}
		out[i] = BendTransform{
			LinePoint: math.P3(d.LinePoint[0], d.LinePoint[1], d.LinePoint[2]),
			LineDir:   math.V3(d.LineDir[0], d.LineDir[1], d.LineDir[2]),
			Up:        math.V3(d.Up[0], d.Up[1], d.Up[2]),
			Out:       math.V3(d.Out[0], d.Out[1], d.Out[2]),
			Angle:     d.Angle,
			Radius:    d.Radius,
			Thickness: d.Thickness,
			Neutral:   d.Neutral,
		}
	}
	return out, nil
}

// restoreSheetMetalUnfold / restoreSheetMetalRefold rebuild the features from their payload.
func restoreSheetMetalUnfold(fs *PartFeatures, d *SheetMetalUnfoldData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal unfold feature is missing its payload")
	}
	bends, err := restoreBendTransforms(d.Bends)
	if err != nil {
		return nil, err
	}
	return NewSheetMetalUnfoldFeatures(fs).Add(&SheetMetalUnfoldDefinition{Bends: bends}), nil
}

func restoreSheetMetalRefold(fs *PartFeatures, d *SheetMetalRefoldData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("sheet-metal refold feature is missing its payload")
	}
	bends, err := restoreBendTransforms(d.Bends)
	if err != nil {
		return nil, err
	}
	return NewSheetMetalRefoldFeatures(fs).Add(&SheetMetalRefoldDefinition{Bends: bends}), nil
}
