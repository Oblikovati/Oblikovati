// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
)

// ModelToleranceData is the serialized form of a ModelToleranceFeature (#866): the GD&T
// feature-control frames and datum labels, with geometry references as encoded keys and the
// characteristic as its wire spelling.
type ModelToleranceData struct {
	Frames []ToleranceFrameData `yaml:"frames,omitempty"`
	Datums []DatumLabelData     `yaml:"datums,omitempty"`
}

// ToleranceFrameData is one serialized feature-control frame.
type ToleranceFrameData struct {
	Geometry       string   `yaml:"geometry"`
	Characteristic string   `yaml:"characteristic"`
	Value          float64  `yaml:"value"`
	Datums         []string `yaml:"datums,omitempty"`
}

// DatumLabelData is one serialized datum label.
type DatumLabelData struct {
	Geometry string `yaml:"geometry"`
	Label    string `yaml:"label"`
}

// serializeModelTolerance captures the GD&T record set.
func serializeModelTolerance(def *ModelToleranceDefinition) *ModelToleranceData {
	d := &ModelToleranceData{}
	for _, f := range def.Frames {
		d.Frames = append(d.Frames, ToleranceFrameData{
			Geometry:       encodeKey(f.GeometryKey),
			Characteristic: f.Characteristic.String(),
			Value:          f.Value,
			Datums:         append([]string(nil), f.Datums...),
		})
	}
	for _, dl := range def.Datums {
		d.Datums = append(d.Datums, DatumLabelData{Geometry: encodeKey(dl.GeometryKey), Label: dl.Label})
	}
	return d
}

// restoreModelTolerance rebuilds a ModelToleranceFeature from its records, erroring on an
// unknown characteristic spelling (no silent loss).
func restoreModelTolerance(fs *PartFeatures, d *ModelToleranceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("modelTolerance feature is missing its payload")
	}
	def := &ModelToleranceDefinition{}
	for _, fr := range d.Frames {
		key, err := decodeKey(fr.Geometry)
		if err != nil {
			return nil, err
		}
		ch, ok := types.ParseGeometricCharacteristic(fr.Characteristic)
		if !ok {
			return nil, fmt.Errorf("modelTolerance: unknown characteristic %q", fr.Characteristic)
		}
		def.Frames = append(def.Frames, ToleranceFrame{GeometryKey: key, Characteristic: ch, Value: fr.Value, Datums: append([]string(nil), fr.Datums...)})
	}
	for _, dl := range d.Datums {
		key, err := decodeKey(dl.Geometry)
		if err != nil {
			return nil, err
		}
		def.Datums = append(def.Datums, DatumLabel{GeometryKey: key, Label: dl.Label})
	}
	return NewToleranceFeatures(fs).AddModelTolerance(def), nil
}
