// SPDX-License-Identifier: GPL-2.0-only

package math

// NewOrientedBoxFromEdges builds an oriented box from a corner and the three edge vectors
// spanning from it — the corner+edge-vectors form kernel/ops.OrientedMinimumRangeBox
// produces and the reference types.OrientedBox exposes (audit B9, #1620: the seam that let
// the former kernel/ops.OrientedBox twin be collapsed into [OrientedBox]). The edges are
// assumed mutually orthogonal; each becomes a unit axis with half its length as the
// half-extent. A zero-length edge yields a zero half-extent on an orthonormally completed
// axis, so a degenerate (flat/segment/point) box stays representable without a non-unit axis.
//
// Example: obb := math.NewOrientedBoxFromEdges(corner, e0, e1, e2)
func NewOrientedBoxFromEdges(corner Point3, e0, e1, e2 Vector3) OrientedBox {
	edges := [3]Vector3{e0, e1, e2}
	axes, half := orientedFrame(edges)
	center := corner
	for _, e := range edges {
		center = center.TranslateBy(e.Scale(0.5))
	}
	return OrientedBox{Center: center, Axes: axes, HalfExtents: half}
}

// orientedFrame splits three (assumed orthogonal) edge vectors into unit axes and
// half-extents, completing any zero-length edge's axis orthonormally so [OrientedBox.Contains]
// can still project onto an orthonormal frame.
func orientedFrame(edges [3]Vector3) ([3]UnitVector3, [3]Scalar) {
	var axes [3]UnitVector3
	var half [3]Scalar
	for i, e := range edges {
		half[i] = e.Length() * 0.5
		if u, err := UnitVector3FromVector(e); err == nil {
			axes[i] = u
		}
	}
	return completeOrthonormal(axes), half
}

// completeOrthonormal fills any unset (zero) axis with a unit vector orthogonal to the
// already-set ones, in order, so the three axes end up orthonormal even after a degenerate
// edge. A solid body never produces a zero edge; this only guards pathological input.
func completeOrthonormal(axes [3]UnitVector3) [3]UnitVector3 {
	for i := range axes {
		if !axisSet(axes[i]) {
			axes[i] = perpToOthers(axes, i)
		}
	}
	return axes
}

// perpToOthers returns a unit vector orthogonal to whichever of the OTHER two axes are set:
// their cross product when both are set, any perpendicular of the single set one, or the
// world axis i when neither is.
func perpToOthers(axes [3]UnitVector3, i int) UnitVector3 {
	j, k := (i+1)%3, (i+2)%3
	switch {
	case axisSet(axes[j]) && axisSet(axes[k]):
		return axes[j].Cross(axes[k]).AsUnit()
	case axisSet(axes[j]):
		return anyPerpendicular(axes[j])
	case axisSet(axes[k]):
		return anyPerpendicular(axes[k])
	default:
		return worldAxis(i)
	}
}

// axisSet reports whether u is a real (non-zero) axis rather than the zero value left by a
// degenerate edge.
func axisSet(u UnitVector3) bool { return u.AsVector().LengthSquared() != 0 }

// worldAxis returns the i-th world basis vector as a unit vector.
func worldAxis(i int) UnitVector3 {
	switch i {
	case 0:
		return V3(1, 0, 0).AsUnit()
	case 1:
		return V3(0, 1, 0).AsUnit()
	default:
		return V3(0, 0, 1).AsUnit()
	}
}
