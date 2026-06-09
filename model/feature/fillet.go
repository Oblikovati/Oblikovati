// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati/kernel/ops"
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
	// A curved body (analytic cylinder) is re-faceted and the selected edges remapped to its faceted
	// segments, so the rolling-ball blend works instead of failing on a degenerate closed edge
	// (#129/#127). A planar body is unchanged (work==body, same keys).
	work, keys := body, edgeKeys
	if origEdges, e := resolveEdges(body, edgeKeys); e == nil {
		pb, mapped := planarizeForEdges(body, origEdges, feat)
		if pb != body {
			work = pb
			keys = make([][]byte, len(mapped))
			for i, me := range mapped {
				keys[i] = me.ReferenceKey()
			}
		}
	}
	result, err := ops.FilletEdges(work, keys, radius)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}
