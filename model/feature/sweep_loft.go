// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/math"
)

// centroidOf returns the average of a point set.
func centroidOf(pts []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range pts {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(pts))
	return math.P3(sx/n, sy/n, sz/n)
}

// resampleLoop returns n points around the closed polygon that PRESERVE its original vertices
// (corners) and fill the remaining budget with collinear points along the edges (proportional to
// edge length), so an elongated polygon keeps its exact shape and area. A plain arc-length
// resample lands samples mid-edge and cuts the corners off any non-square polygon: a 16×2 mm
// rectangle resampled to 4 points became an 18 mm² quad (0.5625× its 32 mm² area), which then
// under-filled every swept solid's volume by the same factor (Oblikovati loft volume deficit,
// found 2026-06-15). Because interior points are collinear, the shape (and area) is unchanged;
// because every loop is resampled to the same n, sections of differing vertex counts still blend
// point-for-point. maxLoopCount guarantees n >= len(poly), so the densest loop (n == m) is
// returned verbatim. A degenerate loop (a point section) expands to n copies of its point — an
// apex the mesher welds to one vertex, so a loft to a point skins a cone/dome.
func resampleLoop(poly []math.Point3, n int) []math.Point3 {
	m := len(poly)
	segLen, total := loopSegmentLengths(poly)
	if m == 0 || total == 0 { // a point/apex section: n copies of its point
		out := make([]math.Point3, n)
		for k := range out {
			if m > 0 {
				out[k] = poly[0]
			}
		}
		return out
	}
	if n <= m { // the densest loop (n == m): keep its corners exactly (n < m can't occur, see above)
		return append([]math.Point3(nil), poly...)
	}
	interior := edgeInteriorCounts(segLen, total, n-m)
	out := make([]math.Point3, 0, n)
	for i := range m {
		a, b := poly[i], poly[(i+1)%m]
		out = append(out, a) // the corner — always preserved
		for t := 1; t <= interior[i]; t++ {
			f := math.Scalar(float64(t) / float64(interior[i]+1))
			out = append(out, a.TranslateBy(a.VectorTo(b).Scale(f)))
		}
	}
	return out
}

// edgeInteriorCounts apportions `extra` interior points across edges of the given lengths,
// proportional to length (largest-remainder rounding so the counts sum to exactly extra) — longer
// edges get more in-between samples. Used by resampleLoop to upsample a loop without moving its
// corners.
func edgeInteriorCounts(segLen []float64, total float64, extra int) []int {
	counts := make([]int, len(segLen))
	if extra <= 0 || total <= 0 {
		return counts
	}
	rema := make([]float64, len(segLen))
	assigned := 0
	for i, l := range segLen {
		ideal := float64(extra) * l / total
		counts[i] = int(ideal) // floor (ideal >= 0)
		rema[i] = ideal - float64(counts[i])
		assigned += counts[i]
	}
	for ; assigned < extra; assigned++ { // hand the leftover to the biggest remainders
		best := 0
		for i := 1; i < len(rema); i++ {
			if rema[i] > rema[best] {
				best = i
			}
		}
		counts[best]++
		rema[best] = -1 // don't pick the same edge twice
	}
	return counts
}

// loopSegmentLengths returns each edge length of the closed polygon and their total.
func loopSegmentLengths(poly []math.Point3) ([]float64, float64) {
	m := len(poly)
	segLen := make([]float64, m)
	total := 0.0
	for i := range m {
		segLen[i] = poly[i].DistanceTo(poly[(i+1)%m])
		total += segLen[i]
	}
	return segLen, total
}
