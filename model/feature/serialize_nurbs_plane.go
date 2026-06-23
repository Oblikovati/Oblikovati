// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Serialized form of the NURBS plane primitive (M36-F03): just its size and control-point count
// — the flat patch is regenerated from the recipe on load.

// NurbsPlaneData is a NURBS plane patch's recipe.
type NurbsPlaneData struct {
	Width  float64 `yaml:"width"`
	Height float64 `yaml:"height"`
	UCount int     `yaml:"uCount"`
	VCount int     `yaml:"vCount"`
}

// serializeNurbsPlane captures a plane-patch definition as its persisted recipe.
func serializeNurbsPlane(def *NurbsPlaneDefinition) *NurbsPlaneData {
	return &NurbsPlaneData{Width: def.Width, Height: def.Height, UCount: def.UCount, VCount: def.VCount}
}

// restoreNurbsPlane rebuilds a NURBS plane feature from its recipe.
func restoreNurbsPlane(fs *PartFeatures, d *NurbsPlaneData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("nurbs plane feature is missing its payload")
	}
	return NewNurbsPlaneFeatures(fs).Add(d.Width, d.Height, d.UCount, d.VCount), nil
}
