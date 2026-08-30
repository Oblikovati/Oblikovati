// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// nearestCrossingInside classifies p by the classical ray-crossing solid classifier: cast a ray from p and read
// the side from the NEAREST boundary crossing alone — an interior point's ray EXITS through its first face
// (its outward normal points the same way as the ray, dir·n > 0), an exterior point's ray ENTERS (dir·n <
// 0). Only the first hit and its orientation decide, so it is crisp at ANY distance — a probe a small
// offset off a face classifies decisively, where the winding field reads ≈½ — and robust to a seam gap
// past the first hit. The outward normal is the orientation-normalized one (fluxFace.sign), so it is right
// even where the stored Reversed flag is not.
//
// A ray whose first hit grazes a trim edge or runs tangent is ambiguous, so the direction is re-selected
// (a classification, not a geometry nudge). ok is false only when every candidate direction grazes — the
// rare degenerate case the winding-number fallback resolves.
func (q *fluxQuery) nearestCrossingInside(p math.Point3, box math.Box) (inside, ok bool) {
	tMax := 2 * float64(box.Diagonal().Length())
	tol := geom.ResolutionForBox(box).Plane()
	for _, d := range rayDirections {
		if in, clean := q.firstHitSide(p, d, tMax, tol); clean {
			return in, true
		}
	}
	return false, false
}

// firstHitSide casts one ray and returns the inside/outside verdict from its nearest in-trim crossing.
// clean is false when any crossing along the ray grazes a boundary (ambiguous), so the caller re-selects
// the direction; a ray with no crossing is a clean "outside".
func (q *fluxQuery) firstHitSide(p math.Point3, dir [3]float64, tMax, tol float64) (inside, clean bool) {
	ray, err := geom.NewLine(p, math.V3(dir[0], dir[1], dir[2]))
	if err != nil {
		return false, false
	}
	best := stdmath.Inf(1)
	var bestFace *fluxFace
	var bestU, bestV float64
	for i := range q.faces {
		f := &q.faces[i]
		band := stdmath.Max(tol, faceBoundaryBand(f.cf))
		for _, hit := range geom.RaySurfaceHits(f.cf.surface, ray, tMax) {
			if hit.T <= tol {
				continue // a hit at p itself; p on the surface is the caller's ON case
			}
			if rayGrazes(f.cf, ray, hit, band) {
				return false, false // ambiguous crossing → re-select the direction
			}
			if hit.T < best && pointInTrimUV(f.cf, hit.Point) {
				best, bestFace, bestU, bestV = hit.T, f, hit.U, hit.V
			}
		}
	}
	if bestFace == nil {
		return false, true // the ray leaves the body without crossing → outside
	}
	outward := bestFace.cf.surface.NormalAt(bestU, bestV).Scale(bestFace.sign)
	return float64(ray.Dir.AsVector().Dot(outward)) > 0, true
}
