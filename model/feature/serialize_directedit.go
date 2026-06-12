// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
)

// YAML codec for the consolidated direct edit (#332). The operation is stored by its
// frozen wire spelling; only the fields its operation uses are emitted.

// DirectEditData is a direct edit's recipe.
type DirectEditData struct {
	Operation   string    `yaml:"operation"`
	Faces       []string  `yaml:"faces,omitempty"`
	Translation []float64 `yaml:"translation,omitempty"`
	Direction   []float64 `yaml:"direction,omitempty"`
	Distance    float64   `yaml:"distance,omitempty"`
	AxisPoint   []float64 `yaml:"axisPoint,omitempty"`
	AxisDir     []float64 `yaml:"axisDir,omitempty"`
	Angle       float64   `yaml:"angle,omitempty"`
	Scale       float64   `yaml:"scale,omitempty"`
	Base        []float64 `yaml:"base,omitempty"`
}

func serializeDirectEdit(def *DirectEditDefinition) *DirectEditData {
	return &DirectEditData{
		Operation:   def.Operation.String(),
		Faces:       encodeKeys(def.FaceKeys),
		Translation: encodeVec3(def.Translation),
		Direction:   encodeVec3(def.Direction),
		Distance:    evalFloat(def.Distance),
		AxisPoint:   encodePoint3(def.AxisPoint),
		AxisDir:     encodeVec3(def.AxisDir),
		Angle:       evalFloat(def.Angle),
		Scale:       evalFloat(def.ScaleFactor),
		Base:        encodePoint3(def.BasePoint),
	}
}

func restoreDirectEdit(fs *PartFeatures, d *DirectEditData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("directEdit feature is missing its payload")
	}
	op, ok := types.ParseDirectEditOperationType(d.Operation)
	if !ok {
		return nil, fmt.Errorf("directEdit: unknown operation %q (want move/size/rotate/delete/scale)", d.Operation)
	}
	keys, err := decodeKeys(d.Faces)
	if err != nil {
		return nil, err
	}
	return NewModifyFeatures(fs).AddDirectEdit(&DirectEditDefinition{
		Operation: op, FaceKeys: keys,
		Translation: decodeVec3(d.Translation),
		Direction:   decodeVec3(d.Direction), Distance: constFloat(d.Distance),
		AxisPoint: decodePoint3(d.AxisPoint), AxisDir: decodeVec3(d.AxisDir), Angle: constFloat(d.Angle),
		ScaleFactor: constFloat(d.Scale), BasePoint: decodePoint3(d.Base),
	}), nil
}
