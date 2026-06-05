// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// EmbossData is an emboss's recipe: its closed profile(s) (a sketch + region indices), the
// depth, whether it engraves (cut) instead of raises (join), and a draft taper.
type EmbossData struct {
	Sketch   int     `yaml:"sketch"`
	Profiles []int   `yaml:"profiles"`
	Depth    float64 `yaml:"depth"`
	Engrave  bool    `yaml:"engrave,omitempty"`
	Taper    float64 `yaml:"taper,omitempty"`
}

func serializeEmboss(def *EmbossDefinition, sk SketchIndexer) (*EmbossData, error) {
	idx, ok := sk.IndexOf(def.Sketch)
	if !ok {
		return nil, fmt.Errorf("emboss references a sketch that is not in the part")
	}
	return &EmbossData{
		Sketch: idx, Profiles: append([]int(nil), def.ProfileIndices...),
		Depth: evalFloat(def.Depth), Engrave: def.Engrave, Taper: def.Taper,
	}, nil
}

func restoreEmboss(fs *PartFeatures, d *EmbossData, sk SketchIndexer) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("emboss feature is missing its payload")
	}
	skt, ok := sk.At(d.Sketch)
	if !ok {
		return nil, fmt.Errorf("emboss references sketch index %d, which does not exist", d.Sketch)
	}
	return NewEmbossFeatures(fs).Add(skt, d.Profiles, constFloat(d.Depth), d.Engrave, d.Taper), nil
}
