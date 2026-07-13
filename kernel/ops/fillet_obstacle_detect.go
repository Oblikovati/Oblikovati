// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/math"
)

// boundaryLine2 is the receded fillet boundary in the host plane: a point and a UNIT direction.
// It is deliberately unexported and local to this detector — the kernel's general-purpose
// geom.Line2d has no signed-distance query, and this slice only ever needs the perp-dot test below.
type boundaryLine2 struct {
	origin math.Point2
	dir    math.Vector2 // must be unit length
}

// signedDist returns the signed distance from p to the boundary line: +ve on the host side,
// -ve on the fillet side (the side dir's left-hand perpendicular points away from).
func (b boundaryLine2) signedDist(p math.Point2) float64 {
	return b.dir.Cross(b.origin.VectorTo(p))
}

// crossing is one intersection of the obstacle rim (as a sampled polyline) with the receded boundary
// line: the rim index just before it and the intersection point in host-plane 2D.
type crossing struct {
	I int         // rim polyline index whose segment [I, I+1] crosses the boundary
	P math.Point2 // the crossing point
}

// rimCrossings returns the boundary crossings of the closed rim polyline, ordered as they appear along
// the rim. A crossing is a SIGN CHANGE of the signed distance to the boundary line larger than the
// model weld — so a vertex merely grazing the boundary (|d| ≤ weld on both sides) is NOT a crossing
// (the tangency guard, spec §Numerical pitfalls).
func rimCrossings(rim []math.Point2, b boundaryLine2, res Resolution) []crossing {
	tol := res.Weld()
	var out []crossing
	n := len(rim)
	for i := 0; i < n; i++ {
		a, c := rim[i], rim[(i+1)%n]
		da, dc := b.signedDist(a), b.signedDist(c)
		if da > tol && dc < -tol || da < -tol && dc > tol {
			out = append(out, crossing{I: i, P: lerpAtZero(a, c, da, dc)})
		}
	}
	return out
}

// lerpAtZero returns the point on segment a→c where the signed distance crosses zero.
func lerpAtZero(a, c math.Point2, da, dc float64) math.Point2 {
	t := da / (da - dc)
	return a.Lerp(c, t) // Point2.Lerp: stable single-eval, exact at t=0/1 (#1654)
}

// obstacleNodes returns the two rim crossing indices bracketing the dip past the boundary, or
// ok=false when the rim does not genuinely cross twice (tangential touch or no dip → honest-reject,
// ADR-3). Exactly two crossings is the single-dip case this slice handles; >2 (a rim weaving across)
// is a tracked follow-up and also returns ok=false here so the caller honest-rejects rather than
// mis-building.
func obstacleNodes(rim []math.Point2, b boundaryLine2, res Resolution) ([2]crossing, bool) {
	cs := rimCrossings(rim, b, res)
	if len(cs) != 2 {
		return [2]crossing{}, false
	}
	return [2]crossing{cs[0], cs[1]}, true
}

// dipsPast reports whether the rim actually dips PAST the boundary between the two crossings (into the
// fillet band), vs. bulging away — the mid-arc sample (the forward arc c0→c1, wrapping the array when
// c0.I > c1.I) must be on the fillet side. side is +1 when the fillet band is on the
// negative-signed-distance side of the boundary (signedDist: host +ve, fillet -ve). A genuine dip has
// the mid sample in the fillet band, so side*signedDist(mid) is NEGATIVE — hence the `< 0` test (a
// bulge keeps the mid on the host side, giving a positive product → false).
func dipsPast(rim []math.Point2, c0, c1 crossing, b boundaryLine2, side float64) bool {
	mid := rim[(c0.I+1+((c1.I-c0.I+len(rim))%len(rim))/2)%len(rim)]
	return side*b.signedDist(mid) < 0
}
