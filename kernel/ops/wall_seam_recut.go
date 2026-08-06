// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Re-cutting a drilled wall's seam clear of its holes (Oblikovati/Oblikovati#2038).
//
// holedConicWallMesh unrolls a full-wrap developable wall into the (u,v) branch the wall's OWN seam
// defines, then shifts each lens hole by whole periods into that branch. A lens that STRADDLES the
// seam fits no branch at all, so holesIntoBranch declined and the face fell through to the flat
// best-fit-plane CDT — which collapses a full wrap and covers only half of it. The B-rep stays valid
// and ops.Validate stays clean, so the damage surfaces only in the integrated quantities: a Ø3 mm
// bore through a Ø100 × 4 mm disk reported 7.1 cm³ against an analytic 30.7 (−77%), silently.
//
// Whether a lens straddles is an accident of where the operand cylinder's angle-0 happens to point —
// brep.SolidCylinder takes it from axisFrame (+Y for a +Z axis), an extruded circle from its sketch
// +X (extrude_analytic.go) — so the "the builder places the seam clear of its holes" precondition
// holesIntoBranch documented cannot be relied on. The seam of a periodic wall is arbitrary, so cut it
// SOMEWHERE ELSE: split the loop back into the two rims it bridges and re-bridge them at the widest
// lens-free angle, through the same bridgeRimsAtSeam the two-rim band uses (two_rim_holed_band.go).

// wallBranch is a wall's unrolled outer loop: the 3D loop, its (u,v) image, and the u-range it spans.
type wallBranch struct {
	loop       []math.Point3
	uv         []math.Point2
	umin, umax float64
}

// unrollWall unrolls a wrapping wall loop into its own branch. ok=false unless the loop wraps a full
// period.
func unrollWall(s geom.Surface, loop []math.Point3) (wallBranch, bool) {
	uv, umin, umax, ok := wrappedWallUV(s, loop)
	if !ok {
		return wallBranch{}, false
	}
	return wallBranch{loop: loop, uv: uv, umin: umin, umax: umax}, true
}

// rebridgedWallLoop re-cuts outer3D's seam into the widest gap clear of the lens holes, returning the
// re-bridged wrapping loop. ok=false when the loop is not a two-rim seam-bridged wall, or when a hole
// itself wraps the period — that is a rim mis-demoted to a hole, twoRimHoledBandMesh's case, and no
// choice of seam angle helps.
func rebridgedWallLoop(s geom.Surface, outer3D []math.Point3, holes3D [][]math.Point3) ([]math.Point3, bool) {
	rims, lenses := splitWrappingHoles(s, holes3D)
	if len(rims) > 0 || len(lenses) == 0 {
		return nil, false
	}
	top, bot, ok := splitSeamBridgedRims(s, outer3D)
	if !ok {
		return nil, false
	}
	return bridgeRimsAtSeam(s, top, bot, lenses)
}

// splitSeamBridgedRims splits a wall's seam-bridged outer loop back into its two rim rings. The loop
// runs one rim from the seam all the way around, crosses to the other rim at the branch edge, and runs
// back — so the points sitting at the lift's two extremes ARE the seam, and the two runs between them
// are the rims. top is the ring with the greater mean v. ok=false when the loop is not that shape.
func splitSeamBridgedRims(s geom.Surface, outer3D []math.Point3) (top, bot []math.Point3, ok bool) {
	onSeam, ok := seamEndMask(s, outer3D)
	if !ok {
		return nil, nil, false
	}
	a, b, ok := ringsBetweenSeams(outer3D, onSeam)
	if !ok {
		return nil, nil, false
	}
	if meanVParam(s, a) < meanVParam(s, b) {
		a, b = b, a
	}
	return a, b, true
}

