// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// SimplifyData is the serialized form of a SimplifyFeature (M20-F13): the faces removed and
// whether internal voids were filled.
type SimplifyData struct {
	RemoveFaces []string `yaml:"removeFaces,omitempty"`
	FillVoids   bool     `yaml:"fillVoids,omitempty"`
}

// UnwrapData is the serialized form of an UnwrapFeature: the cylindrical face flattened.
type UnwrapData struct {
	Face string `yaml:"face"`
}

// restoreSimplify rebuilds a SimplifyFeature.
func restoreSimplify(fs *PartFeatures, d *SimplifyData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("simplify feature is missing its payload")
	}
	keys, err := decodeKeys(d.RemoveFaces)
	if err != nil {
		return nil, err
	}
	return NewModifyFeatures(fs).AddSimplify(keys, d.FillVoids), nil
}

// restoreUnwrap rebuilds an UnwrapFeature.
func restoreUnwrap(fs *PartFeatures, d *UnwrapData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("unwrap feature is missing its payload")
	}
	key, err := decodeKey(d.Face)
	if err != nil {
		return nil, err
	}
	return NewModifyFeatures(fs).AddUnwrap(key), nil
}
