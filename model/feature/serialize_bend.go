// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
)

// BendData is the serialized form of a BendPartFeature: the bend line (sketch index + line
// index), the bend type, the three driving scalars (frozen to their current value), and the
// fold-direction flip. Scalars not used by the bend type round-trip as 0.
type BendData struct {
	Sketch    int     `yaml:"sketch"`
	Line      int     `yaml:"line"`
	BendType  int32   `yaml:"bendType"`
	Radius    float64 `yaml:"radius,omitempty"`
	Angle     float64 `yaml:"angle,omitempty"`
	ArcLength float64 `yaml:"arcLength,omitempty"`
	Flip      bool    `yaml:"flip,omitempty"`
}

// serializeBend projects a bend recipe to its persisted form, erroring if its sketch is not
// in the part (so the bend line cannot be lost silently).
func serializeBend(def *BendPartDefinition, sk SketchIndexer) (*BendData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("bend-part references a sketch that is not in the part")
	}
	return &BendData{
		Sketch: idx, Line: def.LineIndex, BendType: int32(def.BendType),
		Radius: evalFloat(def.Radius), Angle: evalFloat(def.Angle), ArcLength: evalFloat(def.ArcLength),
		Flip: def.Flip,
	}, nil
}

// restoreBend rebuilds a BendPartFeature, erroring on a missing payload or unknown sketch.
func restoreBend(fs *PartFeatures, d *BendData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("bend-part feature is missing its payload")
	}
	s, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("bend-part references sketch %d which is not in the part", d.Sketch)
	}
	def := &BendPartDefinition{
		Sketch: s, LineIndex: d.Line, BendType: types.BendPartType(d.BendType),
		Radius: constFloat(d.Radius), Angle: constFloat(d.Angle), ArcLength: constFloat(d.ArcLength),
		Flip: d.Flip,
	}
	return NewBendPartFeatures(fs).Add(def), nil
}
