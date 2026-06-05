// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// RibData is a rib's recipe: its open profile (a sketch + path index), wall thickness, signed
// depth along the sketch-plane normal, and the boolean operation joining it to the part.
type RibData struct {
	Sketch    int     `yaml:"sketch"`
	Profile   int     `yaml:"profile"`
	Thickness float64 `yaml:"thickness"`
	Depth     float64 `yaml:"depth"`
	Operation string  `yaml:"operation"`
}

func serializeRib(def *RibDefinition, sk SketchIndexer) (*RibData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("rib references a sketch that is not in the part")
	}
	op, err := operationName(def.Operation)
	if err != nil {
		return nil, err
	}
	return &RibData{
		Sketch: idx, Profile: def.ProfileIndex,
		Thickness: evalFloat(def.Thickness), Depth: evalFloat(def.Depth), Operation: op,
	}, nil
}

func restoreRib(fs *PartFeatures, d *RibData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("rib feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("rib references sketch index %d, which does not exist", d.Sketch)
	}
	op, err := parseOperation(d.Operation)
	if err != nil {
		return nil, err
	}
	return NewRibFeatures(fs).Add(skt, d.Profile, constFloat(d.Thickness), constFloat(d.Depth), op), nil
}
