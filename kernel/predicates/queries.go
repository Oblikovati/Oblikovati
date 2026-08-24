// SPDX-License-Identifier: GPL-2.0-only

package predicates

import "math"

// Higher-level exact geometric queries composed from the sign predicates. Because
// they reduce to Orient2D/Orient3D signs, their answers are exact: a point is
// inside a triangle, or a segment pierces one, as one global truth with no
// tolerance. These are the co-refinement building blocks for the ADR-0052
// cell-arrangement core. Points are [3]float64 to keep the package dependency-free.

// TriRegion is the exact position of a point relative to a triangle.
type TriRegion int

const (
	Outside    TriRegion = iota // strictly outside the triangle
	OnBoundary                  // on an edge or vertex
	Inside                      // strictly interior
)

// InTriangleCoplanar returns the exact position of p relative to triangle
// (a,b,c). PRECONDITION: p is coplanar with a,b,c (Orient3D(a,b,c,p)==0) and the
// triangle is non-degenerate; the caller establishes both. It projects onto the
// coordinate plane least parallel to the triangle — an exact operation (dropping
// a coordinate) that keeps the projected triangle non-degenerate — and decides
// with three exact 2D edge orientations.
//
// Example:
//
//	predicates.InTriangleCoplanar(
//	    [3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0},
//	    [3]float64{1, 1, 0}) // Inside
func InTriangleCoplanar(a, b, c, p [3]float64) TriRegion {
	axis := dominantNormalAxis(a, b, c)
	au, av := drop(a, axis)
	bu, bv := drop(b, axis)
	cu, cv := drop(c, axis)
	pu, pv := drop(p, axis)
	d1 := Orient2D(au, av, bu, bv, pu, pv)
	d2 := Orient2D(bu, bv, cu, cv, pu, pv)
	d3 := Orient2D(cu, cv, au, av, pu, pv)
	// p is inside/on iff it is on the same side of all three directed edges. Mixed
	// signs (one strictly left, another strictly right) put it outside.
	hasNeg := d1 < 0 || d2 < 0 || d3 < 0
	hasPos := d1 > 0 || d2 > 0 || d3 > 0
	if hasNeg && hasPos {
		return Outside
	}
	if d1 == 0 || d2 == 0 || d3 == 0 {
		return OnBoundary
	}
	return Inside
}

// SegmentPiercesTriangle reports whether the open segment (p,q) crosses the
// interior of triangle (a,b,c) transversally — the two endpoints strictly
// straddle the triangle's plane and the pierce point is strictly inside. It is
// the exact transversal-crossing test the co-refinement uses to decide that a
// segment must be split by a face. Coplanar segments and endpoint-on-plane
// touches return false: those are boundary-incidence cases the co-refinement
// layer resolves separately, not transversal pierces.
//
// Example:
//
//	predicates.SegmentPiercesTriangle(
//	    [3]float64{1, 1, -1}, [3]float64{1, 1, 1},
//	    [3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0}) // true
func SegmentPiercesTriangle(p, q, a, b, c [3]float64) bool {
	sp := orient3(a, b, c, p)
	sq := orient3(a, b, c, q)
	if sp == 0 || sq == 0 || sp == sq {
		return false // not strictly straddling the plane
	}
	// The line pq pierces the triangle interior iff p,q lie strictly on the same
	// rotational side of all three edges (the three signed tetra volumes agree and
	// are nonzero). A zero would put the pierce on an edge — a boundary touch,
	// excluded here.
	s1 := orient3(p, q, a, b)
	s2 := orient3(p, q, b, c)
	s3 := orient3(p, q, c, a)
	if s1 == 0 || s2 == 0 || s3 == 0 {
		return false
	}
	return s1 == s2 && s2 == s3
}

// dominantNormalAxis returns the coordinate axis (0=x,1=y,2=z) most parallel to
// the triangle's normal, i.e. the one to drop for a non-degenerate 2D projection.
// It uses a plain float normal because only the CHOICE must be reasonable; the
// subsequent Orient2D tests are exact regardless of which valid axis is dropped.
func dominantNormalAxis(a, b, c [3]float64) int {
	ux, uy, uz := b[0]-a[0], b[1]-a[1], b[2]-a[2]
	vx, vy, vz := c[0]-a[0], c[1]-a[1], c[2]-a[2]
	nx := math.Abs(uy*vz - uz*vy)
	ny := math.Abs(uz*vx - ux*vz)
	nz := math.Abs(ux*vy - uy*vx)
	if nx >= ny && nx >= nz {
		return 0
	}
	if ny >= nz {
		return 1
	}
	return 2
}

// drop projects p onto the coordinate plane perpendicular to axis, returning the
// two surviving coordinates. Dropping a coordinate is exact.
func drop(p [3]float64, axis int) (u, v float64) {
	switch axis {
	case 0:
		return p[1], p[2]
	case 1:
		return p[0], p[2]
	default:
		return p[0], p[1]
	}
}

// orient3 is the [3]float64 spelling of Orient3D, kept internal so the query
// functions read as geometry rather than coordinate lists.
func orient3(a, b, c, d [3]float64) int {
	return Orient3D(a[0], a[1], a[2], b[0], b[1], b[2], c[0], c[1], c[2], d[0], d[1], d[2])
}
