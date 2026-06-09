// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"

	"oblikovati.org/math"
)

// rectangularDims validates that the control net and weight net are non-empty,
// rectangular (every row the same length), and the same shape. It returns the
// u (row) and v (column) counts.
func rectangularDims(ctrl [][]math.Point3, weights [][]float64) (uCount, vCount int, err error) {
	uCount = len(ctrl)
	if uCount == 0 || len(ctrl[0]) == 0 {
		return 0, 0, fmt.Errorf("geom: B-spline surface control net is empty")
	}
	vCount = len(ctrl[0])
	for i, row := range ctrl {
		if len(row) != vCount {
			return 0, 0, fmt.Errorf("geom: B-spline surface control net is not rectangular: row 0 has %d, row %d has %d", vCount, i, len(row))
		}
	}
	if len(weights) != uCount || (len(weights) > 0 && len(weights[0]) != vCount) {
		return 0, 0, fmt.Errorf("geom: B-spline surface weight net %dx%d does not match control net %dx%d", len(weights), weightCols(weights), uCount, vCount)
	}
	return uCount, vCount, nil
}

// weightCols returns the column count of the weight net (0 when empty), for
// diagnostics.
func weightCols(weights [][]float64) int {
	if len(weights) == 0 {
		return 0
	}
	return len(weights[0])
}

// positiveNet rejects any non-positive weight in the net.
func positiveNet(weights [][]float64) error {
	for _, row := range weights {
		if err := requirePositiveWeights(row); err != nil {
			return err
		}
	}
	return nil
}

// copyNet deep-copies a control net so the constructed surface is immutable.
func copyNet(ctrl [][]math.Point3) [][]math.Point3 {
	out := make([][]math.Point3, len(ctrl))
	for i, row := range ctrl {
		out[i] = append([]math.Point3(nil), row...)
	}
	return out
}

// copyWeights deep-copies a weight net.
func copyWeights(weights [][]float64) [][]float64 {
	out := make([][]float64, len(weights))
	for i, row := range weights {
		out[i] = append([]float64(nil), row...)
	}
	return out
}
