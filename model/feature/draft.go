// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/math"
)

// draftBody tapers the picked faces of the running body about the pull direction by angle (via
// blend.DraftFacesNeutral) and replaces it; when neutral is non-nil each face pivots on the line where
// it meets that parting plane (#1801), else on the implicit lowest-vertex hinge. A lost face key
// (surfaced by the op) makes the feature go Sick. See kernel/ops/draft.go for the geometry.
func draftBody(in Input, faceKeys [][]byte, pull math.Vector3, neutral *geom.Plane, angle float64, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	result, err := blend.DraftFacesNeutral(body, faceKeys, pull, neutral, angle)
	if err != nil {
		return Output{}, fmt.Errorf("%s: %w", feat, err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}
