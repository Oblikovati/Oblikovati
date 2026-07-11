// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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

// ReplaceFacesMulti replaces each selected face with the target plane it best matches — the
// nearest by its centroid's distance to the plane — and retrims the neighbours. With a single
// target it is exactly [ReplaceFaces]; with several it lets one Replace Face flatten different
// picked faces onto different new faces / work planes (#1886). It errors on a lost pick or an
// empty target set.
func ReplaceFacesMulti(solid *topo.Body, faceKeys [][]byte, targets []geom.Plane) (*topo.Body, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("replace-face: no new faces")
	}
	sel, err := resolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	assign := make(map[uint64]geom.Plane, len(sel))
	for _, f := range solid.Faces() {
		if sel[f.ID()] {
			assign[f.ID()] = nearestPlane(centroidPts(facePolygon(f)), targets)
		}
	}
	return rebuildWithPlanes(solid, "replace-face", true, func(f *topo.Face) geom.Plane {
		if pl, ok := assign[f.ID()]; ok {
			return pl
		}
		return f.Geometry().(geom.Plane)
	}), nil
}

// nearestPlane returns the target plane p minimizes |n·(c − origin)| for — the one c sits closest to.
func nearestPlane(c math.Point3, targets []geom.Plane) geom.Plane {
	best, bestD := targets[0], stdmath.Inf(1)
	for _, pl := range targets {
		if d := stdmath.Abs(float64(pl.Normal().Dot(pl.Origin.VectorTo(c)))); d < bestD {
			best, bestD = pl, d
		}
	}
	return best
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
