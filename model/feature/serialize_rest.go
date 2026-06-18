// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// RestData persists a rest pad (#486): the sketch (by index), the profile indices, the depth, the
// raised/recessed flag and the draft taper.
type RestData struct {
	Sketch   int     `yaml:"sketch"`
	Profiles []int   `yaml:"profiles,omitempty"`
	Depth    float64 `yaml:"depth"`
	Recessed bool    `yaml:"recessed,omitempty"`
	Taper    float64 `yaml:"taper,omitempty"`
}

func serializeRest(def *RestDefinition, sk SketchIndexer) (*RestData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("rest references a sketch that is not in the part")
	}
	return &RestData{
		Sketch:   idx,
		Profiles: append([]int(nil), def.ProfileIndices...),
		Depth:    evalFloat(def.Depth),
		Recessed: def.Recessed,
		Taper:    def.Taper,
	}, nil
}

func restoreRest(fs *PartFeatures, d *RestData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("rest feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("rest references sketch index %d, which does not exist", d.Sketch)
	}
	return NewPlasticFeatures(fs).AddRest(skt, d.Profiles, constFloat(d.Depth), d.Recessed, d.Taper), nil
}
