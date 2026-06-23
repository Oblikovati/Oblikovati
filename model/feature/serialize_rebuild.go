// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Serialized form of the Rebuild feature (M36-F02). The recipe is just the target degree and
// control-point count per direction; the achieved deviation is a recompute result, not part of
// the persisted recipe, so it is not stored (it is recomputed on load).

// RebuildData is a surface rebuild's recipe: the per-direction target degree and CV count.
type RebuildData struct {
	UDegree int `yaml:"uDegree"`
	VDegree int `yaml:"vDegree"`
	UCount  int `yaml:"uCount"`
	VCount  int `yaml:"vCount"`
}

// serializeRebuild captures a rebuild definition as its persisted recipe.
func serializeRebuild(def *RebuildDefinition) *RebuildData {
	return &RebuildData{UDegree: def.UDegree, VDegree: def.VDegree, UCount: def.UCount, VCount: def.VCount}
}

// restoreRebuild rebuilds a Rebuild feature from its recipe.
func restoreRebuild(fs *PartFeatures, d *RebuildData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("rebuild feature is missing its payload")
	}
	return NewRebuildFeatures(fs).Add(d.UDegree, d.VDegree, d.UCount, d.VCount), nil
}
