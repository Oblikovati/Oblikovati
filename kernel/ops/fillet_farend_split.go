// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The TRIM-SIDE two-piece split — the half of complex/D8's blocker that `chainRetrimLoop` (f391c182)
// could not remove on its own (selfcross-trim-report.md §5.2, chain-retrim-report.md §5.3).
//
// THE DEFECT. slideOntoWall slides every station of a band's terminal section onto the stop wall's
// IMPLICIT surface, which extends past the stop FACE's own trim. complex/D8's r=30 band stops on a
// radius-24 quarter round spanning u ∈ [−π/2, 0]; its last five stations land at u > 0, i.e. 6.064 of
// developed length past that round's u=0 ruling, on the flat y = 35.09378 wall next door. One trim curve
// is then made to bound two different faces, which is why the round's developed boundary self-crosses
// (pinching off 1.2111 of 3307.1168) and why its tb tangent point sits 0.762 inside the top face.
//
// THE SPLIT. Each station is slid onto whichever face actually CARRIES its landing, and the section is
// cut at the parameter where the landing crosses the shared edge — solved by bisection on the stop face's
// own box bound, so the junction is the exact triple point (band ∩ round ∩ wall). D8's is
// (223.39418029785, 35.093784332275, 9.393876913), against the closed form z = −20 + √864.
//
// WHAT IT DELIBERATELY DOES NOT DO. One entry, one exit: exactly one contiguous run of off-face stations,
// touching one end of the section. A section that leaves and re-enters, or leaves at both ends, is
// DECLINED — today's single-face trim is kept, byte-for-byte, which is the do-no-harm floor the rest of
// the retrim layer already uses.

// endPiece is one piece of a terminal trim curve, together with the face that carries it. A corner whose
// section stays on its stop face has no pieces at all and keeps corner.endCurve.
type endPiece struct {
	face *topo.Face
	seg  endSeg
}

// farEndJunctionSteps is how many bisections resolve the section parameter at which the landing crosses
// the stop face's own bound. 60 halvings take the initial 1/32 bracket to 3e-20 — below double
// precision's own resolution of the parameter, so the junction is exact to the last bit the bracket can
// express.
const farEndJunctionSteps = 60

// splitTerminalSection resolves c's terminal section into the chain of pieces the faces it actually
// crosses carry, ordered ta → tb. ok=false is the honest decline: for a section that stays on its own
// stop face (where today's single-piece trim is already right), and for every configuration this split
// cannot resolve exactly.
func splitTerminalSection(c corner, in cornerInputs, maxSlide float64) ([]endPiece, bool) {
	arc, err := geom.Arc3dByThreePoints(c.ta, c.mid, c.tb)
	if err != nil {
		return nil, false
	}
	box, ok := faceParamBox(c.endFace, in.weld)
	if !ok {
		return nil, false
	}
	slide := terminalSlide(c, in, maxSlide)
	sides, ok := stationExitSides(arc, c.endFace, box, slide, in.weld)
	if !ok {
		return nil, false
	}
	return splitAtExit(c, in, arc, box, slide, sides)
}

// stationExitSides classifies every station of the section: sideInside when its landing is on the stop
// face's own parameter box, else the bound it crossed. It declines when any station has no landing at all.
func stationExitSides(arc geom.Arc3d, stop *topo.Face, box paramBox, slide axialSlide, tol float64) ([]boxSide, bool) {
	s := stop.Geometry()
	out := make([]boxSide, 0, farEndTrimStations+1)
	for i := 0; i <= farEndTrimStations; i++ {
		q, ok := stationLanding(arc, float64(i)/float64(farEndTrimStations), s, slide)
		if !ok {
			return nil, false
		}
		out = append(out, boxSideOfPoint(s, box, q, tol))
	}
	return out, true
}

// stationLanding is the section point at parameter t, slid onto wall.
func stationLanding(arc geom.Arc3d, t float64, wall geom.Surface, slide axialSlide) (math.Point3, bool) {
	lo, hi := arc.Domain()
	return slideOntoWall(arc.PointAt(lo+t*(hi-lo)), wall, slide)
}

