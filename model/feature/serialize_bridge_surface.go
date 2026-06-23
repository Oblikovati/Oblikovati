// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// Serialized form of the NURBS bridge feature (M36-F09).

// BridgeSurfaceData is a bridge's recipe (the continuity order held to each neighbour).
type BridgeSurfaceData struct {
	OrderA int `yaml:"orderA"`
	OrderB int `yaml:"orderB"`
}

// serializeBridgeSurface captures a bridge definition.
func serializeBridgeSurface(def *BridgeDefinition) *BridgeSurfaceData {
	return &BridgeSurfaceData{OrderA: def.OrderA, OrderB: def.OrderB}
}

// restoreBridgeSurface rebuilds a bridge feature from its recipe.
func restoreBridgeSurface(fs *PartFeatures, d *BridgeSurfaceData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("bridge-surface feature is missing its payload")
	}
	return NewBridgeFeatures(fs).Add(d.OrderA, d.OrderB), nil
}
