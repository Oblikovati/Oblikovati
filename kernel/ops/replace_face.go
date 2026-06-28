// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// ReplaceFaces replaces the surface of each selected face with the target plane and retrims
// the neighbours (via [rebuildWithPlanes]): the selected faces are moved onto target and
// every vertex re-solves at its faces' new plane intersections. It is the planar form of
// "replace face with a surface" — e.g. raising a step's top onto a higher face's plane. The
// topology is preserved, so the result stays valid for a target that does not invert a face.
func ReplaceFaces(solid *topo.Body, faceKeys [][]byte, target geom.Plane) (*topo.Body, error) {
	sel, err := resolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	return rebuildWithPlanes(solid, "replace-face", true, func(f *topo.Face) geom.Plane {
		if sel[f.ID()] {
			return target
		}
		return f.Geometry().(geom.Plane)
	}), nil
}

// PlaneOfFace returns the plane of a face by its reference key (the target source for a
// replace-face), or ok=false if the key no longer resolves or the face is not planar.
func PlaneOfFace(solid *topo.Body, key []byte) (geom.Plane, bool) {
	f, ok := solid.FindFaceByKey(key)
	if !ok {
		return geom.Plane{}, false
	}
	pl, planar := f.Geometry().(geom.Plane)
	return pl, planar
}
