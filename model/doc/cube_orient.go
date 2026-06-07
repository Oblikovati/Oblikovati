// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	stdmath "math"

	"oblikovati/math"
)

// CubeOrient is a document's ViewCube orientation: the world-space images of the cube's
// local +X/+Y/+Z axes (a rotation, stored as three orthonormal columns). The identity is
// the default front; "Set Current View as Front" replaces it so a chosen view becomes the
// cube's front, and "Reset Front" restores the identity. It maps cube-local directions
// (the cube's own face normals) to world directions and back.
type CubeOrient struct{ X, Y, Z math.Vector3 }

// IdentityCubeOrient is the default front: cube-local axes equal world axes.
func IdentityCubeOrient() CubeOrient {
	return CubeOrient{X: math.V3(1, 0, 0), Y: math.V3(0, 1, 0), Z: math.V3(0, 0, 1)}
}

// ToWorld maps a cube-local vector to world space (R·v).
func (o CubeOrient) ToWorld(v math.Vector3) math.Vector3 {
	return o.X.Scale(v.X).Add(o.Y.Scale(v.Y)).Add(o.Z.Scale(v.Z))
}

// ToLocal maps a world vector to cube-local space (Rᵀ·v, the inverse of a rotation).
func (o CubeOrient) ToLocal(v math.Vector3) math.Vector3 {
	return math.V3(o.X.Dot(v), o.Y.Dot(v), o.Z.Dot(v))
}

// IsIdentity reports whether this is the default (un-redefined) front.
func (o CubeOrient) IsIdentity() bool { return o == IdentityCubeOrient() }

// FrontFromView builds the orientation that makes the view looking along fwd (with up) the
// cube's front view, axis-snapped so the cube stays aligned to the principal axes (the
// chosen view is normally already a standard one). fwd/up need not be normalized.
func FrontFromView(fwd, up math.Vector3) CubeOrient {
	y := nearestAxis(fwd) // local +Y maps to the (snapped) view direction ⇒ front view
	z := nearestAxis(up)
	if axisParallel(z, y) {
		z = nearestAxisExcluding(up, y)
	}
	return CubeOrient{X: y.Cross(z), Y: y, Z: z} // right-handed: X = Y × Z
}

// nearestAxis returns the signed principal axis (±X/±Y/±Z) closest to v.
func nearestAxis(v math.Vector3) math.Vector3 {
	ax, ay, az := stdmath.Abs(v.X), stdmath.Abs(v.Y), stdmath.Abs(v.Z)
	switch {
	case ax >= ay && ax >= az:
		return math.V3(axisSign(v.X), 0, 0)
	case ay >= az:
		return math.V3(0, axisSign(v.Y), 0)
	default:
		return math.V3(0, 0, axisSign(v.Z))
	}
}

// nearestAxisExcluding returns the principal axis closest to v that is not parallel to
// exclude (used when the view's up snaps onto the same axis as its direction).
func nearestAxisExcluding(v, exclude math.Vector3) math.Vector3 {
	best, bestDot := math.Vector3{}, -1.0
	for _, a := range []math.Vector3{math.V3(1, 0, 0), math.V3(-1, 0, 0), math.V3(0, 1, 0), math.V3(0, -1, 0), math.V3(0, 0, 1), math.V3(0, 0, -1)} {
		if axisParallel(a, exclude) {
			continue
		}
		if d := a.Dot(v); d > bestDot {
			best, bestDot = a, d
		}
	}
	return best
}

func axisParallel(a, b math.Vector3) bool { return stdmath.Abs(a.Dot(b)) > 0.5 }

func axisSign(f float64) float64 {
	if f < 0 {
		return -1
	}
	return 1
}
