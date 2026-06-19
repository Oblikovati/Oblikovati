// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// draftBody tapers the picked faces of the running body about the pull direction by angle
// (via ops.DraftFaces) and replaces it; a lost face key (surfaced by the op) makes the
// feature go Sick. See kernel/ops/draft.go for the geometry.
func draftBody(in Input, faceKeys [][]byte, pull math.Vector3, angle float64, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	result, err := ops.DraftFaces(body, faceKeys, pull, angle)
	if err != nil {
		return Output{}, fmt.Errorf(errCtxWrap, feat, err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}
