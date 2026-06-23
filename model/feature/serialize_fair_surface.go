// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Serialized form of the fairing feature (M36-F04).

// FairSurfaceData is a fairing's recipe.
type FairSurfaceData struct {
	HoldOrder  int     `yaml:"holdOrder"`
	Strength   float64 `yaml:"strength"`
	Iterations int     `yaml:"iterations"`
}

func serializeFairSurface(def *FairDefinition) *FairSurfaceData {
	return &FairSurfaceData{HoldOrder: def.HoldOrder, Strength: def.Strength, Iterations: def.Iterations}
}

func restoreFairSurface(fs *PartFeatures, d *FairSurfaceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("fair-surface feature is missing its payload")
	}
	return NewFairFeatures(fs).Add(d.HoldOrder, d.Strength, d.Iterations), nil
}
