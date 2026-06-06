// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati/kernel/ops"
)

// shellBody hollows the running body to a wall thickness, opening the removed faces, via
// ops.Shell, and replaces it in the body list. A lost face key or non-positive thickness
// is an error so the feature goes Sick. See kernel/ops/shell.go for the geometry.
func shellBody(in Input, removedFaceKeys [][]byte, thickness float64, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if thickness <= 0 {
		return Output{}, fmt.Errorf("%s: thickness %g must be > 0", feat, thickness)
	}
	result, err := ops.Shell(body, removedFaceKeys, thickness)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}
