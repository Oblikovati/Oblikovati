// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
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
// It is an INDEPENDENT verifier, on purpose. senseFromLoopWinding makes the stored sense definitionally
// the output of loopHandedness, so a gate that shared that reader could never catch a wrong sense — it
// would agree with it by construction. The two therefore differ, deliberately, in exactly two places,
// and in no others:
//
//   - the membership tester: this asks faceTrimUV.contains, which inverts a 3-space point through
//     ParamAt against the face's DEVELOPED boundary; loopHandedness asks trimRegion.contains against
//     the chart, in (u, v). Two ways of asking the face where its material is, so a fault in either
//     shows up as a disagreement rather than as a shared wrong answer.
//   - the decision rule: this returns the FIRST station that resolves the boundary, because a gate
//     wants the first witness of an inverted face; loopHandedness takes a majority over up to 32
//     stations, because a sense has to be right rather than merely witnessed.
//
// The STEP is NOT one of those places. "A quarter of this segment, to its left" is one predicate and
// the ground rules give it one implementation ([quarterArcLeftOf]); computing it twice is how the two
// readers came to disagree on a chart whose axes are not the same unit (Oblikovati/Oblikovati#3512).
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
//
// The STEP is [quarterArcLeftOf], the one reader of "a quarter of this segment, to its left" the kernel
// has — the same one the sense itself is derived from. It used to be a chart-relative median of the
// ring's own (u, v) sampling, which is a second answer to a question already decided elsewhere and
// carries the very defect that reader was fixed for: a quarter TURN in (u, v) is "left" only where the
// chart is isotropic (Oblikovati/Oblikovati#3512). What this verifier keeps of its own is stated above.
func ringWindsWithMaterialOnItsLeft(cf curvedFace, trim *faceTrimUV, ring []math.Point2) (wound, decided bool) {
	sense := math.Scalar(outwardSenseInUV(cf, ring[0]))
	for i := range ring {
		at, dir := ring[i], ring[i].VectorTo(ring[(i+1)%len(ring)])
		left, ok := quarterArcLeftOf(cf.surface, at, dir)
		if !ok {
			continue
		}
		left = left.Scale(sense) // the side the STORED outward normal claims, not the chart's own left
		inLeft := trim.contains(surfacePointOffset(cf.surface, at, left))
		inRight := trim.contains(surfacePointOffset(cf.surface, at, left.Scale(-1)))
		if inLeft == inRight {
			continue // the step resolved no boundary here
		}
		return inLeft, true
	}
	return false, false
}

// surfacePointOffset is the 3-space point a (u, v) offset from at lands on.
func surfacePointOffset(s geom.Surface, at math.Point2, off math.Vector2) math.Point3 {
	q := at.TranslateBy(off)
	return s.PointAt(float64(q.X), float64(q.Y))
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
