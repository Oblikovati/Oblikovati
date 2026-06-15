// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// GrillData is the serialized form of a GrillFeature (#863): the sketch (by index) and the
// boundary vent-profile indices (whose inner loops are the bridging ribs/spars/islands).
type GrillData struct {
	Sketch     int     `yaml:"sketch"`
	Boundaries []int   `yaml:"boundaries"`
	Draft      float64 `yaml:"draft,omitempty"`
}

// serializeGrill captures a grill, recording its sketch by index.
func serializeGrill(def *GrillDefinition, sk SketchIndexer) (*GrillData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("grill: sketch is not indexed by the part")
	}
	return &GrillData{Sketch: idx, Boundaries: append([]int(nil), def.Boundaries...), Draft: def.Draft}, nil
}

// restoreGrill rebuilds a GrillFeature, re-binding its sketch by index.
func restoreGrill(fs *PartFeatures, d *GrillData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("grill feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("grill: sketch %d not found", d.Sketch)
	}
	return NewGrillFeatures(fs).Add(&GrillDefinition{
		Sketch:     skt,
		Boundaries: append([]int(nil), d.Boundaries...),
		Draft:      d.Draft,
	}), nil
}
