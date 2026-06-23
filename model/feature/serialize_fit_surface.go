// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
)

// Serialized form of the fit-surface feature (M36-F15): the region points as a flat [x,y,z] list and
// the degree / control-span targets.

// FitSurfaceData is a fitted surface's recipe.
type FitSurfaceData struct {
	Points [][]float64 `yaml:"points"`
	Degree int         `yaml:"degree"`
	NU     int         `yaml:"nu"`
	NV     int         `yaml:"nv"`
}

// serializeFitSurface captures a fit definition.
func serializeFitSurface(def *FitDefinition) *FitSurfaceData {
	return &FitSurfaceData{Points: encodePoints(def.Points), Degree: def.Degree, NU: def.NU, NV: def.NV}
}

// restoreFitSurface rebuilds a fit feature from its recipe.
func restoreFitSurface(fs *PartFeatures, d *FitSurfaceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("fit-surface feature is missing its payload")
	}
	return NewFitFeatures(fs).Add(decodePoints(d.Points), d.Degree, d.NU, d.NV), nil
}
