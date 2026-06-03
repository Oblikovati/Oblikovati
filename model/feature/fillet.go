// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/ops"
)

// filletBody rounds the selected convex edges of the running body to the given radius via
// ops.FilletEdges (a real rolling-ball blend with cylinder faces), replacing it in the body
// list. A lost edge, a non-convex edge, or a non-positive radius is an error so the feature
// goes Sick. See kernel/ops/fillet.go for the geometry.
func filletBody(in Input, edgeKeys [][]byte, radius float64, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	if radius <= 0 {
		return Output{}, fmt.Errorf("%s: radius %g must be > 0", feat, radius)
	}
	result, err := ops.FilletEdges(body, edgeKeys, radius)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}
