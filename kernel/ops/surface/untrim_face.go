// SPDX-License-Identifier: GPL-2.0-only

package surface

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
)

// Untrim (M36-F11) recovers the natural NURBS bounds of a trimmed face: it rebuilds the face on its
// full surface domain, bounded by the four boundary iso-curves, discarding the trim loops. The base
// surface is unchanged, so re-applying the original trim reproduces the face.

// UntrimFace returns a single-face surface body covering the full domain of the given face's NURBS
// surface (its four boundary iso-curves as the outer loop). It errors when the face is not a NURBS
// surface.
func UntrimFace(b *topo.Body, faceKey []byte) (*topo.Body, error) {
	f, ok := b.FindFaceByKey(faceKey)
	if !ok {
		return nil, fmt.Errorf("surface.UntrimFace: no face with key %x", faceKey)
	}
	surf, ok := f.Geometry().(geom.BSplineSurface)
	if !ok {
		return nil, fmt.Errorf("surface.UntrimFace: face is not a NURBS surface (%T)", f.Geometry())
	}
	return retopo.FullDomainBody(surf, "untrim"), nil
}
