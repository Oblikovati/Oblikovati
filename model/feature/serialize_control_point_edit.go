// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Serialized form of a control-net edit (M36-F03): the list of per-control-point displacements.
// The edit targets the running surface body's NURBS face, so nothing but the displacements needs
// persisting.

// ControlPointEditData is one control-net edit's recipe: its per-CV displacements.
type ControlPointEditData struct {
	Deltas []ControlPointDeltaData `yaml:"deltas"`
}

// ControlPointDeltaData is a single control point's grid index and displacement.
type ControlPointDeltaData struct {
	U  int     `yaml:"u"`
	V  int     `yaml:"v"`
	DX float64 `yaml:"dx"`
	DY float64 `yaml:"dy"`
	DZ float64 `yaml:"dz"`
}

// serializeControlPointEdit captures a control-net edit as its persisted recipe.
func serializeControlPointEdit(def *ControlPointEditDefinition) *ControlPointEditData {
	out := &ControlPointEditData{Deltas: make([]ControlPointDeltaData, len(def.Deltas))}
	for i, d := range def.Deltas {
		out.Deltas[i] = ControlPointDeltaData{U: d.U, V: d.V, DX: float64(d.Delta.X), DY: float64(d.Delta.Y), DZ: float64(d.Delta.Z)}
	}
	return out
}

// restoreControlPointEdit rebuilds a control-net edit feature from its recipe.
func restoreControlPointEdit(fs *PartFeatures, d *ControlPointEditData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("control-point edit feature is missing its payload")
	}
	deltas := make([]geom.ControlPointDelta, len(d.Deltas))
	for i, cd := range d.Deltas {
		deltas[i] = geom.ControlPointDelta{U: cd.U, V: cd.V, Delta: math.V3(math.Scalar(cd.DX), math.Scalar(cd.DY), math.Scalar(cd.DZ))}
	}
	return NewControlPointEditFeatures(fs).Add(deltas), nil
}
