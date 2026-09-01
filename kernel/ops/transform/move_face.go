// SPDX-License-Identifier: GPL-2.0-only

package transform

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/retopo"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// MoveFaces translates the selected faces of a planar solid by delta, retrimming the
// neighbours: each selected face's plane is translated (normal unchanged) and every vertex
// re-solves at its faces' new plane intersections (via [retopo.RebuildWithPlanes]). The
// combinatorial topology is preserved, so the result stays a valid solid for a modest move
// (a move large enough to collapse or invert a face is a follow-up needing retopology).
func MoveFaces(solid *topo.Body, faceKeys [][]byte, delta math.Vector3) (*topo.Body, error) {
	sel, err := retopo.ResolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	return retopo.RebuildWithPlanes(solid, "move-face", true, func(f *topo.Face) geom.Plane {
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
	sel, err := retopo.ResolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	return retopo.RebuildWithPlanes(solid, "offset-face", true, func(f *topo.Face) geom.Plane {
		pl := f.Geometry().(geom.Plane)
		if !sel[f.ID()] {
			return pl
		}
		return shiftPlane(pl, pl.Normal().Scale(dist))
	}), nil
}

// RotateFaces rotates the selected faces' planes by angle about an axis (point + unit
// direction), retrimming the neighbours — the rotate arm of the move-face direct edit
// (#331). Same contract as [MoveFaces]: topology is preserved, so a rotation large enough
// to collapse or invert a face is a follow-up needing retopology.
func RotateFaces(solid *topo.Body, faceKeys [][]byte, axisPoint math.Point3, axisDir math.UnitVector3, angle float64) (*topo.Body, error) {
	sel, err := retopo.ResolveFaceSet(solid, faceKeys)
	if err != nil {
		return nil, err
	}
	rot := math.Rotation4(angle, axisDir, axisPoint)
	return retopo.RebuildWithPlanes(solid, "rotate-face", true, func(f *topo.Face) geom.Plane {
		pl := f.Geometry().(geom.Plane)
		if !sel[f.ID()] {
			return pl
		}
		return rotatePlane(pl, rot)
	}), nil
}

// shiftPlane returns the plane translated by delta (origin moved, basis/normal unchanged).
func shiftPlane(pl geom.Plane, delta math.Vector3) geom.Plane {
	moved, _ := geom.NewPlaneFromAxes(pl.Origin.TranslateBy(delta), pl.UAxis.AsVector(), pl.VAxis.AsVector())
	return moved
}

// rotatePlane returns the plane mapped through the rotation (origin and basis rotated).
func rotatePlane(pl geom.Plane, rot math.Matrix4) geom.Plane {
	moved, _ := geom.NewPlaneFromAxes(rot.TransformPoint(pl.Origin),
		rot.TransformVector(pl.UAxis.AsVector()), rot.TransformVector(pl.VAxis.AsVector()))
	return moved
}
