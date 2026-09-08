// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import "oblikovati.org/math"

// The rim-only ear (ADR-0061, final fix wave, finding 1).
//
// The boundary clearance culls every interior grid node within a fraction of a chord of the rim. Where
// the rim turns a CORNER whose two chords are longer than the corner is wide, that band is the whole
// corner, and the constrained triangulation has nothing to close it with but the two boundary vertices
// either side and the corner itself: a triangle whose three vertices are all on the rim. Such a triangle
// carries no surface point of its own — its chord plane is whatever plane its three rim points lie in —
// and on the figure-eight torus band that plane is the LID's: measured at DefaultQuality on the torus
// R=5 r=2 cut by y=3, the material corner at the pinch is ~102° wide against two chords of 1.23 mm, and
// the mesh emitted (P_lo, pinch, P_hi) with all three vertices on y=3, the lid's own tip triangle with
// the opposite normal. The body read two free edges of degree FOUR — a doubled surface, not a crack.
//
// So a rim-only triangle is never emitted. Each one is split at the surface point under its own (u,v)
// centroid — a point the region already certified material, evaluated on the surface, so nothing is
// moved — and the ONE constrained triangulation is run again over the enlarged point set. No threshold
// decides it: a rim-only triangle that happened to lie flat on the surface would be split too, and
// costs one point.

// chartRimEarRounds bounds the re-triangulations. Every round adds one point inside each ear, so it
// terminates on its own; the bound is what makes "still an ear after that" a decline the router reports
// rather than a loop. Eight is the same bound the fold repair carries (validate.RepairFolds).
const chartRimEarRounds = 8

// keptWithoutRimEars is the chart's kept triangulation with every rim-only ear split. ok=false when
// the bound is spent with an ear still standing; the caller declines the face and the router reports
// the discarded trim.
func (b *chartCover) keptWithoutRimEars(loops [][]int) ([][3]int, bool) {
	for range chartRimEarRounds {
		kept := b.keepChartTriangles(constrainedTriangulationAll(b.xy, loops))
		ears := b.rimEars(kept)
		if len(ears) == 0 {
			return kept, true
		}
		for _, t := range ears {
			b.addEarCentre(t)
		}
	}
	return nil, false
}

// rimEars is every kept triangle whose three vertices are boundary vertices AND which the weld will
// emit. A covering triangle two of whose rim vertices are one 3D point — the figure-eight's single loop
// passes its pinch twice — collapses at the weld and is never in the mesh, so it is no ear.
func (b *chartCover) rimEars(kept [][3]int) [][3]int {
	grid := weldGrid([][]math.Point3{b.pos})
	var ears [][3]int
	for _, t := range kept {
		if b.isRimEar(t, grid) {
			ears = append(ears, t)
		}
	}
	return ears
}

// isRimEar reports whether the triangle is rim-only and survives the weld. The chains are laid into
// the covering before any interior node, so an index below b.rim IS a boundary vertex.
func (b *chartCover) isRimEar(t [3]int, grid float64) bool {
	if t[0] >= b.rim || t[1] >= b.rim || t[2] >= b.rim {
		return false
	}
	p, q, r := quantizePoint(b.pos[t[0]], grid), quantizePoint(b.pos[t[1]], grid), quantizePoint(b.pos[t[2]], grid)
	return p != q && q != r && p != r
}

// addEarCentre adds the surface point under the ear's (u,v) centroid, replicated like any interior
// node. The centroid is inside a triangle the region kept, so it is material and inside the window.
func (b *chartCover) addEarCentre(t [3]int) {
	u, v := b.centroid(t)
	fu, fv := b.r.fold(u, v)
	b.addReplicas(b.s.PointAt(fu, fv), u, v)
}
