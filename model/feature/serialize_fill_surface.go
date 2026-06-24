// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Serialized form of the NURBS boundary-fill feature (M36-F07).

// FillSurfaceData is a fill's recipe (the continuity order imposed on every side, and the number of
// bounding sides; Sides omitted/0 means the classic four-sided fill).
type FillSurfaceData struct {
	Order int `yaml:"order"`
	Sides int `yaml:"sides,omitempty"`
}

// serializeFillSurface captures a fill definition.
func serializeFillSurface(def *FillDefinition) *FillSurfaceData {
	return &FillSurfaceData{Order: def.Order, Sides: def.Sides}
}

// restoreFillSurface rebuilds a fill feature from its recipe.
func restoreFillSurface(fs *PartFeatures, d *FillSurfaceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("fill-surface feature is missing its payload")
	}
	return NewFillFeatures(fs).AddSides(d.Order, d.Sides), nil
}