// seamEndMask marks the loop points sitting at one of the unrolled lift's two extremes — the wall's
// seam, which the loop traverses once at each end of the branch. ok=false unless the loop wraps a
// full period.
func seamEndMask(s geom.Surface, outer3D []math.Point3) ([]bool, bool) {
	us := make([]float64, len(outer3D))
	for i, p := range outer3D {
		us[i], _ = s.ParamAt(p)
	}
	cu := cumulativeUnwrap(us)
	umin, umax := minMax(cu)
	if umax-umin < 2*stdmath.Pi-0.5 {
		return nil, false // not a full wrap: nothing to re-cut
	}
	mask := make([]bool, len(cu))
	for i, u := range cu {
		mask[i] = u-umin <= seamAngularTol || umax-u <= seamAngularTol
	}
	return mask, true
}

// ringsBetweenSeams walks the cyclic loop and splits it at the seam into the two rim rings. A seam
// run's FIRST point closes the ring being walked and its LAST point opens the next — those two are
// the rim vertices the seam joins — and any point between them is seam interior, dropped because the
// re-bridge lays down its own. ok=false unless the loop has exactly the two seam runs a bridged wall
// has.
func ringsBetweenSeams(pts []math.Point3, onSeam []bool) (a, b []math.Point3, ok bool) {
	start := indexOfFalse(onSeam)
	if start < 0 || countSeamRuns(onSeam) != 2 {
		return nil, nil, false
	}
	var rings [2][]math.Point3
	cur, n := 0, len(pts)
	for k := 0; k < n; {
		i := (start + k) % n
		if !onSeam[i] {
			rings[cur] = append(rings[cur], pts[i])
			k++
			continue
		}
		run := seamRunLength(onSeam, i)
		rings[cur] = append(rings[cur], pts[i]) // closes this rim
		cur = 1 - cur
		rings[cur] = append(rings[cur], pts[(i+run-1)%n]) // opens the other
		k += run
	}
	a, b = dropCyclicDuplicates(rings[0]), dropCyclicDuplicates(rings[1])
	return a, b, len(a) >= 3 && len(b) >= 3
}

// indexOfFalse returns the first index whose mask entry is false, or −1 when every entry is true.
func indexOfFalse(mask []bool) int {
	for i, m := range mask {
		if !m {
			return i
		}
	}
	return -1
}

// countSeamRuns counts the maximal cyclic runs of true in mask — a seam-bridged wall has exactly two,
// one at each end of the branch.
func countSeamRuns(mask []bool) int {
	n, runs := len(mask), 0
	for i, m := range mask {
		if m && !mask[(i-1+n)%n] {
			runs++
		}
	}
	return runs
}

// seamRunLength returns the length of the maximal cyclic run of true in mask starting at i.
func seamRunLength(mask []bool, i int) int {
	n := len(mask)
	for k := 1; k <= n; k++ {
		if !mask[(i+k)%n] {
			return k
		}
	}
	return n
}

// dropCyclicDuplicates removes the repeat of the seam vertex: each rim is walked from the seam all
// the way around and back to it, so the vertex the seam is cut at appears twice — as the last two
// points, or as the last and the first.
func dropCyclicDuplicates(ring []math.Point3) []math.Point3 {
	tol := ringCoincidenceTol(ring)
	out := make([]math.Point3, 0, len(ring))
	for _, p := range ring {
		if len(out) > 0 && float64(p.DistanceTo(out[len(out)-1])) <= tol {
			continue
		}
		out = append(out, p)
	}
	for len(out) > 1 && float64(out[len(out)-1].DistanceTo(out[0])) <= tol {
		out = out[:len(out)-1]
	}
	return out
}

// ringCoincidenceTol is the distance below which two ring points are the same place: a small fraction
// of the ring's own mean chord, so the test scales with the model instead of fixing an absolute
// epsilon (ADR-0042).
func ringCoincidenceTol(ring []math.Point3) float64 {
	if len(ring) < 2 {
		return 0
	}
	var sum float64
	for i := 1; i < len(ring); i++ {
		sum += float64(ring[i-1].DistanceTo(ring[i]))
	}
	return 1e-6 * sum / float64(len(ring)-1)
}

// meanVParam is a ring's mean non-periodic parameter — which of a wall's two rims is the "top".
func meanVParam(s geom.Surface, ring []math.Point3) float64 {
	var sum float64
	for _, p := range ring {
		sum += vParam(s, p)
	}
	return sum / float64(len(ring))
}