// splitAtExit builds the two pieces once the off-face run is known: the near piece on the stop face, the
// far piece on the neighbour across the bound the run crossed, meeting at the bisected junction.
func splitAtExit(c corner, in cornerInputs, arc geom.Arc3d, box paramBox, slide axialSlide, sides []boxSide) ([]endPiece, bool) {
	side, offAtTail, ok := singleOffRun(sides)
	if !ok {
		return nil, false
	}
	nb, ok := faceNeighbourOnBoxSide(c.endFace, box, side, in.weld)
	if !ok || !extendableWall(nb.Geometry()) {
		return nil, false
	}
	tj, ok := junctionParam(arc, c.endFace, box, slide, side, sides)
	if !ok {
		return nil, false
	}
	return assemblePieces(c, in, arc, slide, nb, tj, offAtTail)
}

// singleOffRun reports the bound the off-face stations crossed and whether that run sits at the section's
// TAIL (tb end) rather than its head. It declines unless the off-face stations form exactly one
// contiguous run, all crossing the SAME bound, touching exactly one end of the section, and leaving at
// least two on-face stations behind.
func singleOffRun(sides []boxSide) (boxSide, bool, bool) {
	side, lo, hi, ok := offRunExtent(sides)
	if !ok {
		return sideInside, false, false
	}
	last := len(sides) - 1
	touchesOneEnd := (lo == 0) != (hi == last)
	if !touchesOneEnd || len(sides)-(hi-lo+1) < 2 {
		return sideInside, false, false // floating in the middle, both ends, or too little face left
	}
	return side, hi == last, true
}

// offRunExtent is the index range of the single contiguous run of off-face stations and the bound they
// all cross, or ok=false when there is none, when there are two runs, or when two bounds were crossed.
func offRunExtent(sides []boxSide) (boxSide, int, int, bool) {
	lo, hi, side := -1, -1, sideInside
	for i, s := range sides {
		if s == sideInside {
			continue
		}
		if lo < 0 {
			lo, side = i, s
		}
		if s != side || (hi >= 0 && i != hi+1) {
			return sideInside, 0, 0, false
		}
		hi = i
	}
	return side, lo, hi, lo >= 0
}

// junctionParam bisects the section parameter at which the landing's own box coordinate reaches the
// crossed bound — the exact point where the trim leaves the stop face through its shared edge. The
// bracket is the last on-face station and the first off-face one, so the sign change is guaranteed.
func junctionParam(arc geom.Arc3d, stop *topo.Face, box paramBox, slide axialSlide, side boxSide, sides []boxSide) (float64, bool) {
	a, b, ok := crossingBracket(sides)
	if !ok {
		return 0, false
	}
	s, bound := stop.Geometry(), boxSideBound(box, side)
	out := func(t float64) (float64, bool) {
		q, ok := stationLanding(arc, t, s, slide)
		if !ok {
			return 0, false
		}
		u, v := boxParamAt(s, box, q)
		return outwardExcess(side, boxSideParam(side, u, v), bound), true
	}
	return bisectOutwardZero(out, a, b)
}

// crossingBracket returns the two adjacent station parameters that straddle the on/off transition.
func crossingBracket(sides []boxSide) (float64, float64, bool) {
	n := float64(farEndTrimStations)
	for i := 0; i+1 < len(sides); i++ {
		if (sides[i] == sideInside) != (sides[i+1] == sideInside) {
			return float64(i) / n, float64(i+1) / n, true
		}
	}
	return 0, 0, false
}

// outwardExcess is how far past the bound the coordinate lies, signed so it is positive outside — the
// function whose zero is the junction, on either the lo or the hi side of the box.
func outwardExcess(side boxSide, x, bound float64) float64 {
	if side == sideULo || side == sideVLo {
		return bound - x
	}
	return x - bound
}

// bisectOutwardZero halves the bracket [a,b] onto the zero of f, which is negative at the on-face end
// and positive at the off-face one (in either order).
func bisectOutwardZero(f func(float64) (float64, bool), a, b float64) (float64, bool) {
	fa, ok := f(a)
	if !ok {
		return 0, false
	}
	for i := 0; i < farEndJunctionSteps; i++ {
		m := (a + b) / 2
		fm, ok := f(m)
		if !ok {
			return 0, false
		}
		if (fm < 0) == (fa < 0) {
			a, fa = m, fm
			continue
		}
		b = m
	}
	return (a + b) / 2, true
}

