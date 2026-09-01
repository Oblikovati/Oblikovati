// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/heal"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
)

// OffsetFaces returns a sheet body holding just the named faces of b, each on its surface offset by
// distance along the face's outward (material) normal. It is the surface-offset primitive CAM uses
// for 3D surfacing tool compensation: offset the part face by the tool radius, then sample the result
// for drop-cutter projection. reverse offsets the opposite way (into the solid).
//
// Each face keeps its loops; only the surface is swapped for a geom.OffsetSurface. Because the offset
// surface shares the base's (u,v) parameterization and ParamAt, the trimmed-face tessellator
// re-evaluates the same UV region on the offset surface — so the offset face tessellates and samples
// correctly with no edge surgery. The result is a surface (non-solid) body.
func OffsetFaceSurfaces(b *topo.Body, faceKeys [][]byte, distance float64, reverse bool) (*topo.Body, error) {
	if len(faceKeys) == 0 {
		return nil, fmt.Errorf("ops.OffsetFaceSurfaces: no faces selected")
	}
	// Swap each selected face's surface for its offset while the body is still the original solid
	// (single shell), then keep only those faces — so transform.ReplaceFaceSurface never sees a split body.
	out := b
	for _, key := range faceKeys {
		f, ok := out.FindFaceByKey(key)
		if !ok {
			return nil, fmt.Errorf("ops.OffsetFaceSurfaces: no face with key %x", key)
		}
		off, err := geom.NewOffsetSurface(f.Geometry(), signedOffset(distance, reverse, f.Reversed()))
		if err != nil {
			return nil, fmt.Errorf("ops.OffsetFaceSurfaces: offset face %x: %w", key, err)
		}
		if out, err = transform.ReplaceFaceSurface(out, key, off); err != nil {
			return nil, fmt.Errorf("ops.OffsetFaceSurfaces: offset face %x: %w", key, err)
		}
	}
	kept, err := heal.DropFaces(out, faceKeys, true)
	if err != nil {
		return nil, fmt.Errorf("ops.OffsetFaceSurfaces: isolate offset faces: %w", err)
	}
	return kept, nil
}

// signedOffset resolves the offset distance so a positive distance moves the face along its OUTWARD
// (material) normal: reverse flips it, and a reversed face (whose surface normal points into material)
// flips it again, so the surface offset always runs away from the solid for a positive distance.
func signedOffset(distance float64, reverse, faceReversed bool) float64 {
	if reverse {
		distance = -distance
	}
	if faceReversed {
		distance = -distance
	}
	return distance
}
