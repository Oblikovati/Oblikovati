// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati/math"
)

// MoveData is the serialized form of a MoveFeature: the target body index and the
// transform as its 16 row-major cells (Inventor's GetMatrixData form). The cells are
// the general affine form so any rigid move / mirror / uniform scale round-trips.
type MoveData struct {
	Body   int       `yaml:"body"`
	Matrix []float64 `yaml:"matrix"`
}

// serializeMove projects a move recipe to its persisted form.
func serializeMove(def *MoveDefinition) *MoveData {
	cells := def.Transform.Cells()
	return &MoveData{Body: def.BodyIndex, Matrix: cells[:]}
}

// restoreMove rebuilds a MoveFeature, erroring on a missing payload or a matrix that
// is not 16 cells (no silent loss of the transform).
func restoreMove(fs *PartFeatures, d *MoveData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("move feature is missing its payload")
	}
	if len(d.Matrix) != 16 {
		return nil, fmt.Errorf("move feature matrix has %d cells, want 16", len(d.Matrix))
	}
	var cells [16]float64
	copy(cells[:], d.Matrix)
	m := NewModifyFeatures(fs)
	return m.AddMove(d.Body, math.Matrix4FromCells(cells)), nil
}
