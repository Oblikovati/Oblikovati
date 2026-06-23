// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/geom"
)

// Serialized forms of the NURBS extend and untrim features (M36-F11).

// ExtendSurfaceData is a NURBS surface extend's recipe.
type ExtendSurfaceData struct {
	Edge     int     `yaml:"edge"`
	Distance float64 `yaml:"distance"`
	Order    int     `yaml:"order"`
}

// UntrimData is an untrim's recipe (no parameters — it recovers the running face's full domain).
type UntrimData struct{}

// serializeExtendSurface captures an extend definition.
func serializeExtendSurface(def *ExtendSurfaceDefinition) *ExtendSurfaceData {
	return &ExtendSurfaceData{Edge: int(def.Edge), Distance: def.Distance, Order: def.Order}
}

// restoreExtendSurface rebuilds an extend feature from its recipe.
func restoreExtendSurface(fs *PartFeatures, d *ExtendSurfaceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("extend-surface feature is missing its payload")
	}
	return NewExtendSurfaceFeatures(fs).Add(geom.Boundary(d.Edge), d.Distance, d.Order), nil
}

// restoreUntrim rebuilds an untrim feature from its recipe.
func restoreUntrim(fs *PartFeatures, d *UntrimData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("untrim-surface feature is missing its payload")
	}
	return NewUntrimFeatures(fs).Add(), nil
}
