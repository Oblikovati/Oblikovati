// SPDX-License-Identifier: GPL-2.0-only

package predicates

// Orient2D returns the exact sign of the orientation of the triangle (a,b,c):
// +1 if a, b, c occur in counterclockwise order, -1 if clockwise, 0 if collinear.
// The result is exact — never a tolerance decision — so all callers agree on
// which side of line ab a point c lies.
//
// Example:
//
//	predicates.Orient2D(0, 0, 1, 0, 0, 1) // +1 (counterclockwise)
func Orient2D(ax, ay, bx, by, cx, cy float64) int {
	if det, certified := filterOrient2D(ax, ay, bx, by, cx, cy); certified {
		return signOf(det)
	}
	return exactOrient2D(ax, ay, bx, by, cx, cy)
}

// Orient3D returns the exact sign of the position of point d relative to the
// oriented plane through a, b, c: +1 if d lies below the plane (a, b, c appear
// counterclockwise viewed from above d), -1 if above, 0 if the four points are
// coplanar. The result is exact, so "d is on plane(a,b,c)" is one global truth.
//
// Example:
//
//	predicates.Orient3D(0,0,0, 1,0,0, 0,1,0, 0,0,1) // -1 (d above the xy-plane)
func Orient3D(ax, ay, az, bx, by, bz, cx, cy, cz, dx, dy, dz float64) int {
	if det, certified := filterOrient3D(ax, ay, az, bx, by, bz, cx, cy, cz, dx, dy, dz); certified {
		return signOf(det)
	}
	return exactOrient3D(ax, ay, az, bx, by, bz, cx, cy, cz, dx, dy, dz)
}

// signOf maps a certified determinant estimate to its sign.
func signOf(det float64) int {
	switch {
	case det > 0:
		return 1
	case det < 0:
		return -1
	default:
		return 0
	}
}
