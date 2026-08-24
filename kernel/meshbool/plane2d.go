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

// inCircleSign returns the exact in-circle sign of d relative to triangle (a,b,c),
// projected on axis: +1 if d lies strictly inside the circumcircle when a,b,c are
// counterclockwise, -1 outside, 0 cocircular. The exact-arithmetic counterpart of
// predicates.InCircle, on rational Points, so the Delaunay flip that builds the
// robust constrained triangulation never oscillates on a near-cocircular quad.
func inCircleSign(a, b, c, d Point, axis int) int {
	au, av := project(a, axis)
	bu, bv := project(b, axis)
	cu, cv := project(c, axis)
	du, dv := project(d, axis)
	adx, ady := new(big.Rat).Sub(au, du), new(big.Rat).Sub(av, dv)
	bdx, bdy := new(big.Rat).Sub(bu, du), new(big.Rat).Sub(bv, dv)
	cdx, cdy := new(big.Rat).Sub(cu, du), new(big.Rat).Sub(cv, dv)
	alift := ratSquareSum(adx, ady)
	blift := ratSquareSum(bdx, bdy)
	clift := ratSquareSum(cdx, cdy)
	t1 := new(big.Rat).Mul(alift, crossDiff(bdx, cdy, cdx, bdy))
	t2 := new(big.Rat).Mul(blift, crossDiff(cdx, ady, adx, cdy))
	t3 := new(big.Rat).Mul(clift, crossDiff(adx, bdy, bdx, ady))
	return t1.Add(t1, t2).Add(t1, t3).Sign()
}

// ratSquareSum returns x*x + y*y exactly.
func ratSquareSum(x, y *big.Rat) *big.Rat {
	xx := new(big.Rat).Mul(x, x)
	yy := new(big.Rat).Mul(y, y)
	return xx.Add(xx, yy)
}

// segmentsProperlyCross reports whether segments ab and cd cross transversally in
// the projection: each segment strictly straddles the other's supporting line, so
// the crossing is strictly interior to both. Shared or on-line endpoints (a zero
// orientation) are not proper crossings.
func segmentsProperlyCross(a, b, c, d Point, axis int) bool {
	s1 := orient2(a, b, c, axis)
	s2 := orient2(a, b, d, axis)
	s3 := orient2(c, d, a, axis)
	s4 := orient2(c, d, b, axis)
	return s1 != 0 && s2 != 0 && s1 != s2 && s3 != 0 && s4 != 0 && s3 != s4
}

// lerpPoint returns the exact 3D point a + t*(b-a).
func lerpPoint(a, b Point, t *big.Rat) Point {
	return Point{
		X: lerp(a.X, b.X, t),
		Y: lerp(a.Y, b.Y, t),
		Z: lerp(a.Z, b.Z, t),
	}
}
