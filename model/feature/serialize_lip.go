// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// LipData is the serialized form of a LipFeature (M20-F10): the edge path the bead runs
// along, its cross-section size, and whether it is a recessed groove.
type LipData struct {
	Edges  []string `yaml:"edges"`
	Width  float64  `yaml:"width"`
	Height float64  `yaml:"height"`
	Groove bool     `yaml:"groove,omitempty"`
}

// restoreLip rebuilds a LipFeature, erroring on a missing payload.
func restoreLip(fs *PartFeatures, d *LipData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("lip feature is missing its payload")
	}
	keys, err := decodeKeys(d.Edges)
	if err != nil {
		return nil, err
	}
	return NewDressUpFeatures(fs).AddLip(keys, constFloat(d.Width), constFloat(d.Height), d.Groove), nil
}