// assemblePieces lands the two station ranges on their own faces and returns them ordered ta → tb.
func assemblePieces(c corner, in cornerInputs, arc geom.Arc3d, slide axialSlide, nb *topo.Face,
	tj float64, offAtTail bool) ([]endPiece, bool) {
	nbSlide := axialSlide{axis: in.axis, reach: slide.reach, tol: in.weld,
		side: stopFaceAxialSide(nb, c.vertex.Point(), in.axis, in.weld)}
	stopLo, stopHi, nbLo, nbHi := tj, 1.0, 0.0, tj
	if offAtTail {
		stopLo, stopHi, nbLo, nbHi = 0, tj, tj, 1
	}
	near, ok := landedPiece(arc, stopLo, stopHi, c.endFace, slide, in.weld)
	if !ok {
		return nil, false
	}
	far, ok := landedPiece(arc, nbLo, nbHi, nb, nbSlide, in.weld)
	if !ok {
		return nil, false
	}
	chain := chainedEndPieces(near, far, offAtTail)
	// The junction is the triple point: the two faces share the edge the section crosses, so the two
	// pieces must ALREADY meet there. A gap would be bridged silently by the two independent splices and
	// crack the shell, so it is a decline, never a weld.
	if float64(chain[0].seg.to.DistanceTo(chain[1].seg.from)) > in.weld {
		return nil, false
	}
	return chain, true
}

// chainedEndPieces chains the stop-face piece and the neighbour piece head-to-tail along ta → tb.
func chainedEndPieces(near, far endPiece, offAtTail bool) []endPiece {
	if offAtTail {
		return []endPiece{near, far}
	}
	return []endPiece{far, near}
}

// landedPiece slides the section's [t0,t1] sub-range onto face f's wall and returns it as one boundary
// segment on f, declining when any station misses the wall or lands off f's own parameter box.
func landedPiece(arc geom.Arc3d, t0, t1 float64, f *topo.Face, slide axialSlide, tol float64) (endPiece, bool) {
	box, ok := faceParamBox(f, tol)
	if !ok {
		return endPiece{}, false
	}
	pts, ok := landedStations(arc, t0, t1, f.Geometry(), box, slide, tol)
	if !ok {
		return endPiece{}, false
	}
	seg, ok := pieceSeg(pts, tol)
	return endPiece{face: f, seg: seg}, ok
}

// landedStations slides farEndTrimStations+1 samples of the section's [t0,t1] sub-range onto the wall,
// declining as soon as one misses it or lands outside the face's own box.
func landedStations(arc geom.Arc3d, t0, t1 float64, s geom.Surface, box paramBox, slide axialSlide, tol float64) ([]math.Point3, bool) {
	pts := make([]math.Point3, 0, farEndTrimStations+1)
	for i := 0; i <= farEndTrimStations; i++ {
		q, ok := stationLanding(arc, t0+(t1-t0)*float64(i)/float64(farEndTrimStations), s, slide)
		if !ok {
			return nil, false
		}
		if boxSideOfPoint(s, box, q, tol) != sideInside {
			return nil, false
		}
		pts = append(pts, q)
	}
	return pts, true
}

// pieceSeg carries a landed station list as ONE boundary segment: the exact circular arc through its
// ends and midpoint when every station lies on that arc's own circle (which is what a stop plane
// perpendicular to the slide axis produces — the section arc, translated), otherwise the cubic B-spline
// fitted through the stations, as the single-face trim already does.
func pieceSeg(pts []math.Point3, tol float64) (endSeg, bool) {
	mid := pts[len(pts)/2]
	if a, err := geom.Arc3dByThreePoints(pts[0], mid, pts[len(pts)-1]); err == nil && pointsOnArcCircle(a, pts, tol) {
		return endSeg{from: pts[0], to: pts[len(pts)-1], curve: a, mid: mid, arc: true}, true
	}
	curve, err := geom.NewFittedBSplineCurve(pts)
	if err != nil {
		return endSeg{}, false
	}
	return endSeg{from: pts[0], to: pts[len(pts)-1], curve: curve, mid: mid}, true
}

// pointsOnArcCircle reports whether every point lies on the arc's own circle within tol.
func pointsOnArcCircle(a geom.Arc3d, pts []math.Point3, tol float64) bool {
	for _, p := range pts {
		if float64(projectOntoArcCircle(a, p).DistanceTo(p)) > tol {
			return false
		}
	}
	return true
}
