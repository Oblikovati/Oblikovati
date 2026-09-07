// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The winding certificate: does every loop of a face walk with the face's material on its LEFT, seen
// along the face's outward normal? That is the contract every producer signs — an outer loop
// counter-clockwise about the outward normal, a hole clockwise — and it is what the tessellator, the
// analytic integrator and the boolean's own classifiers read the face's region from.
//
// Validate's per-edge test cannot see it. A shell whose every edge is used twice in opposite directions
// is consistently oriented as a use-graph, and it stays so when one face is wound against its normal
// together with the neighbour that shares its edge: the two errors cancel pairwise. The torus tangent
// cut shipped exactly that — one lobe of the figure-eight lid inverted with the torus edge it borders —
// as a valid solid, and only the tessellated volume showed it. This is the post-condition on the
// EMISSION the stage-4 clip left named (ADR-0061).
//
// It is ONE question asked once per loop, and the face's own trim answers it: step off the boundary to
// the side the winding claims the material is on, and ask whether that point is in the face. Reading
// the loop's shoelace, its nesting among the other loops, or its direction against the chart's nearest
// contour segment are three different questions, all of which this replaces — and the last of them was
// wrong, because a chart's contour carries the artificial SEAM as well as the real boundary, and a
// crossing band's rim sample sits exactly on it. The trim knows the difference; the point list does not.

// FaceWindingConsistent reports whether every loop of f winds with the face's material on its left
// about the outward normal. certain is false when no loop offered a decidable station — the step to
// either side of the boundary landed on the same side of the trim, so nothing was measured; ok is then
// true by default and means nothing.
//
// Example: if ok, certain := brep.FaceWindingConsistent(f); certain && !ok { /* an inverted face */ }
func FaceWindingConsistent(f *topo.Face) (ok, certain bool) {
	cf := curvedFaceOf(f)
	if len(cf.loops) == 0 {
		return true, true // a whole surface has no winding to get wrong
	}
	uPer, vPer := surfacePeriodic(cf.surface)
	trim := developFaceTrim(cf)
	for _, loop := range cf.loops {
		ring := loopToUV(cf.surface, loop, uPer, vPer)
		wound, decided := ringWindsWithMaterialOnItsLeft(cf, trim, ring)
		if !decided {
			continue
		}
		certain = true
		if !wound {
			return false, true
		}
	}
	return true, certain
}

// ringWindsWithMaterialOnItsLeft walks a boundary ring and, at each station, steps a short way to the
// side its direction claims the material is on and to the opposite side. A station where the two land
// on DIFFERENT sides of the trim has resolved the boundary and votes; one where they agree measured
// nothing — the step was too long for a thin neck, too short for the sampling's own noise, or the
// station sits at a pole — and is passed over. decided is false when no station voted.
//
// Stations are walked rather than one taken, because a single sample can land on a self-touch, a corner
// or a pole, and a boundary loop is entitled to have those.
func ringWindsWithMaterialOnItsLeft(cf curvedFace, trim *faceTrimUV, ring []math.Point2) (wound, decided bool) {
	sense := outwardSenseInUV(cf, ring[0])
	step := windingProbeStep(ring)
	if step <= 0 {
		return false, false
	}
	for i := range ring {
		at, dir := ring[i], ring[i].VectorTo(ring[(i+1)%len(ring)])
		left, okDir := unitLeftOf(dir, sense)
		if !okDir {
			continue
		}
		inLeft := trim.contains(cf.surface.PointAt(float64(at.X)+float64(left.X)*step, float64(at.Y)+float64(left.Y)*step))
		inRight := trim.contains(cf.surface.PointAt(float64(at.X)-float64(left.X)*step, float64(at.Y)-float64(left.Y)*step))
		if inLeft == inRight {
			continue // the step resolved no boundary here
		}
		return inLeft, true
	}
	return false, false
}

// unitLeftOf is the unit (u,v) direction the face's material lies in if the ring is wound correctly:
// the left of the travel direction, turned round on a chart whose handedness opposes the outward
// normal. ok=false for a degenerate step, which names no direction.
func unitLeftOf(dir math.Vector2, sense float64) (math.Vector2, bool) {
	l := stdmath.Hypot(float64(dir.X), float64(dir.Y))
	if l == 0 {
		return math.Vector2{}, false
	}
	return math.V2(math.Scalar(-float64(dir.Y)/l*sense), math.Scalar(float64(dir.X)/l*sense)), true
}

// windingProbeStep is how far off the boundary the probe steps, in the chart's own parameters: a
// fraction of the ring's own median sampling step, so it is small against the boundary's curvature and
// large against the sampling's noise, at any model scale and on any parameterisation.
func windingProbeStep(ring []math.Point2) float64 {
	steps := make([]float64, 0, len(ring))
	for i := range ring {
		d := stdmath.Hypot(float64(ring[(i+1)%len(ring)].X-ring[i].X), float64(ring[(i+1)%len(ring)].Y-ring[i].Y))
		if d > 0 {
			steps = append(steps, d)
		}
	}
	if len(steps) == 0 {
		return 0
	}
	return windingProbeFraction * medianOf(steps)
}

// windingProbeFraction is the probe step as a fraction of the boundary's own sampling step. A quarter
// keeps the probe well inside the face between two samples while clearing the chord's own sagitta.
const windingProbeFraction = 0.25 // tol:parametric — probe offset as a fraction of the sampling step

// medianOf returns the median of a non-empty slice, leaving the caller's slice unsorted.
func medianOf(xs []float64) float64 {
	c := append([]float64(nil), xs...)
	sort.Float64s(c)
	return c[len(c)/2]
}

// outwardSenseInUV is the sign a loop's (u,v) travel must turn to keep the face's material on its left:
// the chart's handedness (∂P/∂u × ∂P/∂v against the surface's own normal) turned round for a reversed
// face, whose outward normal is the surface normal's opposite.
func outwardSenseInUV(cf curvedFace, at math.Point2) float64 {
	du, dv := cf.surface.DerivativesAt(float64(at.X), float64(at.Y))
	sense := 1.0
	if float64(du.Cross(dv).Dot(cf.surface.NormalAt(float64(at.X), float64(at.Y)))) < 0 {
		sense = -1
	}
	if cf.reversed {
		sense = -sense
	}
	return sense
}
