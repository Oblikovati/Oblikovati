// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import "oblikovati.org/math"

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
// So the window is now the CANDIDATE filter — closed at both ends, which offers at least one translate
// of every triangle — and the choice among candidates is made in two exact steps. WHICH triangles are
// replicas of each other is read off the WELDED 3D triangle, which replicas share exactly: addReplicas
// gives every translate of a covering point the same math.Point3. WHICH of them ships is read off the
// integer SHIFT INDEX each vertex was laid at, summed over the three — an integer the owner chose, so
// no rounding enters and the answer is the same on every platform.
//
// The shift index and not a distance, and that correction is #3518 review I5. The first cut ordered on
// how far the centroid sat from the window's middle, which is not a discriminator at all: a triangle
// whose centroid lies ON uLo is half a window from the middle, and so is its image on uHi — measured,
// the two candidates' offsets differed by EXACTLY 0 on every duplicated triangle of every probed face,
// so the winner was slice order and the doc's claim about rounding was false.
//
// Measured on the near-pinch face: duplicate welded triangles 1 -> 0, and the unpaired edges carrying
// THREE triangles 4 -> 1.
//
// Both coverings use it — the chart-driven mesher and the periodic B-spline band — through
// coverVertices.keepCanonical, so the premise is fixed once (#3518 review I6).
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
// the metric multiply by su both round. The cure is combinatorial and belongs to the constrained
// triangulation — fixed-point coordinates, or an in-circle tie broken on vertex IDENTITY rather than
// on the rounded coordinates — and it is #3542. It is emphatically NOT to move the parameters onto a
// grid so the shift comes out exact: that perturbs the geometry the triangulation reasons about,
// against the ground rule that robustness lives in the combinatorial layer, and it cannot be made
// exact anyway while the triangulation runs on u*su, because floating multiplication does not
// distribute over the shift.
//
// Until then the near-pinch band keeps the unrolled arm (curved_trim_recognize.go's corridor gate) and
// the eight join bodies keep the tessellate.cap-saturated their 64-cell grid earns.

// keepOneReplicaEach keeps exactly ONE covering triangle per 3D triangle.
func (c *coverVertices) keepOneReplicaEach(tris [][3]int, keep []bool) {
	grid := weldGrid([][]math.Point3{c.pos})
	best := map[[3][3]int64]int{}
	for i, t := range tris {
		if !keep[i] {
			continue
		}
		k := weldedTriangleKey(c.pos, t, grid)
		if j, seen := best[k]; !seen || c.translateOf(t) < c.translateOf(tris[j]) {
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

// translateOf is which whole-period translate a triangle is: the sum of its vertices' shift indices.
// Replicas of one triangle differ by three times the shift step, so the sum orders them strictly and
// no two translates of the same triangle can tie.
func (c *coverVertices) translateOf(t [3]int) int {
	return c.at[t[0]] + c.at[t[1]] + c.at[t[2]]
}

// weldedTriangleKey is a covering triangle's identity as a 3D triangle: its three welded vertices in a
// fixed order, so period replicas — which share their 3D points exactly — key the same.
func weldedTriangleKey(pos []math.Point3, t [3]int, grid float64) [3][3]int64 {
	k := [3][3]int64{quantizePoint(pos[t[0]], grid), quantizePoint(pos[t[1]], grid), quantizePoint(pos[t[2]], grid)}
	if weldKeyLess(k[1], k[0]) {
		k[0], k[1] = k[1], k[0]
	}
	if weldKeyLess(k[2], k[1]) {
		k[1], k[2] = k[2], k[1]
	}
	if weldKeyLess(k[1], k[0]) {
		k[0], k[1] = k[1], k[0]
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
