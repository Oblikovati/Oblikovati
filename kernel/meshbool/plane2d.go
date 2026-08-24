// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// Exact 2D geometry within a face's plane. The plane is not axis-aligned in
// general, so points are projected onto the coordinate plane least parallel to the
// face normal (dropping one coordinate is exact and keeps the projection
// non-degenerate). All orientation and crossing decisions are then exact rational
// 2D operations. Constructed crossings are lifted back to exact 3D points that lie
// on the originating segments — the conforming property carried into the plane.

// planeAxis returns the coordinate axis (0=x,1=y,2=z) to drop for a non-degenerate
// 2D projection of triangle tri's plane — the axis most parallel to its normal.
func planeAxis(tri [3]Point) int {
	n := triNormal(tri)
	ax := new(big.Rat).Abs(n[0])
	ay := new(big.Rat).Abs(n[1])
	az := new(big.Rat).Abs(n[2])
	if ax.Cmp(ay) >= 0 && ax.Cmp(az) >= 0 {
		return 0
	}
	if ay.Cmp(az) >= 0 {
		return 1
	}
	return 2
}

// project returns p's two surviving coordinates after dropping axis.
func project(p Point, axis int) (u, v *big.Rat) {
	switch axis {
	case 0:
		return p.Y, p.Z // drop x → keep (y,z)
	case 1:
		return p.X, p.Z // drop y → keep (x,z)
	default:
		return p.X, p.Y // drop z → keep (x,y)
	}
}

// orient2Val returns the exact signed area determinant of the projected triangle
// (a,b,c): (au-cu)(bv-cv) - (av-cv)(bu-cu).
func orient2Val(a, b, c Point, axis int) *big.Rat {
	au, av := project(a, axis)
	bu, bv := project(b, axis)
	cu, cv := project(c, axis)
	return crossDiff(
		new(big.Rat).Sub(au, cu), new(big.Rat).Sub(bv, cv),
		new(big.Rat).Sub(av, cv), new(big.Rat).Sub(bu, cu))
}

// orient2 returns the exact sign of orient2Val: +1 CCW, -1 CW, 0 collinear, in the
// projected plane.
func orient2(a, b, c Point, axis int) int {
	return orient2Val(a, b, c, axis).Sign()
}

// SegSegCross returns the exact point where segment ab meets the line through c,d,
// both lying in the projection plane of axis. PRECONDITION: a and b lie strictly
// on opposite sides of line cd (orient2(c,d,a) and orient2(c,d,b) opposite,
// nonzero). The result lies exactly on segment ab and on line cd.
func SegSegCross(a, b, c, d Point, axis int) Point {
	da := orient2Val(c, d, a, axis)
	db := orient2Val(c, d, b, axis)
	// t solves da + t*(db-da) = 0 (orient2 is affine along ab): t = da/(da-db).
	t := new(big.Rat).Quo(da, new(big.Rat).Sub(da, db))
	return lerpPoint(a, b, t)
}

// lerpPoint returns the exact 3D point a + t*(b-a).
func lerpPoint(a, b Point, t *big.Rat) Point {
	return Point{
		X: lerp(a.X, b.X, t),
		Y: lerp(a.Y, b.Y, t),
		Z: lerp(a.Z, b.Z, t),
	}
}
