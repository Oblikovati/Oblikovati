// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The covering's CANONICAL translate (Oblikovati/Oblikovati#3518).
//
// A covering mesher replicates its points a whole period either way and triangulates the lot once, so
// every triangle near the branch window's edge exists twice — once at each end. Exactly one may ship.
// That used to be the half-open window test alone, and for a POINT it is exact: of p and its
// translates exactly one satisfies uLo <= u < uHi.
//
// For a TRIANGLE it is not. The test reads the centroid, and a replica's centroid is recomputed from
// its own shifted vertices, so it equals the original's plus a period only to within rounding.
// Measured on the near-pinch crossing rods (r = 3 joined to r = 3.00004) at PropertyQuality: one
// corridor triangle's centroid landed on u = uLo = pi exactly while its +2pi replica computed just
// BELOW uHi, the half-open test took both, and the face shipped that triangle twice.
//
// So the window is now the CANDIDATE filter — chartRegion.windowCandidate, closed at both ends, which
// guarantees at least one translate of every triangle is offered — and the choice among candidates is
// made on the WELDED 3D triangle, which period replicas share exactly: addReplicas gives every
// translate of a covering point the same math.Point3. One translate per 3D triangle, picked by a
// total order (nearest the window's middle, then the triangulation's own order), so the answer cannot
// turn on rounding. Measured on that face: duplicate welded triangles 1 -> 0, and the unpaired edges
// carrying THREE triangles 4 -> 1.
//
// WHAT IS STILL LEFT, because the next task (#3517) turns on it. Six of the sixteen near-pinch rows
// still come back with 4-8 unpaired edges that are no rim segment, every one of them at the seam. The
// cause is not this selection and not the replication pad — chartCoverPadStations at 3, 6, 12 and 24
// gives byte-identical numbers. It is that the two ends of the branch window are triangulated
// INDEPENDENTLY and do not agree. Measured, on the same face: the corridor's four ear-centre points
// near u = pi form a near-rectangle 0.0035 x 0.00029 in (u,v), so the two candidate diagonals are
// within rounding of each other, and the low image splits it (3.139868, 5.998945)-(3.143318,
// 5.998657) while the +2pi image splits it the other way. The canonical selection then takes one
// triangle from each variant and the quad is left with a crack and a three-triangle edge.
//
// A covering's premise is that its two ends are the same triangulation, and that holds only if the
// translated coordinates are EXACTLY the originals plus a period. They are not: fl(u + 2pi) and then
// the metric multiply by su both round. Snapping the covering's parameters onto a binary grid so the
// period shift is exact was tried and measured — it takes the sixteen rows from 10 clean to 14 — but
// it cannot be made exact while the triangulation runs on u*su, because floating multiplication does
// not distribute over the shift; the remaining two rows are the proof. Making it exact needs the
// constrained triangulation to work in fixed point, which is its own change.
//
// Until then the near-pinch band keeps the unrolled arm (curved_trim_recognize.go's corridor gate) and
// the eight join bodies keep the tessellate.cap-saturated their 64-cell grid earns.

// keepOneReplicaEach keeps exactly ONE covering triangle per 3D triangle.
func (b *chartCover) keepOneReplicaEach(tris [][3]int, keep []bool) {
	grid := weldGrid([][]math.Point3{b.pos})
	best := map[[3][3]int64]int{}
	for i, t := range tris {
		if !keep[i] {
			continue
		}
		k := weldedTriangleKey(b.pos, t, grid)
		if j, seen := best[k]; !seen || b.nearerTheWindowCentre(t, tris[j]) {
			best[k] = i
		}
	}
	chosen := make([]bool, len(tris))
	for _, i := range best {
		chosen[i] = true
	}
	for i := range keep {
		keep[i] = keep[i] && chosen[i]
	}
}

// weldedTriangleKey is a covering triangle's identity as a 3D triangle: its three welded vertices in a
// fixed order, so period replicas — which share their 3D points exactly — key the same.
func weldedTriangleKey(pos []math.Point3, t [3]int, grid float64) [3][3]int64 {
	k := [3][3]int64{quantizePoint(pos[t[0]], grid), quantizePoint(pos[t[1]], grid), quantizePoint(pos[t[2]], grid)}
	for i := range 3 {
		for j := i + 1; j < 3; j++ {
			if weldKeyLess(k[j], k[i]) {
				k[i], k[j] = k[j], k[i]
			}
		}
	}
	return k
}

// weldKeyLess is the total order on welded vertex keys the triangle key sorts by.
func weldKeyLess(a, c [3]int64) bool {
	if a[0] != c[0] {
		return a[0] < c[0]
	}
	if a[1] != c[1] {
		return a[1] < c[1]
	}
	return a[2] < c[2]
}

// nearerTheWindowCentre reports whether a sits nearer the branch window's middle than c does — the
// canonical translate, chosen on the centroid so it does not depend on which vertex came first.
func (b *chartCover) nearerTheWindowCentre(a, c [3]int) bool {
	return b.centreOffset(a) < b.centreOffset(c)
}

// centreOffset is how far a triangle's centroid stands from the branch window's middle, as the larger
// of the two wrapping axes' offsets in periods (a bounded axis contributes nothing).
func (b *chartCover) centreOffset(t [3]int) float64 {
	u, v := b.centroid(t)
	return stdmath.Max(axisCentreOffset(u, b.r.uLo, b.r.uHi, b.r.uPer),
		axisCentreOffset(v, b.r.vLo, b.r.vHi, b.r.vPer))
}

// axisCentreOffset is one axis's distance from the window's middle, in periods.
func axisCentreOffset(x, lo, hi float64, periodic bool) float64 {
	if !periodic || hi <= lo {
		return 0
	}
	return stdmath.Abs(x-(lo+hi)/2) / (hi - lo)
}
