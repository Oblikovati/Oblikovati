// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// MoveFaces translates the selected faces of a planar solid by delta, retrimming the
// neighbours: each selected face's plane is translated (normal unchanged) and every vertex
// re-solves at its faces' new plane intersections (via [rebuildWithPlanes]). The
// combinatorial topology is preserved, so the result stays a valid solid for a modest move
// (a move large enough to collapse or invert a face is a follow-up needing retopology).
func MoveFaces(solid *topo.Body, faceKeys [][]byte, delta math.Vector3) (*topo.Body, error) {
	sel, err := resolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	return rebuildWithPlanes(solid, "move-face", func(f *topo.Face) geom.Plane {
		pl := f.Geometry().(geom.Plane)
		if !sel[f.ID()] {
			return pl
		}
		return shiftPlane(pl, delta)
	}), nil
}

// OffsetFaces moves each selected face along its own outward normal by dist (positive grows
// the solid there, negative shaves it) — MoveFaces applied per face in its normal direction.
func OffsetFaces(solid *topo.Body, faceKeys [][]byte, dist float64) (*topo.Body, error) {
	sel, err := resolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	return rebuildWithPlanes(solid, "offset-face", func(f *topo.Face) geom.Plane {
		pl := f.Geometry().(geom.Plane)
		if !sel[f.ID()] {
			return pl
		}
		return shiftPlane(pl, pl.Normal().Scale(dist))
	}), nil
}

// shiftPlane returns the plane translated by delta (origin moved, basis/normal unchanged).
func shiftPlane(pl geom.Plane, delta math.Vector3) geom.Plane {
	moved, _ := geom.NewPlaneFromAxes(pl.Origin.TranslateBy(delta), pl.UAxis.AsVector(), pl.VAxis.AsVector())
	return moved
}
